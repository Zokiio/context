package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
)

func TestOrientationTextAttributesRecordedAcceptance(t *testing.T) {
	var stdout, stderr bytes.Buffer
	version := 1
	result := orientation.Result{SchemaVersion: 1, Complete: true, InventoryComplete: true,
		WorkItems: []orientation.WorkItem{{ID: str("leaf"), Title: str("Completed leaf"), Source: "/bundle/leaf.md", Execution: str("completed"),
			Acceptance: &orientation.AcceptanceSummary{
				Status: "stale", CheckStatus: "fail",
				Record: &orientation.Reference{Path: "/bundle/acceptance.md", ID: str("acceptance"), Title: str("Current decision"), From: "/bundle/leaf.md", Link: "acceptance.md"},
				Actor:  &orientation.Actor{Kind: "workflow", Identity: "build-agent"}, DecidedAt: str("2026-09-15T15:00:00+02:00"),
				TestedRevision:     &orientation.TestedRevision{Origin: "local-artifact", Revision: "leaf-build-123"},
				FingerprintVersion: &version, TicketSHA256: str(strings.Repeat("a", 64)), CriteriaSHA256: str(strings.Repeat("b", 64)),
				HumanApprovals: []orientation.HumanApproval{{Actor: &orientation.Actor{Kind: "human", Identity: "Alice"}, DecidedAt: str("2026-09-15T16:00:00+02:00"),
					TestedRevision: &orientation.TestedRevision{Origin: "review-artifact", Revision: "reviewed-leaf-123"}}},
				Requirements: []orientation.Snapshot{{Reference: orientation.Reference{Path: "/docs/spec.md", From: "/bundle/acceptance.md", Link: "../docs/spec.md"},
					SHA256: str(strings.Repeat("c", 64)), CurrentSHA256: str(strings.Repeat("c", 64)), Status: "valid"}},
				Evidence: []orientation.Snapshot{{Reference: orientation.Reference{Path: "/docs/evidence.md", From: "/bundle/acceptance.md", Link: "../docs/evidence.md"},
					SHA256: str(strings.Repeat("d", 64)), CurrentSHA256: str(strings.Repeat("e", 64)), Status: "stale",
					Reasons: []orientation.Finding{{Code: "snapshot_changed", Message: "Evidence bytes changed.", Path: "/docs/evidence.md"}}}},
				Reasons: []orientation.Finding{{Code: "acceptance_stale", Message: "Reassess the changed evidence.", Path: "/bundle/acceptance.md"}},
			},
		}},
	}
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", ".", "--detail"}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return result, nil
	}})
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stderr=%q stdout=%q", status, stderr.String(), stdout.String())
	}
	for _, fact := range []string{
		"acceptance: stale", "acceptance check: fail", "Current decision [acceptance]", "/bundle/acceptance.md", "link \"acceptance.md\"",
		"actor: workflow build-agent", "decided at: 2026-09-15T15:00:00+02:00", "tested revision (historical): local-artifact leaf-build-123",
		"accepted fingerprint: version 1", "accepted ticketSHA256: " + strings.Repeat("a", 64), "accepted criteriaSHA256: " + strings.Repeat("b", 64),
		"Human approvals:", "actor: human Alice", "decided at: 2026-09-15T16:00:00+02:00", "tested revision (historical): review-artifact reviewed-leaf-123",
		"Requirements:", "/docs/spec.md", "link \"../docs/spec.md\"", "recordedSHA256: " + strings.Repeat("c", 64), "currentSHA256: " + strings.Repeat("c", 64),
		"Evidence:", "/docs/evidence.md", "status: stale", "recordedSHA256: " + strings.Repeat("d", 64), "currentSHA256: " + strings.Repeat("e", 64),
		"snapshot_changed: Evidence bytes changed.", "acceptance_stale: Reassess the changed evidence.",
	} {
		if !strings.Contains(stdout.String(), fact) {
			t.Errorf("missing acceptance fact %q", fact)
		}
	}
	if t.Failed() {
		t.Log(stdout.String())
	}
	text := stdout.String()
	if start := strings.Index(text, "Human approvals:"); start >= 0 {
		end := strings.Index(text[start:], "Requirements:")
		if end < 0 || strings.Contains(text[start:start+end], "build-agent") {
			t.Fatalf("workflow actor must remain separate from human approval: %s", text)
		}
	}
}

func TestOrientationTextShowsNoAcceptanceForUnfinishedWork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), []string{"ctx", "orient", "--bundle", ".", "--detail"}, &stdout, &stderr, cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		return orientation.Result{Complete: true, WorkItems: []orientation.WorkItem{{ID: str("work"), Execution: str("unstarted")}}}, nil
	}})
	if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "acceptance: none recorded") {
		t.Fatalf("no current acceptance is separate from unknown validation: status=%d stderr=%q stdout=%q", status, stderr.String(), stdout.String())
	}
}
