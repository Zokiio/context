// Package taskcontext assembles current source files for an explicitly selected ticket.
package taskcontext

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type Result struct {
	SchemaVersion     int          `json:"schemaVersion"`
	Complete          bool         `json:"complete"`
	TraversalComplete bool         `json:"traversalComplete"`
	Sources           []Source     `json:"sources"`
	Diagnostics       []Diagnostic `json:"diagnostics"`
}

type Source struct {
	Path    string   `json:"path"`
	Text    string   `json:"text"`
	SHA256  string   `json:"sha256"`
	Reasons []Reason `json:"reasons"`
}

type Reason struct {
	Kind string `json:"kind"`
	From string `json:"from,omitempty"`
	Link string `json:"link,omitempty"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path"`
	From     string `json:"from,omitempty"`
	Link     string `json:"link,omitempty"`
}

// Assemble returns incomplete data for unavailable sources and invalid frontmatter.
// Errors mean the operation could not run. It does not print, exit, or write files.
func Assemble(ctx context.Context, request Request) (Result, error) {
	result := Result{SchemaVersion: 1, Complete: true, TraversalComplete: true, Sources: []Source{}, Diagnostics: []Diagnostic{}}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if request.ProjectDir == "" || request.TicketPath == "" {
		return Result{}, errors.New("project and ticket are required")
	}
	if request.MaxFiles < 0 || request.MaxBytes < 0 {
		return Result{}, errors.New("source limits must be positive")
	}
	if request.MaxFiles == 0 {
		request.MaxFiles = DefaultMaxFiles
	}
	if request.MaxBytes == 0 {
		request.MaxBytes = DefaultMaxBytes
	}
	reader, err := newSourceReader(request)
	if err != nil {
		return Result{}, err
	}
	defer reader.close()
	path := request.TicketPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(reader.project, path)
	}

	indices := map[string]int{}
	var totalBytes int64
	include := func(source Source, reason Reason) int {
		if index, exists := indices[source.Path]; exists {
			for _, existing := range result.Sources[index].Reasons {
				if existing == reason {
					return index
				}
			}
			result.Sources[index].Reasons = append(result.Sources[index].Reasons, reason)
			return index
		}
		limit := ""
		if len(result.Sources) >= request.MaxFiles {
			limit = fmt.Sprintf("file limit of %d", request.MaxFiles)
		}
		if int64(len(source.Text)) > request.MaxBytes-totalBytes {
			limit = fmt.Sprintf("source byte limit of %d", request.MaxBytes)
		}
		if limit != "" {
			result.Complete, result.TraversalComplete = false, false
			result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "source_limit_exceeded", Severity: "error", Message: "source would exceed " + limit + "; collection stopped before including it", Path: source.Path, From: reason.From, Link: reason.Link})
			return -1
		}
		totalBytes += int64(len(source.Text))
		source.Reasons = []Reason{reason}
		index := len(result.Sources)
		indices[source.Path] = index
		result.Sources = append(result.Sources, source)
		return index
	}
	unavailable := func(diagnostic Diagnostic, reason Reason, ticket bool) {
		diagnostic.From, diagnostic.Link = reason.From, reason.Link
		result.Diagnostics = append(result.Diagnostics, diagnostic)
		result.Complete = false
		if ticket {
			result.TraversalComplete = false
		}
	}
	type pendingTicket struct {
		path   string
		reason Reason
		link   relationship
	}
	reportPending := func(breached string, pending []pendingTicket) {
		reported := map[string]bool{breached: true}
		for _, item := range pending {
			path := item.path
			if item.reason.Kind != "root" {
				path, _ = reader.linkPath(item.reason.From, item.link)
			}
			if resolved, err := filepath.EvalSymlinks(path); err == nil {
				path = resolved
			}
			if _, included := indices[path]; included {
				continue
			}
			key := path
			if key == "" {
				key = item.reason.From + "\x00" + item.reason.Link
			}
			if reported[key] {
				continue
			}
			reported[key] = true
			result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "source_omitted", Severity: "error", Message: "known pending source was not processed after the limit breach; undiscovered relationships are not listed", Path: path, From: item.reason.From, Link: item.reason.Link})
		}
	}
	queue := []pendingTicket{{path: path, reason: Reason{Kind: "root"}}}
	explored := map[string]bool{}
	blockers := map[string][]string{}
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		pending := queue[0]
		queue = queue[1:]
		path := pending.path
		var diagnostic *Diagnostic
		if pending.reason.Kind != "root" {
			path, diagnostic = reader.linkPath(pending.reason.From, pending.link)
		}
		var ticket Source
		if diagnostic == nil {
			// Enforce ticket scope even if this file was already selected as a document.
			ticket, diagnostic = reader.read(path, ticketSource)
		}
		if diagnostic != nil {
			unavailable(*diagnostic, pending.reason, true)
			continue
		}
		index := include(ticket, pending.reason)
		if index < 0 {
			reportPending(ticket.Path, queue)
			return result, nil
		}
		if pending.reason.Kind == "blocked_by" && addBlockerEdge(blockers, pending.reason.From, ticket.Path) {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "dependency_cycle", Severity: "warning", Message: "blocker relationship forms a dependency cycle", Path: ticket.Path, From: pending.reason.From, Link: pending.reason.Link})
		}
		if explored[ticket.Path] {
			continue
		}
		explored[ticket.Path] = true
		// A document may have been selected earlier. Discover relationships from
		// the retained bytes, which are the bytes described by the returned digest.
		ticket = result.Sources[index]
		doc, err := parseDocument([]byte(ticket.Text))
		if err != nil {
			unavailable(Diagnostic{Code: "invalid_frontmatter", Severity: "error", Message: err.Error(), Path: ticket.Path}, pending.reason, true)
			continue
		}
		links, diagnostics := extractRelationships(doc.body, ticket.Path)
		if len(diagnostics) > 0 {
			result.Complete = false
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
		}
		// Discover every outgoing blocker before reading documents. A document limit
		// breach must still report blockers already known from this ticket's body.
		for _, link := range links {
			if link.kind == "blocked_by" {
				queue = append(queue, pendingTicket{reason: Reason{Kind: link.kind, From: ticket.Path, Link: link.link}, link: link})
			}
		}
		documents := []pendingTicket{}
		for _, kind := range []string{"spec", "context"} {
			for _, link := range links {
				if link.kind == kind {
					documents = append(documents, pendingTicket{reason: Reason{Kind: kind, From: ticket.Path, Link: link.link}, link: link})
				}
			}
		}
		for position, pending := range documents {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			path, diagnostic := reader.linkPath(ticket.Path, pending.link)
			var source Source
			if diagnostic == nil {
				source, diagnostic = reader.read(path, documentSource)
			}
			if diagnostic != nil {
				unavailable(*diagnostic, pending.reason, false)
				continue
			}
			if include(source, pending.reason) < 0 {
				known := append(documents[position+1:], queue...)
				reportPending(source.Path, known)
				return result, nil
			}
		}

	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return result, nil
}

// addBlockerEdge reports whether a new edge closes a cycle. Reaching an already
// explored ticket alone is not a cycle: separate branches often share blockers.
func addBlockerEdge(graph map[string][]string, from, to string) bool {
	for _, existing := range graph[from] {
		if existing == to {
			return false
		}
	}
	pending := []string{to}
	seen := map[string]bool{}
	cycle := false
	for len(pending) > 0 {
		path := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if path == from {
			cycle = true
			break
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		pending = append(pending, graph[path]...)
	}
	graph[from] = append(graph[from], to)
	return cycle
}
