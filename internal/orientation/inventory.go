package orientation

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Zokiio/context/internal/recordread"
)

func (e *evaluator) inventory() {
	if e.stopped {
		e.result.InventoryComplete = false
		return
	}
	paths := []string{}
	err := filepath.WalkDir(e.reader.Project(), func(path string, entry fs.DirEntry, walkErr error) error {
		if e.ctx.Err() != nil {
			return e.ctx.Err()
		}
		if walkErr != nil {
			e.result.InventoryComplete = false
			e.diagnose(Diagnostic{Code: "source_unreadable", Severity: "error", Message: walkErr.Error(), Path: path})
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if target, err := os.Stat(path); err == nil && target.IsDir() {
				return nil
			}
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" || path == filepath.Join(e.reader.Project(), "project.md") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil && e.ctx.Err() == nil {
		e.result.InventoryComplete = false
		e.diagnose(Diagnostic{Code: "source_unreadable", Severity: "error", Message: err.Error(), Path: e.reader.Project()})
	}
	sort.Strings(paths)
	for _, path := range paths {
		relative, _ := filepath.Rel(e.reader.Project(), path)
		source := e.read(path, recordread.RecordSource, Reason{Kind: "inventory", Link: filepath.ToSlash(relative)})
		if source == nil {
			e.result.InventoryComplete = false
			continue
		}
		e.parse(*source)
	}
}

func (e *evaluator) selectSources() {
	for index := 0; index < len(e.recordOrder); index++ {
		r := e.recordOrder[index]
		if r != e.manifest && r.kind != "WorkItem" {
			continue
		}
		for _, section := range r.sections.Sections {
			for _, link := range section.Links {
				e.selectSource(r, section.Start, link)
			}
		}
	}
}

func (e *evaluator) selectSource(r *record, sectionStart int, link recordread.Relationship) {
	if r == e.manifest && link.Kind != "goal" && link.Kind != "commitment" && link.Kind != "open_decision" {
		return
	}
	if r != e.manifest && link.Kind != "spec" && link.Kind != "context" && link.Kind != "blocked_by" && link.Kind != "blocked_by_decision" && link.Kind != "acceptance" {
		return
	}
	s := &selection{from: r.source.Path, sectionStart: sectionStart, link: link, reference: Reference{From: r.source.Path, Link: link.Link}}
	e.selections = append(e.selections, s)
	path, diagnostic := e.reader.LinkPath(r.source.Path, link)
	s.reference.Path = path
	if diagnostic != nil {
		e.diagnose(Diagnostic(*diagnostic))
		return
	}
	role := recordread.RecordSource
	if link.Kind == "goal" || link.Kind == "spec" || link.Kind == "context" {
		role = recordread.DocumentSource
	}
	s.source = e.read(path, role, Reason{Kind: link.Kind, From: r.source.Path, Link: link.Link})
	if s.source == nil {
		return
	}
	s.reference.Path = s.source.Path
	s.target = e.records[s.source.Path]
	if s.target == nil && role == recordread.RecordSource {
		s.target = e.parse(*s.source)
	}
	if s.target != nil {
		s.reference.ID, s.reference.Title = s.target.id, s.target.title
	}
}

func (e *evaluator) present() {
	e.checkIdentities()
	e.presentManifest()
	records := append([]*record{}, e.recordOrder...)
	sort.SliceStable(records, func(i, j int) bool {
		left, right := stringValue(records[i].id), stringValue(records[j].id)
		if left != right {
			return left < right
		}
		return records[i].source.Path < records[j].source.Path
	})
	for _, r := range records {
		switch r.kind {
		case "WorkItem":
			e.presentWorkItem(r)
		case "Decision":
			e.presentDecision(r)
		}
	}
	e.presentShortlist()
}

func (e *evaluator) presentManifest() {
	r := e.manifest
	if r == nil {
		return
	}
	if r.kind != "Project" || r.id == nil || r.title == nil || r.ambiguous {
		e.diagnose(Diagnostic{Code: "invalid_profile", Severity: "error", Message: "project manifest requires type Project, id and title", Path: r.source.Path})
	} else {
		e.result.Project = &Project{ID: *r.id, Title: *r.title, Source: r.source.Path, Metadata: metadataForOutput(r.doc.Metadata)}
	}
	goalsKnown := e.declaration(r, "Goals", false)
	commitmentsKnown := e.declaration(r, "Current commitments", true)
	decisionsKnown := e.declaration(r, "Open decisions", true)
	if e.result.Project != nil {
		e.result.Project.GoalsKnown = goalsKnown
		e.result.Project.CommitmentsKnown = commitmentsKnown
		e.result.Project.OpenDecisionsKnown = decisionsKnown
	}
	for _, section := range r.sections.Sections {
		switch section.Name {
		case "Goals":
			if content := strings.TrimSpace(section.Content); content != "" {
				references := []Reference{}
				for _, s := range e.selections {
					if s.from == r.source.Path && s.link.Kind == "goal" && s.sectionStart == section.Start {
						references = append(references, s.reference)
					}
				}
				e.result.Goals = append(e.result.Goals, Goal{Text: content, Source: r.source.Path, References: references})
			}
		}
	}
	for _, s := range e.selections {
		if s.from == r.source.Path && s.link.Kind == "commitment" {
			e.result.CurrentCommitments = append(e.result.CurrentCommitments, s.reference)
		}
	}
}

func (e *evaluator) presentWorkItem(r *record) {
	work := WorkItem{ID: r.id, Title: r.title, Source: r.source.Path, Lifecycle: stringPointer(stringField(r.doc.Metadata, "status")), Specifications: []Reference{}, Checks: []Check{}, Readiness: "unknown", ExclusionReasons: []Finding{}, Metadata: metadataForOutput(r.doc.Metadata)}
	setWorkItemFingerprints(&work, r.doc.Body)
	if _, authored := r.doc.Metadata["status"]; !authored {
		work.Lifecycle = stringPointer("stable")
	}
	work.IdentityAmbiguous = r.ambiguous
	work.Triage = e.enumField(r, "triage", "needs-triage", "needs-info", "ready-for-agent", "ready-for-human", "wontfix")
	work.Execution = e.enumField(r, "execution", "unstarted", "in-progress", "completed", "cancelled")
	committed := false
	if e.result.Project != nil && e.result.Project.CommitmentsKnown && !r.ambiguous {
		work.Committed = &committed
	}
	for _, s := range e.selections {
		if s.link.Kind == "commitment" && s.target == r && !r.ambiguous {
			yes := true
			work.Committed = &yes
		}
		if s.link.Kind == "spec" && s.from == r.source.Path {
			work.Specifications = append(work.Specifications, s.reference)
		}
	}
	e.evaluateReadiness(r, &work)
	e.evaluateEligibility(&work)
	ref := Reference{Path: r.source.Path, ID: r.id, Title: r.title}
	if stringValue(work.Execution) == "in-progress" {
		e.result.InProgress = append(e.result.InProgress, ref)
	}
	if stringValue(work.Execution) != "completed" && stringValue(work.Execution) != "cancelled" && stringValue(work.Execution) != "in-progress" && work.Committed != nil && !*work.Committed {
		e.result.Backlog = append(e.result.Backlog, ref)
	}
	e.result.WorkItems = append(e.result.WorkItems, work)
}

func (e *evaluator) presentDecision(r *record) {
	decision := Decision{ID: r.id, Title: r.title, Source: r.source.Path, State: stringPointer(stringField(r.doc.Metadata, "decisionState")), References: []Reference{}, Metadata: metadataForOutput(r.doc.Metadata)}
	decision.IdentityAmbiguous = r.ambiguous
	for _, s := range e.selections {
		if s.target == r && (s.link.Kind == "open_decision" || s.link.Kind == "blocked_by_decision") {
			decision.References = append(decision.References, s.reference)
		}
	}
	e.result.Decisions = append(e.result.Decisions, decision)
	e.diagnose(Diagnostic{Code: "unsupported_check", Severity: "error", Message: "decision evaluation is not implemented yet", Path: r.source.Path})
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
