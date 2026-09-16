package cli

import (
	"fmt"
	"strings"

	"github.com/Zokiio/context/internal/orientation"
)

func writeAcceptance(output *strings.Builder, acceptance *orientation.AcceptanceSummary) {
	if acceptance == nil {
		fmt.Fprintln(output, "    acceptance: none recorded")
		return
	}
	fmt.Fprintf(output, "    acceptance: %s\n    acceptance check: %s\n", acceptance.Status, acceptance.CheckStatus)
	if acceptance.Record == nil {
		fmt.Fprintln(output, "    Acceptance record: unknown")
	} else {
		writeReferences(output, "    Acceptance record", []orientation.Reference{*acceptance.Record})
	}
	writeAttribution(output, "      ", acceptance.Actor, acceptance.DecidedAt, acceptance.TestedRevision)
	if acceptance.FingerprintVersion == nil {
		fmt.Fprintln(output, "      accepted fingerprint: unknown")
	} else {
		fmt.Fprintf(output, "      accepted fingerprint: version %d\n", *acceptance.FingerprintVersion)
	}
	fmt.Fprintf(output, "      accepted ticketSHA256: %s\n      accepted criteriaSHA256: %s\n", knownString(acceptance.TicketSHA256), knownString(acceptance.CriteriaSHA256))
	if len(acceptance.HumanApprovals) == 0 {
		fmt.Fprintln(output, "    Human approvals: none recorded")
	} else {
		fmt.Fprintln(output, "    Human approvals:")
		for i, approval := range acceptance.HumanApprovals {
			fmt.Fprintf(output, "      Approval %d:\n", i+1)
			writeAttribution(output, "        ", approval.Actor, approval.DecidedAt, approval.TestedRevision)
		}
	}
	writeSnapshots(output, "Requirements", acceptance.Requirements)
	writeSnapshots(output, "Evidence", acceptance.Evidence)
	for _, reason := range acceptance.Reasons {
		writeFinding(output, "      ", reason)
	}
}

func writeAttribution(output *strings.Builder, indent string, actor *orientation.Actor, decidedAt *string, revision *orientation.TestedRevision) {
	if actor == nil {
		fmt.Fprintf(output, "%sactor: unknown\n", indent)
	} else {
		fmt.Fprintf(output, "%sactor: %s %s\n", indent, actor.Kind, actor.Identity)
	}
	fmt.Fprintf(output, "%sdecided at: %s\n", indent, knownString(decidedAt))
	if revision == nil {
		fmt.Fprintf(output, "%stested revision (historical): unknown\n", indent)
	} else {
		fmt.Fprintf(output, "%stested revision (historical): %s %s\n", indent, revision.Origin, revision.Revision)
	}
}

func writeSnapshots(output *strings.Builder, heading string, snapshots []orientation.Snapshot) {
	if len(snapshots) == 0 {
		fmt.Fprintf(output, "    %s: none observed\n", heading)
		return
	}
	fmt.Fprintf(output, "    %s:\n", heading)
	for _, snapshot := range snapshots {
		writeReferences(output, "      Source", []orientation.Reference{snapshot.Reference})
		fmt.Fprintf(output, "        status: %s\n        recordedSHA256: %s\n        currentSHA256: %s\n", snapshot.Status, knownString(snapshot.SHA256), knownString(snapshot.CurrentSHA256))
		for _, reason := range snapshot.Reasons {
			writeFinding(output, "        ", reason)
		}
	}
}
