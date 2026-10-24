// Package factoryrecords publishes an already admitted acceptance plan.
// It coordinates cooperating writers and never replaces immutable evidence.
package factoryrecords

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/rogpeppe/go-internal/lockedfile"
)

var ErrConflict = errors.New("acceptance publication conflicts with current files")

// Plan contains exact bytes retained before any publication side effect. Paths
// may be absolute; relative record paths resolve against BundlePath.
type Plan struct {
	BundlePath         string   `json:"bundlePath"`
	AllowedSourceRoots []string `json:"allowedSourceRoots"`
	ProjectID          string   `json:"projectId"`
	WorkItemID         string   `json:"workItemId"`
	TicketPath         string   `json:"ticketPath"`
	ExpectedTicket     []byte   `json:"expectedTicket"`
	AcceptancePath     string   `json:"acceptancePath"`
	AcceptanceBytes    []byte   `json:"acceptanceBytes"`
	EvidencePath       string   `json:"evidencePath"`
	EvidenceBytes      []byte   `json:"evidenceBytes"`
	LockPath           string   `json:"lockPath,omitempty"`
}

type Result struct {
	TicketPath          string `json:"ticketPath"`
	TicketContentSHA256 string `json:"ticketContentSHA256"`
	AcceptancePath      string `json:"acceptancePath"`
	AcceptanceSHA256    string `json:"acceptanceSHA256"`
	EvidencePath        string `json:"evidencePath"`
	EvidenceSHA256      string `json:"evidenceSHA256"`
}

// Publish writes complete immutable files first, then compares and replaces the
// ticket. A matching partially published plan is safe to resume.
func Publish(ctx context.Context, p Plan) (Result, error) {
	var result Result
	bundle, err := discovery.CanonicalPath(p.BundlePath)
	if err != nil {
		return result, err
	}
	ticket, err := recordPath(bundle, p.TicketPath)
	if err != nil {
		return result, err
	}
	acceptance, err := recordPath(bundle, p.AcceptancePath)
	if err != nil {
		return result, err
	}
	evidence, err := sourcePath(p.EvidencePath, p.AllowedSourceRoots)
	if err != nil {
		return result, err
	}
	if ticket == acceptance || evidence == ticket || evidence == acceptance || within(bundle, evidence) {
		return result, errors.New("evidence must be outside the bundle and all publication paths must differ")
	}
	if len(p.EvidenceBytes) == 0 || !utf8.Valid(p.EvidenceBytes) || !utf8.Valid(p.ExpectedTicket) || !utf8.Valid(p.AcceptanceBytes) {
		return result, errors.New("publication records and nonempty evidence must be UTF-8")
	}
	if err := identities(p, bundle); err != nil {
		return result, err
	}
	updated, err := completedTicket(p.ExpectedTicket, ticket, acceptance)
	if err != nil {
		return result, err
	}
	lockPath := p.LockPath
	if lockPath == "" {
		lockPath = ticket + ".factory.lock"
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0700); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	unlock, err := lockedfile.MutexAt(lockPath).Lock()
	if err != nil {
		return result, err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	current, err := os.ReadFile(ticket)
	if err != nil {
		return result, err
	}
	if !bytes.Equal(current, p.ExpectedTicket) && !bytes.Equal(current, updated) {
		return result, fmt.Errorf("%w: ticket changed", ErrConflict)
	}
	if err := immutable(evidence, p.EvidenceBytes); err != nil {
		return result, err
	}
	if err := immutable(acceptance, p.AcceptanceBytes); err != nil {
		return result, err
	}
	if !bytes.Equal(current, updated) {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := replaceExpected(ticket, p.ExpectedTicket, updated); err != nil {
			return result, err
		}
	}
	return Result{TicketPath: ticket, TicketContentSHA256: digest(updated), AcceptancePath: acceptance, AcceptanceSHA256: digest(p.AcceptanceBytes), EvidencePath: evidence, EvidenceSHA256: digest(p.EvidenceBytes)}, nil
}

// Reconcile uses the exact persisted plan. Different existing bytes are a
// conflict, rather than permission to overwrite an earlier observation.
func Reconcile(ctx context.Context, p Plan) (Result, error) { return Publish(ctx, p) }

func identities(p Plan, bundle string) error {
	manifest, err := os.ReadFile(filepath.Join(bundle, "project.md"))
	if err != nil {
		return err
	}
	project, err := recordread.ParseDocument(manifest)
	if err != nil {
		return err
	}
	ticket, err := recordread.ParseDocument(p.ExpectedTicket)
	if err != nil {
		return err
	}
	acceptance, err := recordread.ParseDocument(p.AcceptanceBytes)
	if err != nil {
		return err
	}
	field := func(m map[string]any, key string) string { s, _ := m[key].(string); return strings.TrimSpace(s) }
	if p.ProjectID == "" || p.WorkItemID == "" || field(project.Metadata, "type") != "Project" || field(project.Metadata, "id") != p.ProjectID || field(ticket.Metadata, "type") != "WorkItem" || field(ticket.Metadata, "id") != p.WorkItemID || field(acceptance.Metadata, "type") != "Acceptance" || field(acceptance.Metadata, "projectId") != p.ProjectID || field(acceptance.Metadata, "workItemId") != p.WorkItemID {
		return errors.New("publication identities do not match project, work item and acceptance")
	}
	return nil
}

func completedTicket(source []byte, ticketPath, acceptancePath string) ([]byte, error) {
	doc, err := recordread.ParseDocument(source)
	if err != nil {
		return nil, err
	}
	sections := recordread.Sections(doc.Body, ticketPath, map[string]string{"Acceptance": "acceptance"})
	if len(sections.Sections) != 1 || sections.InvalidSections["Acceptance"] {
		return nil, errors.New("prepare one valid current Acceptance section before admission")
	}
	section := sections.Sections[0]
	rel, err := filepath.Rel(filepath.Dir(ticketPath), acceptancePath)
	if err != nil {
		return nil, err
	}
	destination := (&url.URL{Path: filepath.ToSlash(rel)}).EscapedPath()
	newline := "\n"
	if bytes.Contains(doc.Body, []byte("\r\n")) {
		newline = "\r\n"
	}
	content := newline + "- [Factory acceptance](" + destination + ")" + newline + newline
	body := append([]byte{}, doc.Body[:section.ContentStart]...)
	body = append(body, []byte(content)...)
	body = append(body, doc.Body[section.End:]...)
	if len(section.Links) > 0 {
		same := len(section.Links) == 1 && section.Links[0].Destination == destination
		if !same {
			history := "Previous acceptance:" + newline + strings.TrimSpace(section.Content) + newline + newline
			comments := recordread.Sections(body, ticketPath, map[string]string{"Comments": "comments"})
			if len(comments.Sections) > 0 {
				at := comments.Sections[0].End
				next := append([]byte{}, body[:at]...)
				next = append(next, []byte(history)...)
				body = append(next, body[at:]...)
			} else {
				if len(body) > 0 && body[len(body)-1] != '\n' {
					return nil, errors.New("prepare acceptance history before admission for a ticket without a terminal newline")
				}
				body = append(body, []byte("## Comments"+newline+newline+history)...)
			}
		}
	}
	metadataStart := bytes.IndexByte(source, '\n') + 1
	if metadataStart <= 0 {
		return nil, errors.New("work item requires YAML frontmatter")
	}
	metadataEnd := metadataStart
	for metadataEnd < len(source) {
		line, _, _ := bytes.Cut(source[metadataEnd:], []byte("\n"))
		if string(bytes.TrimSuffix(line, []byte("\r"))) == "---" {
			break
		}
		metadataEnd += len(line) + 1
	}
	if metadataEnd >= len(source) {
		return nil, errors.New("unterminated work-item frontmatter")
	}
	file, err := parser.ParseBytes(source[metadataStart:metadataEnd], parser.ParseComments)
	if err != nil {
		return nil, err
	}
	path, err := yaml.PathString("$.execution")
	if err != nil {
		return nil, err
	}
	node, err := path.FilterFile(file)
	if err != nil {
		return nil, fmt.Errorf("update authored execution field: %w", err)
	}
	value, ok := node.(*ast.StringNode)
	if !ok {
		return nil, errors.New("work-item execution must be a string scalar")
	}
	// Change the scalar in place so authored comments attached to it survive.
	value.Value = "completed"
	header := "---" + newline + strings.TrimRight(file.String(), "\r\n") + newline + "---" + newline
	return append([]byte(header), body...), nil
}

func recordPath(bundle, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(bundle, path)
	}
	resolved, err := discovery.CanonicalPath(path)
	if err != nil {
		return "", err
	}
	if !within(bundle, resolved) || filepath.Ext(resolved) != ".md" {
		return "", errors.New("record path must be a Markdown file inside the bundle")
	}
	return resolved, nil
}

func sourcePath(path string, roots []string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("evidence path must be absolute")
	}
	resolved, err := discovery.CanonicalPath(path)
	if err != nil {
		return "", err
	}
	for _, root := range roots {
		canonical, err := discovery.CanonicalPath(root)
		if err != nil {
			return "", err
		}
		if within(canonical, resolved) {
			return resolved, nil
		}
	}
	return "", errors.New("evidence path is outside allowed source roots")
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func immutable(path string, data []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if !bytes.Equal(existing, data) {
			return fmt.Errorf("%w: immutable destination differs", ErrConflict)
		}
		return syncDirectory(filepath.Dir(path))
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".factory-publish-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = writeFile(file, data, 0600); err != nil {
		return err
	}
	if err = os.Link(file.Name(), path); errors.Is(err, os.ErrExist) {
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(existing, data) {
			return fmt.Errorf("%w: immutable destination differs", ErrConflict)
		}
	} else if err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func replaceExpected(path string, before, after []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".factory-ticket-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = writeFile(file, after, info.Mode().Perm()); err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, before) {
		return fmt.Errorf("%w: ticket changed before replacement", ErrConflict)
	}
	if err = os.Rename(file.Name(), path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func writeFile(file *os.File, data []byte, mode os.FileMode) error {
	err := file.Chmod(mode)
	if err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	return err
}

func syncDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
