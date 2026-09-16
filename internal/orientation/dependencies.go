package orientation

func (e *evaluator) dependencyCheck(r *record) Check {
	check := Check{Name: "dependencies", Status: "pass", Reasons: []Finding{}}
	appendCheck := func(status string, reasons ...Finding) {
		check.Status = combineCheckStatus(check.Status, status)
		check.Reasons = append(check.Reasons, reasons...)
	}
	if !e.declaration(r, "Blocked by", true) {
		appendCheck("unknown", Finding{Code: "incomplete_relationship_declaration", Message: "Blocked by declaration is missing, malformed, or unavailable", Path: r.source.Path})
	}
	for _, s := range e.selections {
		if s.from != r.source.Path || s.link.Kind != "blocked_by" {
			continue
		}
		prerequisite := s.target
		if s.source == nil || prerequisite == nil || prerequisite.kind != "WorkItem" || prerequisite.id == nil || prerequisite.title == nil || prerequisite.ambiguous {
			appendCheck("unknown", Finding{Code: "dependency_unavailable", Message: "prerequisite must identify an available, unambiguous WorkItem", Path: s.reference.Path, From: s.from, Link: s.link.Link})
			continue
		}
		execution := e.enumField(prerequisite, "execution", "unstarted", "in-progress", "completed", "cancelled")
		switch stringValue(execution) {
		case "unstarted", "in-progress":
			appendCheck("fail", Finding{Code: "unfinished_dependency", Message: "prerequisite is not completed", Path: prerequisite.source.Path, From: s.from, Link: s.link.Link})
		case "cancelled":
			appendCheck("fail", Finding{Code: "cancelled_dependency", Message: "cancelled prerequisite cannot satisfy an edge", Path: prerequisite.source.Path, From: s.from, Link: s.link.Link})
		case "completed":
			accepted := e.evaluateAcceptance(prerequisite)
			appendCheck(accepted.CheckStatus, accepted.Reasons...)
			decisions := e.blockingDecisionCheck(prerequisite)
			appendCheck(decisions.Status, decisions.Reasons...)
			dependencies := e.emptyRelationshipCheck(prerequisite, "Blocked by", "dependencies", "no_dependencies")
			appendCheck(dependencies.Status, dependencies.Reasons...)
		default:
			appendCheck("unknown", Finding{Code: "dependency_execution_unknown", Message: "prerequisite execution state is unavailable", Path: prerequisite.source.Path, From: s.from, Link: s.link.Link})
		}
	}
	if len(check.Reasons) == 0 {
		check.Reasons = append(check.Reasons, Finding{Code: "no_dependencies", Message: "Blocked by explicitly declares none", Path: r.source.Path})
	}
	return check
}
