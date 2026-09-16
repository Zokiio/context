package orientation_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestAcceptedCompletedLeafSatisfiesDirectPrerequisite(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || !got.Complete {
		t.Fatalf("evaluation complete=%v diagnostics=%+v error=%v", got.Complete, got.Diagnostics, err)
	}
	leaf, dependent := findWork(t, got, "leaf.md"), findWork(t, got, "dependent.md")
	if leaf.Acceptance == nil || leaf.Acceptance.Status != "valid" || leaf.Acceptance.CheckStatus != "pass" || dependent.Readiness != "ready" || !dependent.Eligible || checkStatus(dependent, "dependencies") != "pass" {
		t.Fatalf("leaf=%+v dependent=%+v", leaf, dependent)
	}
	accepted := leaf.Acceptance
	if accepted.Actor == nil || accepted.Actor.Kind != "workflow" || accepted.Actor.Identity != "test-agent" || accepted.DecidedAt == nil || accepted.TestedRevision == nil || accepted.TestedRevision.Revision != "historical-revision" || len(accepted.Evidence) != 1 || accepted.Evidence[0].Status != "valid" {
		t.Fatalf("acceptance attribution: %+v", accepted)
	}
}

func TestAcceptanceMetadataIsExplicitAndSeparatelyAttributed(t *testing.T) {
	for _, tc := range []struct{ name, before, after, code string }{
		{"project mismatch", "projectId: project", "projectId: other", "acceptance_subject_mismatch"},
		{"subject mismatch", "workItemId: leaf", "workItemId: other", "acceptance_subject_mismatch"},
		{"actor kind", "kind: workflow", "kind: bot", "invalid_acceptance_metadata"},
		{"actor identity", "identity: test-agent", "identity: ''", "invalid_acceptance_metadata"},
		{"missing actor", "actor: {kind: workflow, identity: test-agent}\n", "", "invalid_acceptance_metadata"},
		{"timestamp offset", "2026-09-16T09:00:00+02:00", "2026-09-16T09:00:00", "invalid_acceptance_metadata"},
		{"timestamp validity", "2026-09-16T09:00:00+02:00", "2026-99-16T09:00:00+02:00", "invalid_acceptance_metadata"},
		{"invalid offset hour", "2026-09-16T09:00:00+02:00", "2026-09-16T09:00:00+24:00", "invalid_acceptance_metadata"},
		{"invalid offset minute", "2026-09-16T09:00:00+02:00", "2026-09-16T09:00:00+02:60", "invalid_acceptance_metadata"},
		{"tested origin", "origin: local-tests", "origin: ''", "invalid_acceptance_metadata"},
		{"tested revision", "revision: historical-revision", "revision: ''", "invalid_acceptance_metadata"},
		{"unsupported version", "fingerprintVersion: 1", "fingerprintVersion: 2", "unsupported_fingerprint_version"},
		{"string version", "fingerprintVersion: 1", "fingerprintVersion: '1'", "invalid_acceptance_metadata"},
		{"float version", "fingerprintVersion: 1", "fingerprintVersion: 1.0", "invalid_acceptance_metadata"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := acceptedLeafProject(t, "", "None\n", "")
			replaceFile(t, project, "acceptance.md", tc.before, tc.after)
			got := orientProject(t, project)
			accepted := findWork(t, got, "leaf.md").Acceptance
			if got.Complete || accepted == nil || accepted.Status != "unknown" || !hasCode(got, tc.code) || findWork(t, got, "dependent.md").Eligible {
				t.Fatalf("acceptance=%+v complete=%v diagnostics=%+v", accepted, got.Complete, got.Diagnostics)
			}
		})
	}
	approval := "humanApprovals:\n  - actor: {kind: human, identity: Alice}\n    decidedAt: '2026-09-17T10:00:00Z'\n    testedRevision: {origin: manual-check, revision: independently-tested}\n"
	project := acceptedLeafProject(t, "", "None\n", approval)
	got := orientProject(t, project)
	accepted := findWork(t, got, "leaf.md").Acceptance
	if !got.Complete || accepted.Status != "valid" || len(accepted.HumanApprovals) != 1 || accepted.HumanApprovals[0].Actor.Identity != "Alice" || accepted.HumanApprovals[0].TestedRevision.Revision != "independently-tested" || accepted.Actor.Kind != "workflow" {
		t.Fatalf("attribution: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
	replaceFile(t, project, "acceptance.md", "kind: human", "kind: workflow")
	got = orientProject(t, project)
	accepted = findWork(t, got, "leaf.md").Acceptance
	if got.Complete || accepted.Status != "unknown" || accepted.HumanApprovals[0].Actor != nil || !hasCode(got, "invalid_acceptance_metadata") {
		t.Fatalf("workflow was promoted to human: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
}

func TestCurrentAcceptanceRequiresExactlyOneTypedUnambiguousRecord(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, string)
		code   string
	}{
		{"missing", func(t *testing.T, p string) { replaceFile(t, p, "leaf.md", "[Current](acceptance.md)", "None") }, "invalid_current_acceptance"},
		{"multiple", func(t *testing.T, p string) {
			replaceFile(t, p, "leaf.md", "[Current](acceptance.md)", "[Current](acceptance.md) [Another](acceptance.md)")
		}, "invalid_current_acceptance"},
		{"unresolved", func(t *testing.T, p string) {
			replaceFile(t, p, "leaf.md", "[Current](acceptance.md)", "[Current](acceptance.md) [Other][missing]")
		}, "invalid_current_acceptance"},
		{"malformed section", func(t *testing.T, p string) {
			replaceFile(t, p, "leaf.md", "## Acceptance\n[Current](acceptance.md)", "## Acceptance\n[Current](acceptance.md)\n## Acceptance\nAsk the team later.")
		}, "invalid_current_acceptance"},
		{"wrong type", func(t *testing.T, p string) { replaceFile(t, p, "acceptance.md", "type: Acceptance", "type: Other") }, "acceptance_unavailable"},
		{"duplicate identity", func(t *testing.T, p string) {
			writeFile(t, p, "duplicate.md", "---\ntype: Other\nid: accepted-leaf\n---\n")
		}, "invalid_acceptance_identity"},
		{"missing source", func(t *testing.T, p string) {
			if err := os.Remove(filepath.Join(p, "acceptance.md")); err != nil {
				t.Fatal(err)
			}
		}, "acceptance_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := acceptedLeafProject(t, "", "None\n", "")
			tc.mutate(t, project)
			got := orientProject(t, project)
			accepted := findWork(t, got, "leaf.md").Acceptance
			if got.Complete || accepted.Status != "unknown" || !hasCode(got, tc.code) || findWork(t, got, "dependent.md").Eligible {
				t.Fatalf("acceptance=%+v diagnostics=%+v", accepted, got.Diagnostics)
			}
		})
	}
}

func TestAcceptanceStalenessRetainsKnownFailureAndUnknownEvidence(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	writeFile(t, project, "evidence.txt", "Edited observations.\n")
	got := orientProject(t, project)
	accepted := findWork(t, got, "leaf.md").Acceptance
	if !got.Complete || accepted.Status != "stale" || accepted.Evidence[0].Status != "stale" || checkStatus(findWork(t, got, "dependent.md"), "dependencies") != "fail" {
		t.Fatalf("acceptance=%+v diagnostics=%+v", accepted, got.Diagnostics)
	}
	replaceFile(t, project, "leaf.md", "A useful result.", "A different criterion.")
	if err := os.Remove(filepath.Join(project, "evidence.txt")); err != nil {
		t.Fatal(err)
	}
	got = orientProject(t, project)
	accepted = findWork(t, got, "leaf.md").Acceptance
	for _, code := range []string{"ticket_fingerprint_changed", "criteria_fingerprint_changed", "snapshot_unavailable"} {
		if !hasFinding(accepted.Reasons, code) {
			t.Fatalf("missing %s: %+v", code, accepted.Reasons)
		}
	}
	if got.Complete || accepted.Status != "stale" || accepted.CheckStatus != "fail" || accepted.Evidence[0].Status != "unknown" || findWork(t, got, "dependent.md").Readiness != "blocked" {
		t.Fatalf("acceptance=%+v diagnostics=%+v", accepted, got.Diagnostics)
	}
}

func TestAcceptanceSnapshotSyntaxAndReferenceLinks(t *testing.T) {
	digest := sourceHash("Recorded passing observations.\n")
	for _, tc := range []struct {
		name, evidence string
		valid          bool
	}{
		{"inline", "- [Evidence](evidence.txt) `" + digest + "`\n", true},
		{"reference", "- [Evidence][proof] `" + digest + "`\n\n[proof]: evidence.txt\n", true},
		{"ordered", "1. [Evidence](evidence.txt) `" + digest + "`\n", true},
		{"repeated", "- [Evidence](evidence.txt) `" + digest + "`\n- [Again](evidence.txt#details) `" + digest + "`\n", true},
		{"paragraph", "[Evidence](evidence.txt) `" + digest + "`\n", false},
		{"no digest", "- [Evidence](evidence.txt)\n", false},
		{"plain digest", "- [Evidence](evidence.txt) " + digest + "\n", false},
		{"uppercase digest", "- [Evidence](evidence.txt) `" + strings.ToUpper(digest) + "`\n", false},
		{"short digest", "- [Evidence](evidence.txt) `abc123`\n", false},
		{"two links", "- [Evidence](evidence.txt) [Other](evidence.txt) `" + digest + "`\n", false},
		{"two digests", "- [Evidence](evidence.txt) `" + digest + "` `" + digest + "`\n", false},
		{"unresolved extra link", "- [Evidence](evidence.txt) [Other][missing] `" + digest + "`\n", false},
		{"nested", "- [Evidence](evidence.txt) `" + digest + "`\n  - [Other](evidence.txt) `" + digest + "`\n", false},
		{"fenced fake", "```md\n- [Evidence](evidence.txt) `" + digest + "`\n```\n", false},
		{"none", "None\n", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := acceptedLeafProject(t, "", "None\n", "")
			data, err := os.ReadFile(filepath.Join(project, "acceptance.md"))
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, project, "acceptance.md", strings.Split(string(data), "## Evidence\n")[0]+"## Evidence\n"+tc.evidence)
			got := orientProject(t, project)
			accepted := findWork(t, got, "leaf.md").Acceptance
			if got.Complete != tc.valid || (accepted.Status == "valid") != tc.valid || findWork(t, got, "dependent.md").Eligible != tc.valid {
				t.Fatalf("acceptance=%+v diagnostics=%+v", accepted, got.Diagnostics)
			}
		})
	}
}

func TestRequirementSnapshotsCoverSelectedFilesAndKeepExtraRequirements(t *testing.T) {
	spec, contextText, extra := "Authored specification.\n", "Shared context.\n", "Additional requirement.\n"
	decision := decisionRecord("choice", "resolved", "## Resolution\nAn answer.\n")
	leafExtra := "## Spec\n[Spec](spec.txt)\n## Context\n[Context](context.txt)\n"
	requirements := "- [Spec](spec.txt) `" + sourceHash(spec) + "`\n- [Context](context.txt) `" + sourceHash(contextText) + "`\n- [Extra](extra.txt) `" + sourceHash(extra) + "`\n- [Decision](choice.md) `" + sourceHash(decision) + "`\n"
	project := acceptedLeafProject(t, leafExtra, requirements, "")
	writeFile(t, project, "spec.txt", spec)
	writeFile(t, project, "context.txt", contextText)
	writeFile(t, project, "extra.txt", extra)
	writeFile(t, project, "choice.md", decision)
	replaceFile(t, project, "leaf.md", "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](choice.md)")
	reacceptLeaf(t, project, requirements, "")
	got := orientProject(t, project)
	if !got.Complete || !findWork(t, got, "dependent.md").Eligible {
		t.Fatalf("valid requirements: %+v", got.Diagnostics)
	}
	writeFile(t, project, "extra.txt", "Changed extra requirement.\n")
	got = orientProject(t, project)
	accepted := findWork(t, got, "leaf.md").Acceptance
	if !got.Complete || accepted.Status != "stale" || !hasFinding(accepted.Reasons, "snapshot_changed") {
		t.Fatalf("extra snapshot: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
	writeFile(t, project, "extra.txt", extra)
	replaceFile(t, project, "acceptance.md", "- [Context](context.txt) `"+sourceHash(contextText)+"`\n", "")
	got = orientProject(t, project)
	accepted = findWork(t, got, "leaf.md").Acceptance
	if !got.Complete || accepted.Status != "stale" || !hasFinding(accepted.Reasons, "requirement_membership_changed") {
		t.Fatalf("membership: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
	if err := os.Remove(filepath.Join(project, "extra.txt")); err != nil {
		t.Fatal(err)
	}
	got = orientProject(t, project)
	accepted = findWork(t, got, "leaf.md").Acceptance
	if got.Complete || accepted.Status != "stale" || !hasFinding(accepted.Reasons, "snapshot_unavailable") {
		t.Fatalf("missing extra snapshot must coexist with known membership failure: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
	replaceFile(t, project, "acceptance.md", "## Evidence", "## Unstructured evidence")
	got = orientProject(t, project)
	accepted = findWork(t, got, "leaf.md").Acceptance
	if got.Complete || accepted.Status != "stale" || !hasFinding(accepted.Reasons, "requirement_membership_changed") || !hasFinding(accepted.Reasons, "missing_snapshot_section") {
		t.Fatalf("known requirement membership failure lost beside malformed evidence: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
}

func TestAcceptanceRejectsSelfAndConflictingSnapshots(t *testing.T) {
	for _, tc := range []struct{ name, requirements, code string }{
		{"self requirement", "- [Self](acceptance.md) `" + strings.Repeat("a", 64) + "`\n", "acceptance_self_reference"},
		{"conflicting", "- [Same](evidence.txt) `" + strings.Repeat("a", 64) + "`\n", "conflicting_snapshot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := acceptedLeafProject(t, "", tc.requirements, "")
			got := orientProject(t, project)
			accepted := findWork(t, got, "leaf.md").Acceptance
			if got.Complete || accepted.Status != "unknown" || !hasCode(got, tc.code) {
				t.Fatalf("acceptance=%+v diagnostics=%+v", accepted, got.Diagnostics)
			}
		})
	}
	project := acceptedLeafProject(t, "", "None\n", "")
	replaceFile(t, project, "acceptance.md", "[Evidence](evidence.txt)", "[Self](acceptance.md)")
	got := orientProject(t, project)
	if got.Complete || findWork(t, got, "leaf.md").Acceptance.Status != "unknown" || !hasCode(got, "acceptance_self_reference") {
		t.Fatalf("self evidence: %+v", got.Diagnostics)
	}
}

func TestSubjectSnapshotsUseFilteredFingerprintAndHistoryDoesNotExpand(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	before := orientProject(t, project)
	leaf := findWork(t, before, "leaf.md")
	requirements := "- [Subject](leaf.md) `" + *leaf.TicketSHA256 + "`\n"
	reacceptLeaf(t, project, requirements, "")
	writeFile(t, project, "old-acceptance.md", "---\ntype: Acceptance\nid: historical\ntitle: Prior acceptance\n---\n## Requirements\n- [Old](missing-old.txt) `"+strings.Repeat("a", 64)+"`\n## Evidence\n- [Old evidence](https://example.test/missing) `"+strings.Repeat("b", 64)+"`\n")
	replaceFile(t, project, "leaf.md", "[Current](acceptance.md)", "[Current](acceptance.md)\n## Comments\nOriginal decision: [History](old-acceptance.md)\n")
	got := orientProject(t, project)
	accepted := findWork(t, got, "leaf.md").Acceptance
	if !got.Complete || accepted.Status != "valid" || accepted.Requirements[0].CurrentSHA256 == nil || *accepted.Requirements[0].CurrentSHA256 != *leaf.TicketSHA256 {
		t.Fatalf("filtered subject snapshot: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
	for _, source := range got.Sources {
		for _, reason := range source.Reasons {
			if reason.From == filepath.Join(project, "old-acceptance.md") {
				t.Fatalf("historical source expanded: %+v", source)
			}
		}
	}
}

func TestDirectDependenciesRetainExecutionDecisionAndNonleafReasons(t *testing.T) {
	for _, state := range []string{"unstarted", "in-progress", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			project := acceptedLeafProject(t, "", "None\n", "")
			replaceFile(t, project, "leaf.md", "execution: completed", "execution: "+state)
			got := orientProject(t, project)
			dependent := findWork(t, got, "dependent.md")
			if !got.Complete || dependent.Readiness != "blocked" || checkStatus(dependent, "dependencies") != "fail" {
				t.Fatalf("state=%s dependent=%+v diagnostics=%+v", state, dependent, got.Diagnostics)
			}
		})
	}
	project := acceptedLeafProject(t, "", "None\n", "")
	writeFile(t, project, "prior.md", workItem("prior", "unstarted", ""))
	replaceFile(t, project, "leaf.md", "## Blocked by\nNone", "## Blocked by\n[Prior](prior.md)")
	reacceptLeaf(t, project, "None\n", "")
	got := orientProject(t, project)
	leaf, dependent := findWork(t, got, "leaf.md"), findWork(t, got, "dependent.md")
	if got.Complete || leaf.Acceptance.Status != "valid" || checkStatus(dependent, "dependencies") != "unknown" || !hasCode(got, "unsupported_check") {
		t.Fatalf("nonleaf leaf=%+v dependent=%+v diagnostics=%+v", leaf, dependent, got.Diagnostics)
	}
	decision := decisionRecord("choice", "open", "")
	writeFile(t, project, "choice.md", decision)
	replaceFile(t, project, "leaf.md", "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](choice.md)")
	reacceptLeaf(t, project, "- [Decision](choice.md) `"+sourceHash(decision)+"`\n", "")
	got = orientProject(t, project)
	dependent = findWork(t, got, "dependent.md")
	if got.Complete || dependent.Readiness != "blocked" || !hasCheckFinding(dependent, "dependencies", "open_decision") || !hasCheckFinding(dependent, "dependencies", "unsupported_check") {
		t.Fatalf("mixed decision fail and unsupported chain: %+v diagnostics=%+v", dependent, got.Diagnostics)
	}
}

func TestAcceptanceSnapshotsRespectScopeAndSharedBudget(t *testing.T) {
	for _, tc := range []struct {
		name, content, link, code string
		allowed, valid            bool
	}{
		{"authorized text", "Retained proof.\n", "../proof.txt", "", true, true},
		{"unauthorized", "Retained proof.\n", "../proof.txt", "source_outside_scope", false, false},
		{"remote", "Retained proof.\n", "https://example.test/proof", "unsupported_source", true, false},
		{"missing", "Retained proof.\n", "../absent.txt", "source_missing", true, false},
		{"binary", string([]byte{0xff}), "../proof.txt", "invalid_source_encoding", true, false},
		{"external record", "---\ntype: WorkItem\nid: external\ntitle: External record\n---\n", "../proof.txt", "source_outside_scope", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := acceptedLeafProject(t, "", "None\n", "")
			parent := filepath.Dir(project)
			writeFile(t, parent, "proof.txt", tc.content)
			replaceFile(t, project, "acceptance.md", "[Evidence](evidence.txt) `"+sourceHash("Recorded passing observations.\n")+"`", "[Evidence]("+tc.link+") `"+sourceHash(tc.content)+"`")
			request := orientation.Request{ProjectDir: project}
			if tc.allowed {
				request.AllowedSourceDirs = []string{parent}
			}
			got, err := orientation.Orient(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			accepted := findWork(t, got, "leaf.md").Acceptance
			if got.Complete != tc.valid || (accepted.Status == "valid") != tc.valid || (tc.code != "" && !hasCode(got, tc.code)) {
				t.Fatalf("acceptance=%+v diagnostics=%+v", accepted, got.Diagnostics)
			}
		})
	}
	project := acceptedLeafProject(t, "", "None\n", "")
	digest := sourceHash("Recorded passing observations.\n")
	replaceFile(t, project, "acceptance.md", "## Requirements\nNone", "## Requirements\n- [Shared](evidence.txt#requirements) `"+digest+"`")
	var bytes int64
	for _, path := range []string{"project.md", "leaf.md", "dependent.md", "acceptance.md", "evidence.txt"} {
		info, err := os.Stat(filepath.Join(project, path))
		if err != nil {
			t.Fatal(err)
		}
		bytes += info.Size()
	}
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, MaxFiles: 5, MaxBytes: bytes})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Complete || len(got.Sources) != 5 {
		t.Fatalf("shared snapshots counted twice: sources=%+v diagnostics=%+v", got.Sources, got.Diagnostics)
	}
	for _, source := range got.Sources {
		if filepath.Base(source.Path) == "evidence.txt" && (len(source.Reasons) != 2 || source.Reasons[0].Kind != "requirement" || source.Reasons[1].Kind != "evidence") {
			t.Fatalf("snapshot provenance: %+v", source)
		}
	}
	got, err = orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, MaxFiles: 4, MaxBytes: bytes})
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete || !got.InventoryComplete || !hasCode(got, "source_limit_exceeded") || findWork(t, got, "leaf.md").Acceptance.Status != "unknown" {
		t.Fatalf("snapshot limit: %+v", got.Diagnostics)
	}
	got, err = orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, MaxFiles: 5, MaxBytes: bytes - 1})
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete || !hasCode(got, "source_limit_exceeded") || findWork(t, got, "leaf.md").Acceptance.Status != "unknown" {
		t.Fatalf("snapshot byte limit: %+v", got.Diagnostics)
	}
}

func TestAcceptanceRejectsExternalCurrentRecordAlreadyReadAsContext(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	data, err := os.ReadFile(filepath.Join(project, "acceptance.md"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Dir(project), "outside.txt", string(data))
	replaceFile(t, project, "leaf.md", "## Acceptance\n[Current](acceptance.md)", "## Context\n[Known](../outside.txt)\n## Acceptance\n[Current](../outside.txt)")
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, AllowedSourceDirs: []string{filepath.Dir(project)}})
	if err != nil {
		t.Fatal(err)
	}
	accepted := findWork(t, got, "leaf.md").Acceptance
	if got.Complete || accepted.Status != "unknown" || !hasCode(got, "source_outside_scope") {
		t.Fatalf("cached external record accepted: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
}

func TestAcceptanceUnavailableEvidenceAndInvalidMetadataDoNotPanic(t *testing.T) {
	for _, extra := range []string{"humanApprovals: null\n", "humanApprovals: {actor: bot}\n", "humanApprovals: [null, plain, {actor: 1}]\n"} {
		project := acceptedLeafProject(t, "", "None\n", extra)
		got := orientProject(t, project)
		if got.Complete || findWork(t, got, "leaf.md").Acceptance.Status != "unknown" || !hasCode(got, "invalid_acceptance_metadata") {
			t.Fatalf("invalid approval profile: %+v", got.Diagnostics)
		}
	}
	withMetadata := acceptedLeafProject(t, "", "None\n", "custom: {nested: yes, numeric: .nan}\n")
	observed := orientProject(t, withMetadata)
	if !observed.Complete || findWork(t, observed, "leaf.md").Acceptance.Metadata["custom"] == nil {
		t.Fatalf("unknown acceptance metadata lost: %+v", observed.Diagnostics)
	}
	if _, err := json.Marshal(observed); err != nil {
		t.Fatalf("accepted unknown metadata must remain JSON encodable: %v", err)
	}
	project := acceptedLeafProject(t, "", "None\n", "")
	if err := os.Mkdir(filepath.Join(project, "directory"), 0700); err != nil {
		t.Fatal(err)
	}
	replaceFile(t, project, "acceptance.md", "[Evidence](evidence.txt)", "[Evidence](directory)")
	got := orientProject(t, project)
	if got.Complete || findWork(t, got, "leaf.md").Acceptance.Status != "unknown" || !hasCode(got, "source_unreadable") {
		t.Fatalf("unreadable evidence: %+v", got.Diagnostics)
	}
}

func TestAcceptanceMissingSectionsMalformedDigestsAndNewRequirementMembership(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{
		{"missing requirements", "## Requirements\nNone\n", ""},
		{"missing evidence", "## Evidence", "## Historical evidence"},
		{"prose requirements", "## Requirements\nNone", "## Requirements\nRequirements not gathered yet."},
		{"malformed subject digest", "ticketSHA256:", "unusedSHA256:"},
		{"malformed criteria digest", "criteriaSHA256:", "unusedSHA256:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := acceptedLeafProject(t, "", "None\n", "")
			replaceFile(t, project, "acceptance.md", tc.before, tc.after)
			got := orientProject(t, project)
			if got.Complete || findWork(t, got, "leaf.md").Acceptance.Status != "unknown" {
				t.Fatalf("missing metadata/sections: %+v", got.Diagnostics)
			}
		})
	}
	project := acceptedLeafProject(t, "", "None\n", "")
	writeFile(t, project, "new.txt", "New requirement.\n")
	replaceFile(t, project, "leaf.md", "## Acceptance\n", "## Context\n[New](new.txt)\n## Acceptance\n")
	got := orientProject(t, project)
	accepted := findWork(t, got, "leaf.md").Acceptance
	if !got.Complete || accepted.Status != "stale" || !hasFinding(accepted.Reasons, "requirement_membership_changed") || !hasFinding(accepted.Reasons, "ticket_fingerprint_changed") {
		t.Fatalf("new selected requirement: %+v diagnostics=%+v", accepted, got.Diagnostics)
	}
}

func TestTaskContextDoesNotExpandCurrentAcceptanceOrItsSnapshots(t *testing.T) {
	project := acceptedLeafProject(t, "", "None\n", "")
	got, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "leaf.md"})
	if err != nil || !got.Complete || len(got.Sources) != 1 || filepath.Base(got.Sources[0].Path) != "leaf.md" {
		t.Fatalf("task context changed selection: %+v error=%v", got, err)
	}
}

func reacceptLeaf(t *testing.T, project, requirements, metadataExtra string) {
	t.Helper()
	got := orientProject(t, project)
	leaf := findWork(t, got, "leaf.md")
	writeFile(t, project, "acceptance.md", acceptanceRecord(leaf, requirements, "- [Evidence](evidence.txt) `"+sourceHash("Recorded passing observations.\n")+"`\n", metadataExtra))
}

func hasCheckFinding(work orientation.WorkItem, name, code string) bool {
	for _, check := range work.Checks {
		if check.Name == name {
			return hasFinding(check.Reasons, code)
		}
	}
	return false
}

func orientProject(t *testing.T, project string) orientation.Result {
	t.Helper()
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func replaceFile(t *testing.T, project, path, before, after string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(project, path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), before) {
		t.Fatalf("missing replacement %q in %s", before, path)
	}
	writeFile(t, project, path, strings.Replace(string(data), before, after, 1))
}

func hasFinding(findings []orientation.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func acceptedLeafProject(t *testing.T, leafExtra, requirements, metadataExtra string) string {
	t.Helper()
	project := writeProject(t, map[string]string{
		"project.md":   committedManifest("dependent.md"),
		"leaf.md":      workItem("leaf", "completed", leafExtra),
		"dependent.md": strings.Replace(workItem("dependent", "unstarted", ""), "## Blocked by\nNone", "## Blocked by\n[Leaf](leaf.md)", 1),
		"evidence.txt": "Recorded passing observations.\n",
	})
	before, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	leaf := findWork(t, before, "leaf.md")
	writeFile(t, project, "acceptance.md", acceptanceRecord(leaf, requirements, "- [Evidence](evidence.txt) `"+sourceHash("Recorded passing observations.\n")+"`\n", metadataExtra))
	writeFile(t, project, "leaf.md", workItem("leaf", "completed", leafExtra)+"## Acceptance\n[Current](acceptance.md)\n")
	return project
}

func acceptanceRecord(work orientation.WorkItem, requirements, evidence, metadataExtra string) string {
	return fmt.Sprintf("---\ntype: Acceptance\nid: accepted-%s\ntitle: Accepted work\nprojectId: project\nworkItemId: %s\nactor: {kind: workflow, identity: test-agent}\ndecidedAt: '2026-09-16T09:00:00+02:00'\ntestedRevision: {origin: local-tests, revision: historical-revision}\nfingerprintVersion: 1\nticketSHA256: %s\ncriteriaSHA256: %s\n%s---\n## Requirements\n%s## Evidence\n%s", *work.ID, *work.ID, *work.TicketSHA256, *work.CriteriaSHA256, metadataExtra, requirements, evidence)
}

func sourceHash(text string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(text))) }

func writeFile(t *testing.T, project, path, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(project, path), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
