package orientation

import (
	"sort"
	"strings"

	"github.com/Zokiio/context/internal/recordread"
)

func (e *evaluator) evaluateDecision(r *record) Decision {
	if decision, exists := e.decisionResults[r.source.Path]; exists {
		return decision
	}
	decision := Decision{ID: r.id, Title: r.title, Source: r.source.Path, IdentityAmbiguous: r.ambiguous, State: e.enumField(r, "decisionState", "open", "resolved"), CheckStatus: "pass", Reasons: []Finding{}, References: []Reference{}, AffectedWork: []Reference{}, Metadata: MetadataForOutput(r.doc.Metadata)}
	unknown := func(code, message string) {
		decision.CheckStatus = "unknown"
		decision.Reasons = append(decision.Reasons, Finding{Code: code, Message: message, Path: r.source.Path})
		e.diagnose(Diagnostic{Code: code, Severity: "error", Message: message, Path: r.source.Path})
	}
	if r.id == nil || r.title == nil {
		unknown("invalid_profile", "decision requires a nonempty identity and title")
	}
	if r.ambiguous {
		unknown("duplicate_identity", "decision identity is ambiguous")
	}
	if decision.State == nil {
		unknown("invalid_profile", "decisionState must be open or resolved")
	}
	parts := []string{}
	sections := recordread.Sections(r.doc.Body, r.source.Path, map[string]string{"Resolution": "resolution"})
	for _, section := range sections.Sections {
		if hasAuthoredContent([]byte(section.Content)) {
			parts = append(parts, strings.TrimSpace(section.Content))
		}
	}
	if len(parts) > 0 {
		decision.Resolution = stringPointer(strings.Join(parts, "\n\n"))
	}
	if stringValue(decision.State) == "resolved" && decision.Resolution == nil {
		unknown("missing_resolution", "resolved decision requires a nonempty Resolution section")
	}
	if decision.CheckStatus != "unknown" {
		code, message := "decision_resolved", "decision has an authored resolution"
		if stringValue(decision.State) == "open" {
			decision.CheckStatus = "fail"
			code = "open_decision"
			message = "decision remains open"
		}
		decision.Reasons = append(decision.Reasons, Finding{Code: code, Message: message, Path: r.source.Path})
	}
	seenReferences, seenWork := map[string]bool{}, map[string]bool{}
	for _, s := range e.selections {
		if s.target != r || (s.link.Kind != "open_decision" && s.link.Kind != "blocked_by_decision") {
			continue
		}
		key := s.from + "\x00" + s.link.Link
		if !seenReferences[key] {
			decision.References = append(decision.References, s.reference)
			seenReferences[key] = true
		}
		if s.link.Kind == "blocked_by_decision" && !seenWork[s.from] {
			if work := e.records[s.from]; work != nil {
				decision.AffectedWork = append(decision.AffectedWork, Reference{Path: s.from, ID: work.id, Title: work.title})
				seenWork[s.from] = true
			}
		}
	}
	sort.SliceStable(decision.AffectedWork, func(i, j int) bool {
		left, right := decision.AffectedWork[i], decision.AffectedWork[j]
		if stringValue(left.ID) != stringValue(right.ID) {
			return stringValue(left.ID) < stringValue(right.ID)
		}
		return left.Path < right.Path
	})
	e.decisionResults[r.source.Path] = decision
	return decision
}

func (e *evaluator) blockingDecisionCheck(r *record) Check {
	check := Check{Name: "blocking_decisions", Status: "pass", Reasons: []Finding{}}
	appendReasons := func(status string, reasons ...Finding) {
		mergeChecks(&check, Check{Status: status, Reasons: reasons})
	}
	if !e.declaration(r, "Blocked by decisions", true) {
		appendReasons("unknown", Finding{Code: "incomplete_relationship_declaration", Message: "Blocked by decisions declaration is missing, malformed, or unavailable", Path: r.source.Path})
	}
	for _, s := range e.selections {
		if s.from != r.source.Path || s.link.Kind != "blocked_by_decision" {
			continue
		}
		if s.target == nil || s.source == nil || s.target.kind != "Decision" {
			code, message := "decision_unavailable", "blocking decision source is unavailable"
			if s.source != nil {
				code = "invalid_profile"
				message = "blocking relationship must identify a Decision record"
			}
			appendReasons("unknown", Finding{Code: code, Message: message, Path: s.reference.Path, From: s.from, Link: s.link.Link})
			continue
		}
		decision := e.evaluateDecision(s.target)
		for _, reason := range decision.Reasons {
			reason.From, reason.Link = s.from, s.link.Link
			appendReasons(decision.CheckStatus, reason)
		}
	}
	if len(check.Reasons) == 0 {
		check.Reasons = append(check.Reasons, Finding{Code: "no_blocking_decisions", Message: "Blocked by decisions explicitly declares none", Path: r.source.Path})
	}
	return check
}

func (e *evaluator) presentDecision(r *record) {
	for _, s := range e.selections {
		if s.target == r && (s.link.Kind == "open_decision" || s.link.Kind == "blocked_by_decision") {
			e.result.Decisions = append(e.result.Decisions, e.evaluateDecision(r))
			return
		}
	}
}
