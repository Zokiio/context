package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/Zokiio/context/internal/orientation"
)

func renderOrientation(output io.Writer, result orientation.Result) error {
	var text strings.Builder
	if result.Project == nil {
		fmt.Fprintln(&text, "Project: unknown")
	} else {
		fmt.Fprintf(&text, "Project: %s [%s]\n", result.Project.Title, result.Project.ID)
		fmt.Fprintf(&text, "  source: %s\n", result.Project.Source)
	}
	completion := func(complete bool) string {
		if complete {
			return "complete"
		}
		return "partial"
	}
	fmt.Fprintf(&text, "Evaluation: %s; inventory: %s\n", completion(result.Complete), completion(result.InventoryComplete))
	fmt.Fprintln(&text)
	switch {
	case result.Project == nil || !result.Project.GoalsKnown:
		fmt.Fprintln(&text, "Goals: unknown")
	case len(result.Goals) == 0:
		fmt.Fprintln(&text, "Goals: none")
	default:
		fmt.Fprintln(&text, "Goals:")
	}
	for _, goal := range result.Goals {
		fmt.Fprintln(&text, strings.TrimSpace(goal.Text))
		fmt.Fprintf(&text, "  source: %s\n", goal.Source)
		writeReferences(&text, "  References", goal.References)
	}
	fmt.Fprintln(&text)
	writeProjectReferences(&text, "Current commitments", result.CurrentCommitments, result.Project != nil && result.Project.CommitmentsKnown)
	writeReferences(&text, "Shortlist", result.Shortlist)
	writeReferences(&text, "In progress", result.InProgress)
	writeReferences(&text, "Backlog", result.Backlog)
	fmt.Fprintln(&text, "\nWork inventory:")
	if len(result.WorkItems) == 0 {
		fmt.Fprintln(&text, "  none observed")
	}
	for _, item := range result.WorkItems {
		fmt.Fprintf(&text, "  %s [%s]\n    source: %s\n    identity ambiguous: %t\n", knownString(item.Title), knownString(item.ID), item.Source, item.IdentityAmbiguous)
		fmt.Fprintf(&text, "    triage: %s; execution: %s; lifecycle: %s\n", knownString(item.Triage), knownString(item.Execution), knownString(item.Lifecycle))
		committed := "unknown"
		if item.Committed != nil {
			committed = fmt.Sprint(*item.Committed)
		}
		fmt.Fprintf(&text, "    committed: %s; readiness: %s; eligible: %t\n", committed, item.Readiness, item.Eligible)
		writeReferences(&text, "    Specifications", item.Specifications)
		for _, check := range item.Checks {
			fmt.Fprintf(&text, "    %s: %s\n", check.Name, check.Status)
			for _, reason := range check.Reasons {
				writeFinding(&text, "      ", reason)
			}
		}
		for _, reason := range item.ExclusionReasons {
			writeFinding(&text, "    excluded: ", reason)
		}
		if item.Acceptance == nil {
			fmt.Fprintln(&text, "    acceptance: unknown")
		} else {
			fmt.Fprintf(&text, "    acceptance: %s\n", item.Acceptance.Status)
			if item.Acceptance.Record != nil {
				writeReferences(&text, "    Acceptance record", []orientation.Reference{*item.Acceptance.Record})
			}
		}
		if item.FingerprintVersion == nil {
			fmt.Fprintln(&text, "    fingerprint: unknown")
		} else {
			fmt.Fprintf(&text, "    fingerprint: version %d\n", *item.FingerprintVersion)
			fmt.Fprintf(&text, "      ticketSHA256: %s\n      criteriaSHA256: %s\n", knownString(item.TicketSHA256), knownString(item.CriteriaSHA256))
		}
	}
	if len(result.WorkItems) > 0 {
		fmt.Fprintln(&text, "Task context: use a work item's source as --ticket with the same --project and --allow-source arguments.")
	}
	fmt.Fprintln(&text)
	var indexedDecisions []orientation.Reference
	if result.Project != nil {
		for _, decision := range result.Decisions {
			for _, ref := range decision.References {
				if ref.From == result.Project.Source {
					indexedDecisions = append(indexedDecisions, ref)
				}
			}
		}
	}
	writeProjectReferences(&text, "Open decisions", indexedDecisions, result.Project != nil && result.Project.OpenDecisionsKnown)
	fmt.Fprintln(&text, "Decision inventory:")
	if len(result.Decisions) == 0 {
		fmt.Fprintln(&text, "  none observed")
	}
	for _, decision := range result.Decisions {
		fmt.Fprintf(&text, "  %s [%s]\n    source: %s\n    identity ambiguous: %t\n", knownString(decision.Title), knownString(decision.ID), decision.Source, decision.IdentityAmbiguous)
		fmt.Fprintf(&text, "    state: %s\n    resolution: %s\n", knownString(decision.State), knownString(decision.Resolution))
		writeReferences(&text, "    References", decision.References)
	}
	fmt.Fprintln(&text, "\nSources:")
	if len(result.Sources) == 0 {
		fmt.Fprintln(&text, "  none observed")
	}
	for _, source := range result.Sources {
		fmt.Fprintf(&text, "  %s\n    sha256: %s\n", source.Path, source.SHA256)
		for _, reason := range source.Reasons {
			fmt.Fprintf(&text, "    %s%s\n", reason.Kind, relationshipDetails(reason.From, reason.Link))
		}
	}
	fmt.Fprintln(&text, "\nDiagnostics:")
	if len(result.Diagnostics) == 0 {
		fmt.Fprintln(&text, "  none")
	}
	for _, diagnostic := range result.Diagnostics {
		writeFinding(&text, "  "+diagnostic.Severity+" ", orientation.Finding{
			Code: diagnostic.Code, Message: diagnostic.Message, Path: diagnostic.Path,
			From: diagnostic.From, Link: diagnostic.Link,
		})
	}
	_, err := io.WriteString(output, text.String())
	return err
}

func writeProjectReferences(output *strings.Builder, heading string, refs []orientation.Reference, known bool) {
	if known {
		writeReferences(output, heading, refs)
		return
	}
	fmt.Fprintf(output, "%s: unknown\n", heading)
	if len(refs) > 0 {
		writeReferences(output, "  Known references", refs)
	}
}

func writeFinding(output *strings.Builder, prefix string, finding orientation.Finding) {
	fmt.Fprintf(output, "%s%s: %s", prefix, finding.Code, finding.Message)
	if finding.Path != "" {
		fmt.Fprintf(output, " [%s]", finding.Path)
	}
	fmt.Fprintln(output, relationshipDetails(finding.From, finding.Link))
}

func writeReferences(output *strings.Builder, heading string, refs []orientation.Reference) {
	if len(refs) == 0 {
		fmt.Fprintf(output, "%s: none\n", heading)
		return
	}
	fmt.Fprintf(output, "%s:\n", heading)
	for _, ref := range refs {
		fmt.Fprintf(output, "  %s [%s] %s%s\n", knownString(ref.Title), knownString(ref.ID), ref.Path, relationshipDetails(ref.From, ref.Link))
	}
}

func knownString(value *string) string {
	if value == nil {
		return "unknown"
	}
	return *value
}

func relationshipDetails(from, link string) string {
	if from == "" && link == "" {
		return ""
	}
	return fmt.Sprintf(" (from %s; link %q)", from, link)
}
