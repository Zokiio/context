package orientation

import (
	"regexp"
	"strings"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

func (e *evaluator) evaluateReadiness(r *record, work *WorkItem) {
	work.Checks = []Check{e.criteriaCheck(r), e.contextCheck(r), e.emptyRelationshipCheck(r, "Blocked by", "dependencies", "no_dependencies"), e.blockingDecisionCheck(r)}
	work.Readiness = "ready"
	for _, check := range work.Checks {
		if check.Status == "fail" {
			work.Readiness = "blocked"
		}
		if check.Status == "unknown" && work.Readiness != "blocked" {
			work.Readiness = "unknown"
		}
	}
}

func (e *evaluator) criteriaCheck(r *record) Check {
	present, nonempty := false, false
	sections := recordread.Sections(r.doc.Body, r.source.Path, map[string]string{"Acceptance criteria": "acceptance_criteria"})
	for _, section := range sections.Sections {
		present = true
		if hasAuthoredContent([]byte(section.Content)) {
			nonempty = true
		}
	}
	if !present {
		return e.check(r, "acceptance_criteria", "unknown", "missing_required_section", "missing required section: Acceptance criteria")
	}
	if !nonempty {
		return e.check(r, "acceptance_criteria", "fail", "empty_acceptance_criteria", "Acceptance criteria contains no nonempty criterion")
	}
	return e.check(r, "acceptance_criteria", "pass", "acceptance_criteria_present", "at least one nonempty acceptance criterion is authored")
}

func (e *evaluator) contextCheck(r *record) Check {
	check := Check{Name: "required_context", Status: "pass", Reasons: []Finding{}}
	if r.id == nil || r.title == nil || r.ambiguous {
		code := "invalid_profile"
		message := "work identity and title must be available"
		if r.ambiguous {
			code = "duplicate_identity"
			message = "work identity is ambiguous"
		}
		check.Status = "unknown"
		check.Reasons = append(check.Reasons, Finding{Code: code, Message: message, Path: r.source.Path})
	}
	for _, s := range e.selections {
		if s.from != r.source.Path || (s.link.Kind != "spec" && s.link.Kind != "context") {
			continue
		}
		if s.source == nil {
			check.Status = "unknown"
			check.Reasons = append(check.Reasons, Finding{Code: "required_context_unavailable", Message: "selected requirement or context source is unavailable", Path: s.reference.Path, From: s.from, Link: s.link.Link})
			continue
		}
		if _, err := recordread.ParseDocument([]byte(s.source.Text)); err != nil {
			check.Status = "unknown"
			reason := Finding{Code: "invalid_frontmatter", Message: err.Error(), Path: s.source.Path, From: s.from, Link: s.link.Link}
			check.Reasons = append(check.Reasons, reason)
			e.diagnose(Diagnostic{Code: reason.Code, Severity: "error", Message: reason.Message, Path: reason.Path, From: reason.From, Link: reason.Link})
		}
	}
	for _, name := range []string{"Spec", "Context"} {
		for _, section := range r.sections.Sections {
			if section.Name != name {
				continue
			}
			if content := strings.TrimSpace(section.Text); len(section.Links) == 0 && content != "" && content != "None" {
				check.Status = "unknown"
				reason := Finding{Code: "invalid_relationship_section", Message: name + " must contain file links, be empty, or contain the literal None", Path: r.source.Path}
				check.Reasons = append(check.Reasons, reason)
				e.diagnose(Diagnostic{Code: reason.Code, Severity: "error", Message: reason.Message, Path: reason.Path})
			}
		}
		if r.sections.InvalidSections[name] {
			check.Status = "unknown"
			check.Reasons = append(check.Reasons, Finding{Code: "unresolved_reference", Message: "unresolved Markdown reference in " + name, Path: r.source.Path})
		}
	}
	if check.Status == "pass" {
		check.Reasons = append(check.Reasons, Finding{Code: "required_context_available", Message: "all explicitly selected requirement and context sources are available", Path: r.source.Path})
	}
	return check
}

var emptyCheckbox = regexp.MustCompile(`\[[ xX]\]`)

func hasAuthoredContent(body []byte) bool {
	var content strings.Builder
	codeContent := false
	tree := parser.New().Parse(body)
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.Heading:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			content.WriteString(node.Value.Value(body))
		case *ast.CodeSpan:
			codeContent = codeContent || strings.TrimSpace(node.Value.Value(body)) != ""
		case *ast.CodeBlock:
			codeContent = codeContent || strings.TrimSpace(node.Value.Str(body)) != ""
		case *ast.AutoLink:
			content.WriteString(node.Destination.Value(body))
		}
		return ast.WalkContinue, nil
	})
	return codeContent || strings.TrimSpace(emptyCheckbox.ReplaceAllString(content.String(), "")) != ""
}

func (e *evaluator) emptyRelationshipCheck(r *record, sectionName, checkName, noneCode string) Check {
	if !e.declaration(r, sectionName, true) {
		return e.check(r, checkName, "unknown", "incomplete_relationship_declaration", sectionName+" declaration is missing, malformed, or unavailable")
	}
	for _, section := range r.sections.Sections {
		if section.Name == sectionName && len(section.Links) > 0 {
			return e.check(r, checkName, "unknown", "unsupported_check", sectionName+" relationship evaluation is not implemented yet")
		}
	}
	return e.check(r, checkName, "pass", noneCode, sectionName+" explicitly declares none")
}

func (e *evaluator) check(r *record, name, status, code, message string) Check {
	if status == "unknown" {
		e.diagnose(Diagnostic{Code: code, Severity: "error", Message: message, Path: r.source.Path})
	}
	return Check{Name: name, Status: status, Reasons: []Finding{{Code: code, Message: message, Path: r.source.Path}}}
}

func (e *evaluator) evaluateEligibility(work *WorkItem) {
	exclude := func(code, message string) {
		work.ExclusionReasons = append(work.ExclusionReasons, Finding{Code: code, Message: message, Path: work.Source})
	}
	if work.ID == nil || work.Title == nil {
		exclude("invalid_identity", "work identity and title must be available")
	}
	if work.IdentityAmbiguous {
		exclude("duplicate_identity", "record identity is ambiguous")
	}
	if e.result.Project == nil {
		exclude("project_identity_unknown", "shortlist is suppressed because project identity is unavailable")
	}
	if !e.result.InventoryComplete {
		exclude("inventory_incomplete", "shortlist is suppressed because project inventory is incomplete")
	}
	switch stringValue(work.Execution) {
	case "unstarted":
	case "in-progress":
		exclude("already_in_progress", "work is already in progress")
	case "completed":
		exclude("completed_work", "work is recorded as completed")
	case "cancelled":
		exclude("cancelled_work", "work is recorded as cancelled")
	default:
		exclude("execution_unknown", "execution state is unknown")
	}
	if work.Committed == nil {
		exclude("commitment_unknown", "current commitment membership is unknown")
	} else if !*work.Committed {
		exclude("not_committed", "work is outside current commitments")
	}
	if work.Triage == nil {
		exclude("triage_unknown", "triage role is unknown")
	} else if *work.Triage != "ready-for-agent" {
		exclude("triage_not_ready", "triage is not ready-for-agent")
	}
	if work.Readiness != "ready" {
		exclude("readiness_"+work.Readiness, "readiness is "+work.Readiness+"; inspect its checks")
	}
	work.Eligible = len(work.ExclusionReasons) == 0
}

func (e *evaluator) presentShortlist() {
	eligible := map[string]WorkItem{}
	for _, work := range e.result.WorkItems {
		if work.Eligible {
			eligible[work.Source] = work
		}
	}
	for _, reference := range e.result.CurrentCommitments {
		if work, exists := eligible[reference.Path]; exists {
			e.result.Shortlist = append(e.result.Shortlist, Reference{Path: work.Source, ID: work.ID, Title: work.Title, From: reference.From, Link: reference.Link})
			delete(eligible, reference.Path)
		}
	}
}
