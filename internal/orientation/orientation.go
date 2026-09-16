package orientation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Zokiio/context/internal/recordread"
)

// Orient returns partial facts for unavailable records. Operation errors mean
// the configured scope could not be used or evaluation was cancelled.
func Orient(ctx context.Context, request Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if request.ProjectDir == "" {
		return Result{}, errors.New("project is required")
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
	reader, err := recordread.NewReader(request.ProjectDir, request.AllowedSourceDirs)
	if err != nil {
		return Result{}, err
	}
	defer reader.Close()
	e := evaluator{ctx: ctx, request: request, reader: reader, result: emptyResult(), captured: map[string]int{}, records: map[string]*record{}, omitted: map[string]bool{}, inspected: map[string]bool{}}
	manifest := e.read(filepath.Join(reader.Project(), "project.md"), recordread.RecordSource, Reason{Kind: "project"})
	if manifest != nil {
		e.manifest = e.parse(*manifest)
		if e.manifest != nil && e.manifest.kind != "Project" && e.manifest.kind != "WorkItem" {
			e.manifest.sections = recordread.Sections(e.manifest.doc.Body, manifest.Path, sectionKinds)
		}
	}
	e.inventory()
	e.selectSources()
	e.present()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return e.result, nil
}

func emptyResult() Result {
	return Result{SchemaVersion: 1, Complete: true, InventoryComplete: true, Goals: []Goal{}, CurrentCommitments: []Reference{}, WorkItems: []WorkItem{}, Shortlist: []Reference{}, InProgress: []Reference{}, Backlog: []Reference{}, Decisions: []Decision{}, Sources: []Source{}, Diagnostics: []Diagnostic{}}
}

type evaluator struct {
	ctx         context.Context
	request     Request
	reader      *recordread.Reader
	result      Result
	captured    map[string]int
	records     map[string]*record
	recordOrder []*record
	manifest    *record
	selections  []*selection
	totalBytes  int64
	stopped     bool
	omitted     map[string]bool
	inspected   map[string]bool
}

type record struct {
	source    recordread.Source
	doc       recordread.Document
	kind      string
	id, title *string
	sections  recordread.SectionsResult
	ambiguous bool
}

type selection struct {
	sectionStart int
	from         string
	link         recordread.Relationship
	reference    Reference
	source       *recordread.Source
	target       *record
}

var sectionKinds = map[string]string{
	"Goals": "goal", "Current commitments": "commitment", "Open decisions": "open_decision",
	"Spec": "spec", "Context": "context", "Blocked by": "blocked_by", "Blocked by decisions": "blocked_by_decision", "Acceptance": "acceptance",
}

func stringField(metadata map[string]any, name string) string {
	value, _ := metadata[name].(string)
	return strings.TrimSpace(value)
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (e *evaluator) diagnose(d Diagnostic) {
	e.result.Diagnostics = append(e.result.Diagnostics, d)
	if d.Severity == "error" {
		e.result.Complete = false
	}
}

func (e *evaluator) read(path string, role recordread.Role, reason Reason) *recordread.Source {
	if e.ctx.Err() != nil {
		return nil
	}
	if e.stopped {
		resolved := path
		if canonical, err := filepath.EvalSymlinks(path); err == nil {
			resolved = canonical
		}
		if _, captured := e.captured[resolved]; !captured {
			if !e.omitted[resolved] {
				e.omitted[resolved] = true
				e.diagnose(Diagnostic{Code: "source_omitted", Severity: "error", Message: "known pending source was not processed after the limit breach; undiscovered records and relationships are not listed", Path: resolved, From: reason.From, Link: reason.Link})
			}
			return nil
		}
	}
	source, diagnostic := e.reader.Read(path, role)
	if source.Path != "" && !e.inspected[source.Path] {
		limit := ""
		if len(e.inspected) >= e.request.MaxFiles {
			limit = fmt.Sprintf("file limit of %d", e.request.MaxFiles)
		}
		if int64(len(source.Text)) > e.request.MaxBytes-e.totalBytes {
			limit = fmt.Sprintf("source byte limit of %d", e.request.MaxBytes)
		}
		if limit != "" {
			e.stopped = true
			e.omitted[source.Path] = true
			e.diagnose(Diagnostic{Code: "source_limit_exceeded", Severity: "error", Message: "source would exceed " + limit + "; collection stopped before including it", Path: source.Path, From: reason.From, Link: reason.Link})
			return nil
		}
		e.inspected[source.Path] = true
		e.totalBytes += int64(len(source.Text))
	}
	if diagnostic != nil {
		d := Diagnostic(*diagnostic)
		d.From, d.Link = reason.From, reason.Link
		e.diagnose(d)
		return nil
	}
	if index, exists := e.captured[source.Path]; exists {
		for _, existing := range e.result.Sources[index].Reasons {
			if existing == reason {
				return &source
			}
		}
		e.result.Sources[index].Reasons = append(e.result.Sources[index].Reasons, reason)
	} else {
		e.captured[source.Path] = len(e.result.Sources)
		e.result.Sources = append(e.result.Sources, Source{Path: source.Path, SHA256: source.SHA256, Reasons: []Reason{reason}})
	}
	return &source
}

func (e *evaluator) parse(source recordread.Source) *record {
	if existing, ok := e.records[source.Path]; ok {
		return existing
	}
	doc, err := recordread.ParseDocument([]byte(source.Text))
	if err != nil {
		e.records[source.Path] = nil
		e.result.InventoryComplete = false
		e.diagnose(Diagnostic{Code: "invalid_frontmatter", Severity: "error", Message: err.Error(), Path: source.Path})
		return nil
	}
	r := &record{source: source, doc: doc, kind: stringField(doc.Metadata, "type"), id: stringPointer(stringField(doc.Metadata, "id")), title: stringPointer(stringField(doc.Metadata, "title"))}
	e.records[source.Path] = r
	e.recordOrder = append(e.recordOrder, r)
	if r.kind == "Project" || r.kind == "WorkItem" {
		r.sections = recordread.Sections(doc.Body, source.Path, sectionKinds)
		for _, diagnostic := range r.sections.Diagnostics {
			e.diagnose(Diagnostic(diagnostic))
		}
	}
	return r
}
