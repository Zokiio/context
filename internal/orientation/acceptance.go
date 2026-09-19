package orientation

import (
	"strings"

	"github.com/Zokiio/context/internal/recordread"
)

func (e *evaluator) evaluateAcceptance(subject *record) *AcceptanceSummary {
	if result, exists := e.acceptanceResults[subject.source.Path]; exists {
		return result
	}
	links := []*selection{}
	for _, s := range e.selections {
		if s.from == subject.source.Path && s.link.Kind == "acceptance" {
			links = append(links, s)
		}
	}
	malformed := subject.sections.InvalidSections["Acceptance"]
	for _, section := range subject.sections.Sections {
		if section.Name == "Acceptance" && len(section.Links) == 0 {
			text := strings.TrimSpace(section.Text)
			malformed = malformed || (text != "" && text != "None")
		}
	}
	if len(links) == 0 && stringField(subject.doc.Metadata, "execution") != "completed" && !malformed {
		e.acceptanceResults[subject.source.Path] = nil
		return nil
	}
	result := &AcceptanceSummary{Status: "valid", CheckStatus: "pass", HumanApprovals: []HumanApproval{}, Requirements: []Snapshot{}, Evidence: []Snapshot{}, Reasons: []Finding{}, Metadata: map[string]any{}}
	e.acceptanceResults[subject.source.Path] = result
	unknown := func(code, message string) {
		e.acceptanceFinding(result, "unknown", Finding{Code: code, Message: message, Path: subject.source.Path})
	}
	if len(links) != 1 {
		unknown("invalid_current_acceptance", "current Acceptance requires exactly one record link")
		return result
	}
	selected := links[0]
	ref := selected.reference
	result.Record = &ref
	if malformed {
		unknown("invalid_current_acceptance", "current Acceptance contains a malformed section or unresolved Markdown reference")
	}
	r := selected.target
	if selected.source == nil || r == nil || r.kind != "Acceptance" {
		unknown("acceptance_unavailable", "current Acceptance must resolve to an available Acceptance record")
		return result
	}
	result.Metadata = MetadataForOutput(r.doc.Metadata)
	if r.id == nil || r.title == nil || r.ambiguous || subject.id == nil || subject.ambiguous || e.result.Project == nil {
		unknown("invalid_acceptance_identity", "Acceptance and its project and work identities must be present and unambiguous")
	}
	if e.result.Project != nil && stringField(r.doc.Metadata, "projectId") != e.result.Project.ID {
		unknown("acceptance_subject_mismatch", "Acceptance projectId does not match the subject project")
	}
	if subject.id == nil || stringField(r.doc.Metadata, "workItemId") != stringValue(subject.id) {
		unknown("acceptance_subject_mismatch", "Acceptance workItemId does not match the subject work")
	}
	e.acceptanceMetadata(r, result)
	current := fingerprintBody(subject.doc.Body)
	if result.FingerprintVersion != nil && *result.FingerprintVersion == 1 {
		if result.TicketSHA256 != nil && *result.TicketSHA256 != current.TicketSHA256 {
			e.acceptanceFinding(result, "fail", Finding{Code: "ticket_fingerprint_changed", Message: "ticket requirements differ from the accepted fingerprint", Path: subject.source.Path})
		}
		if current.CriteriaSHA256 == nil {
			unknown("criteria_fingerprint_unavailable", "subject has no Acceptance criteria section to compare")
		} else if result.CriteriaSHA256 != nil && *result.CriteriaSHA256 != *current.CriteriaSHA256 {
			e.acceptanceFinding(result, "fail", Finding{Code: "criteria_fingerprint_changed", Message: "acceptance criteria differ from the accepted fingerprint", Path: subject.source.Path})
		}
	}
	e.acceptanceSnapshots(subject, r, result, current.TicketSHA256)
	if result.CheckStatus == "pass" {
		result.Reasons = append(result.Reasons, Finding{Code: "acceptance_valid", Message: "recorded acceptance matches current requirements and retained evidence", Path: r.source.Path})
	}
	return result
}

func (e *evaluator) acceptanceFinding(result *AcceptanceSummary, status string, reason Finding) {
	result.CheckStatus = combineCheckStatus(result.CheckStatus, status)
	switch result.CheckStatus {
	case "fail":
		result.Status = "stale"
	case "unknown":
		result.Status = "unknown"
	}
	result.Reasons = append(result.Reasons, reason)
	if status == "unknown" {
		e.diagnose(Diagnostic{Code: reason.Code, Severity: "error", Message: reason.Message, Path: reason.Path, From: reason.From, Link: reason.Link})
	}
}

func (e *evaluator) acceptanceMetadata(r *record, result *AcceptanceSummary) {
	invalid := func(field string) {
		e.acceptanceFinding(result, "unknown", Finding{Code: "invalid_acceptance_metadata", Message: "Acceptance requires a valid " + field, Path: r.source.Path})
	}
	result.Actor = acceptanceActor(r.doc.Metadata["actor"], false)
	if result.Actor == nil {
		invalid("actor with human or workflow kind and nonempty identity")
	}
	result.DecidedAt = acceptanceTime(r.doc.Metadata["decidedAt"])
	if result.DecidedAt == nil {
		invalid("decidedAt timestamp with an explicit UTC offset")
	}
	result.TestedRevision = acceptanceRevision(r.doc.Metadata["testedRevision"])
	if result.TestedRevision == nil {
		invalid("testedRevision origin and revision")
	}
	result.FingerprintVersion = acceptanceVersion(r.doc.Metadata["fingerprintVersion"])
	if result.FingerprintVersion == nil {
		invalid("integer fingerprintVersion")
	} else if *result.FingerprintVersion != 1 {
		e.acceptanceFinding(result, "unknown", Finding{Code: "unsupported_fingerprint_version", Message: "only fingerprint version 1 is supported", Path: r.source.Path})
	}
	result.TicketSHA256 = acceptanceDigest(r.doc.Metadata["ticketSHA256"])
	if result.TicketSHA256 == nil {
		invalid("lowercase ticketSHA256")
	}
	result.CriteriaSHA256 = acceptanceDigest(r.doc.Metadata["criteriaSHA256"])
	if result.CriteriaSHA256 == nil {
		invalid("lowercase criteriaSHA256")
	}
	if raw, exists := r.doc.Metadata["humanApprovals"]; exists {
		entries, ok := raw.([]any)
		if !ok {
			invalid("humanApprovals list")
		}
		for _, raw := range entries {
			entry, ok := raw.(map[string]any)
			approval := HumanApproval{Actor: acceptanceActor(entry["actor"], true), DecidedAt: acceptanceTime(entry["decidedAt"]), TestedRevision: acceptanceRevision(entry["testedRevision"])}
			result.HumanApprovals = append(result.HumanApprovals, approval)
			if !ok || approval.Actor == nil || approval.DecidedAt == nil || approval.TestedRevision == nil {
				invalid("humanApprovals entry with its own human actor, offset timestamp and tested revision")
			}
		}
	}
}

func acceptanceActor(value any, humanOnly bool) *Actor {
	fields, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	kind, _ := fields["kind"].(string)
	identity := stringField(fields, "identity")
	if identity == "" || (kind != "human" && (kind != "workflow" || humanOnly)) {
		return nil
	}
	return &Actor{Kind: kind, Identity: identity}
}

func acceptanceRevision(value any) *TestedRevision {
	fields, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	origin, revision := stringField(fields, "origin"), stringField(fields, "revision")
	if origin == "" || revision == "" {
		return nil
	}
	return &TestedRevision{Origin: origin, Revision: revision}
}

func acceptanceTime(value any) *string {
	text, ok := value.(string)
	if !ok || !recordread.ValidOffsetTimestamp(text) {
		return nil
	}
	return &text
}

func acceptanceDigest(value any) *string {
	text, ok := value.(string)
	if !ok || !sha256Pattern.MatchString(text) {
		return nil
	}
	return &text
}

func acceptanceVersion(value any) *int {
	var version int
	switch value := value.(type) {
	case int:
		version = value
	case int64:
		version = int(value)
		if int64(version) != value {
			return nil
		}
	case uint64:
		version = int(value)
		if version < 0 || uint64(version) != value {
			return nil
		}
	default:
		return nil
	}
	return &version
}

func (e *evaluator) acceptanceSnapshots(subject, record *record, result *AcceptanceSummary, ticketSHA256 string) {
	captured := e.acceptanceSources[record.source.Path]
	for _, reason := range captured.reasons {
		e.acceptanceFinding(result, "unknown", reason)
	}
	conflicting := map[string]bool{}
	seen := map[string]string{}
	for _, entry := range captured.entries {
		if entry.source == nil {
			continue
		}
		if expected, exists := seen[entry.source.Path]; exists && expected != entry.digest {
			conflicting[entry.source.Path] = true
		}
		seen[entry.source.Path] = entry.digest
	}
	requirements := map[string]bool{}
	for _, entry := range captured.entries {
		snapshot := Snapshot{Reference: entry.reference, SHA256: stringPointer(entry.digest), Status: "valid", Reasons: []Finding{}}
		problem := func(status, code, message string) {
			snapshot.Status = status
			reason := Finding{Code: code, Message: message, Path: entry.reference.Path, From: record.source.Path, Link: entry.link.Link}
			snapshot.Reasons = append(snapshot.Reasons, reason)
			checkStatus := "unknown"
			if status == "stale" {
				checkStatus = "fail"
			}
			e.acceptanceFinding(result, checkStatus, reason)
		}
		if entry.source == nil {
			problem("unknown", "snapshot_unavailable", "snapshot source is missing, unreadable, or outside the authorized scope")
		} else {
			current := entry.source.SHA256
			if entry.source.Path == subject.source.Path {
				current = ticketSHA256
			}
			snapshot.CurrentSHA256 = &current
			if entry.kind == "requirement" {
				requirements[entry.source.Path] = true
			}
			switch {
			case entry.source.Path == record.source.Path:
				problem("unknown", "acceptance_self_reference", "Acceptance cannot snapshot itself or use itself as evidence")
			case conflicting[entry.source.Path]:
				problem("unknown", "conflicting_snapshot", "the same source has conflicting recorded snapshot digests")
			case current != entry.digest:
				problem("stale", "snapshot_changed", "snapshot source differs from the recorded digest")
			}
		}
		if entry.kind == "requirement" {
			result.Requirements = append(result.Requirements, snapshot)
		} else {
			result.Evidence = append(result.Evidence, snapshot)
		}
	}
	context := e.contextCheck(subject)
	if context.Status == "unknown" {
		for _, reason := range context.Reasons {
			e.acceptanceFinding(result, "unknown", reason)
		}
	}
	if !e.declaration(subject, "Blocked by decisions", true) {
		e.acceptanceFinding(result, "unknown", Finding{Code: "incomplete_relationship_declaration", Message: "blocking decision requirement membership is unavailable", Path: subject.source.Path})
	}
	for _, selected := range e.selections {
		if selected.from != subject.source.Path || (selected.link.Kind != "spec" && selected.link.Kind != "context" && selected.link.Kind != "blocked_by_decision") {
			continue
		}
		if selected.source == nil {
			e.acceptanceFinding(result, "unknown", Finding{Code: "requirement_membership_unknown", Message: "selected requirement membership cannot be compared because its source is unavailable", Path: selected.reference.Path, From: selected.from, Link: selected.link.Link})
		} else if !requirements[selected.source.Path] && captured.requirementsKnown {
			e.acceptanceFinding(result, "fail", Finding{Code: "requirement_membership_changed", Message: "selected requirement source has no recorded Requirements snapshot", Path: selected.source.Path, From: selected.from, Link: selected.link.Link})
		}
	}
}
