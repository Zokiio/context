package orientation

import "strings"

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
	value := stringField(r.doc.Metadata, name)
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
		if content := strings.TrimSpace(section.Text); linkOnly && content != "" && content != "None" && len(section.Links) == 0 {
			complete = false
			e.diagnose(Diagnostic{Code: "invalid_relationship_section", Severity: "error", Message: name + " must contain file links, be empty, or contain the literal None", Path: r.source.Path})
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
