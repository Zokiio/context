package orientation

import (
	"context"
	"errors"
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
	reader, err := recordread.NewCapture(request.ProjectDir, request.AllowedSourceDirs, recordread.Limits{MaxFiles: request.MaxFiles, MaxBytes: request.MaxBytes})
	if err != nil {
		return Result{}, err
	}
	defer reader.Close()
	return orientWithCapture(ctx, reader)
}

// OrientWithCapture evaluates a project using source bytes and budget shared
// with other readers in the same caller operation. The caller owns the capture
// and must close it.
func OrientWithCapture(ctx context.Context, capture *recordread.Capture) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if capture == nil {
		return Result{}, errors.New("capture is required")
	}
	return orientWithCapture(ctx, capture)
}

func orientWithCapture(ctx context.Context, reader *recordread.Capture) (Result, error) {
	exhausted := reader.Exhausted()
	e := evaluator{ctx: ctx, reader: reader, result: emptyResult(), captured: map[string]int{}, records: map[string]*record{}, stopped: exhausted, exhaustedBeforeOrientation: exhausted, omitted: map[string]bool{}, decisionResults: map[string]Decision{}, acceptanceSources: map[string]acceptanceSources{}, acceptanceResults: map[string]*AcceptanceSummary{}}
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
	ctx                        context.Context
	reader                     *recordread.Capture
	result                     Result
	captured                   map[string]int
	records                    map[string]*record
	recordOrder                []*record
	manifest                   *record
	selections                 []*selection
	stopped                    bool
	exhaustedBeforeOrientation bool
	omitted                    map[string]bool
	decisionResults            map[string]Decision
	acceptanceSources          map[string]acceptanceSources
	acceptanceResults          map[string]*AcceptanceSummary
	dependencies               map[string]*dependencyNode
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
	source, diagnostic := e.reader.Read(path, role)
	if source.Path != "" {
		if limit := e.reader.Admit(source); limit != nil {
			e.stopped = true
			if e.omitted[source.Path] {
				return nil
			}
			e.omitted[source.Path] = true
			e.diagnose(Diagnostic{Code: "source_limit_exceeded", Severity: "error", Message: "source would exceed " + limit.Error() + "; collection stopped before including it", Path: source.Path, From: reason.From, Link: reason.Link})
			return nil
		}
	}
	if diagnostic != nil {
		d := Diagnostic(*diagnostic)
		d.From, d.Link = reason.From, reason.Link
		if d.Code == "source_omitted" {
			if e.omitted[d.Path] {
				return nil
			}
			e.omitted[d.Path] = true
			d.Message = "known pending source was not processed after the limit breach; undiscovered records and relationships are not listed"
		}
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
