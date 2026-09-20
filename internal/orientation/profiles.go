package orientation

import (
	"net/url"
	"strings"
)

func (e *evaluator) checkIdentities() {
	byID := map[string][]*record{}
	for _, r := range e.recordOrder {
		if r.kind != "" && r.id != nil {
			byID[*r.id] = append(byID[*r.id], r)
		}
		switch r.kind {
		case "Project", "WorkItem", "Decision", "Acceptance":
			if r.id == nil {
				e.invalidField(r, "id")
			}
			if r.title == nil {
				e.invalidField(r, "title")
			}
		}
	}
	// Emit in source order rather than map iteration order.
	for _, r := range e.recordOrder {
		if r.id != nil && len(byID[*r.id]) > 1 {
			r.ambiguous = true
			e.diagnose(Diagnostic{Code: "duplicate_identity", Severity: "error", Message: "record ID is shared by multiple typed records: " + *r.id, Path: r.source.Path})
		}
	}
	for _, s := range e.selections {
		want := ""
		switch s.link.Kind {
		case "commitment", "blocked_by":
			want = "WorkItem"
		case "open_decision", "blocked_by_decision":
			want = "Decision"
		case "acceptance":
			want = "Acceptance"
		}
		if s.source != nil && want != "" && (s.target == nil || s.target.kind != want) {
			e.diagnose(Diagnostic{Code: "invalid_profile", Severity: "error", Message: "relationship requires a " + want + " record", Path: s.source.Path, From: s.from, Link: s.link.Link})
		}
	}
}

func (e *evaluator) invalidField(r *record, field string) {
	e.diagnose(Diagnostic{Code: "invalid_profile", Severity: "error", Message: r.kind + " requires a valid " + field, Path: r.source.Path})
}

func (e *evaluator) enumField(r *record, name string, allowed ...string) *string {
	value, _ := r.doc.Metadata[name].(string)
	for _, valid := range allowed {
		if value == valid {
			return &value
		}
	}
	e.invalidField(r, name)
	return nil
}

func (e *evaluator) declaration(r *record, name string, linkOnly bool) bool {
	present, complete := false, true
	for _, section := range r.sections.Sections {
		if section.Name != name {
			continue
		}
		present = true
		if !section.Complete {
			complete = false
		}
		if content := strings.TrimSpace(section.Text); linkOnly && !emptyDeclaration(content) && len(section.Links) == 0 {
			complete = false
			e.diagnose(Diagnostic{Code: "invalid_relationship_section", Severity: "error", Message: name + " must contain file links, be empty, or contain the literal None or None.", Path: r.source.Path})
		}
	}
	if !present {
		e.diagnose(Diagnostic{Code: "missing_required_section", Severity: "error", Message: "missing required section: " + name, Path: r.source.Path})
		return false
	}
	kind := sectionKinds[name]
	for _, s := range e.selections {
		if s.from != r.source.Path || s.link.Kind != kind {
			continue
		}
		if s.source == nil {
			complete = false
			continue
		}
		if name == "Current commitments" && (s.target == nil || s.target.kind != "WorkItem" || s.target.id == nil || s.target.ambiguous) {
			complete = false
		}
		if name == "Open decisions" && (s.target == nil || s.target.kind != "Decision" || s.target.id == nil || s.target.ambiguous) {
			complete = false
		}
	}
	return complete
}

func emptyDeclaration(content string) bool {
	switch strings.TrimSpace(content) {
	case "", "None", "None.":
		return true
	default:
		return false
	}
}

// executionField keeps an absent upstream state unknown without treating a
// labeled tracker snapshot as a malformed native WorkItem.
func (e *evaluator) executionField(r *record) *string {
	if _, present := r.doc.Metadata["execution"]; !present && isTrackerSnapshot(r) {
		return nil
	}
	return e.enumField(r, "execution", "unstarted", "in-progress", "completed", "cancelled")
}

func isTrackerSnapshot(r *record) bool {
	if r.kind != "WorkItem" {
		return false
	}
	source, ok := r.doc.Metadata["sourceURL"].(string)
	if !ok {
		return false
	}
	parsed, err := url.Parse(source)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
}
