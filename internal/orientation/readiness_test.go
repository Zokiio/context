package orientation_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
)

func TestEligibleWorkUsesCommitmentOrderAndSeparateStates(t *testing.T) {
	project := writeProject(t, map[string]string{
		"project.md":    committedManifest("b.md", "a.md", "active.md", "done.md", "cancelled.md", "triage.md", "deprecated.md"),
		"a.md":          workItem("a", "unstarted", ""),
		"b.md":          workItem("b", "unstarted", ""),
		"backlog.md":    workItem("c", "unstarted", ""),
		"active.md":     workItem("d", "in-progress", ""),
		"done.md":       workItem("e", "completed", ""),
		"cancelled.md":  workItem("f", "cancelled", ""),
		"triage.md":     strings.Replace(workItem("g", "unstarted", ""), "ready-for-agent", "needs-info", 1),
		"deprecated.md": strings.Replace(workItem("h", "unstarted", ""), "execution: unstarted", "execution: unstarted\nstatus: deprecated", 1),
	})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || got.Complete || !hasCode(got, "invalid_current_acceptance") || !got.InventoryComplete {
		t.Fatalf("evaluation: complete=%v diagnostics=%+v error=%v", got.Complete, got.Diagnostics, err)
	}
	if !reflect.DeepEqual(referenceIDs(got.Shortlist), []string{"b", "a", "h"}) {
		t.Fatalf("shortlist: %+v", got.Shortlist)
	}
	if !reflect.DeepEqual(referenceIDs(got.InProgress), []string{"d"}) || !reflect.DeepEqual(referenceIDs(got.Backlog), []string{"c"}) {
		t.Fatalf("groups: active=%v backlog=%v", referenceIDs(got.InProgress), referenceIDs(got.Backlog))
	}
	for _, work := range got.WorkItems {
		if work.Readiness != "ready" || len(work.Checks) != 4 {
			t.Fatalf("readiness: %+v", work)
		}
		for _, check := range work.Checks {
			if check.Status != "pass" || len(check.Reasons) == 0 {
				t.Fatalf("check: %+v", check)
			}
		}
		wantEligible := *work.ID == "a" || *work.ID == "b" || *work.ID == "h"
		if work.Eligible != wantEligible || (!wantEligible && len(work.ExclusionReasons) == 0) {
			t.Fatalf("eligibility: %+v", work)
		}
	}
}

func committedManifest(paths ...string) string {
	links := []string{}
	for _, path := range paths {
		links = append(links, "- [Work]("+path+")")
	}
	return strings.Replace(emptyManifest(), "## Current commitments\nNone", "## Current commitments\n"+strings.Join(links, "\n"), 1)
}

func referenceIDs(references []orientation.Reference) []string {
	ids := []string{}
	for _, r := range references {
		if r.ID != nil {
			ids = append(ids, *r.ID)
		}
	}
	return ids
}

func findWork(t *testing.T, result orientation.Result, path string) orientation.WorkItem {
	t.Helper()
	for _, work := range result.WorkItems {
		if filepath.Base(work.Source) == path {
			return work
		}
	}
	t.Fatalf("work %s absent", path)
	return orientation.WorkItem{}
}

func TestReadinessDistinguishesMissingEmptyAndUnsupportedConditions(t *testing.T) {
	base := workItem("work", "unstarted", "")
	for _, tc := range []struct {
		name, body, check, status, readiness string
		complete                             bool
		extra                                map[string]string
	}{
		{"empty criteria", strings.Replace(base, "- A useful result.\n", "", 1), "acceptance_criteria", "fail", "blocked", true, nil},
		{"missing criteria", strings.Replace(base, "## Acceptance criteria", "## Description", 1), "acceptance_criteria", "unknown", "unknown", false, nil},
		{"empty checkbox", strings.Replace(base, "- A useful result.", "- [ ]", 1), "acceptance_criteria", "fail", "blocked", true, nil},
		{"empty checked checkbox", strings.Replace(base, "- A useful result.", "- [x]", 1), "acceptance_criteria", "fail", "blocked", true, nil},
		{"link-only criterion", strings.Replace(base, "- A useful result.", "- [ ] [Authored requirement](ordinary-reference.md)", 1), "acceptance_criteria", "pass", "ready", true, nil},
		{"code criterion", strings.Replace(base, "- A useful result.", "- `[ ]`", 1), "acceptance_criteria", "pass", "ready", true, nil},
		{"heading without criterion", strings.Replace(base, "- A useful result.", "### Planned", 1), "acceptance_criteria", "fail", "blocked", true, nil},
		{"criteria references are prose", strings.Replace(base, "- A useful result.", "- Follow [the requirement][ordinary-reference].", 1), "acceptance_criteria", "pass", "ready", true, nil},
		{"missing selected context", base + "## Context\n[Required](missing.md)\n", "required_context", "unknown", "unknown", false, nil},
		{"prose-only spec", base + "## Spec\nWe will find a spec later.\n", "required_context", "unknown", "unknown", false, nil},
		{"missing dependency declaration", strings.Replace(base, "## Blocked by\nNone\n", "", 1), "dependencies", "unknown", "unknown", false, nil},
		{"malformed decisions declaration", strings.Replace(base, "## Blocked by decisions\nNone", "## Blocked by decisions\nAsk someone first.", 1), "blocking_decisions", "unknown", "unknown", false, nil},
		{"unaccepted completed dependency", strings.Replace(base, "## Blocked by\nNone", "## Blocked by\n[Previous](previous.md)", 1), "dependencies", "unknown", "unknown", false, map[string]string{"previous.md": workItem("previous", "completed", "")}},
		{"resolved decision", strings.Replace(base, "## Blocked by decisions\nNone", "## Blocked by decisions\n[Choice](decision.md)", 1), "blocking_decisions", "pass", "ready", true, map[string]string{"decision.md": "---\ntype: Decision\nid: choice\ntitle: Choice\ndecisionState: resolved\n---\n## Resolution\nAn answer.\n"}},
		{"known failure with unknown", strings.Replace(base, "- A useful result.\n", "", 1) + "## Context\n[Required](missing.md)\n", "required_context", "unknown", "blocked", false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"project.md": committedManifest("work.md"), "work.md": tc.body}
			for path, body := range tc.extra {
				files[path] = body
			}
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, files)})
			if err != nil {
				t.Fatal(err)
			}
			work := findWork(t, got, "work.md")
			if got.Complete != tc.complete || work.Readiness != tc.readiness || checkStatus(work, tc.check) != tc.status {
				t.Fatalf("complete=%v readiness=%s checks=%+v diagnostics=%+v", got.Complete, work.Readiness, work.Checks, got.Diagnostics)
			}
			if tc.readiness != "ready" && (work.Eligible || len(work.ExclusionReasons) == 0) {
				t.Fatalf("ineligible work: %+v", work)
			}
			if tc.name == "known failure with unknown" && checkStatus(work, "acceptance_criteria") != "fail" {
				t.Fatal("known failure lost")
			}
		})
	}
}

func checkStatus(work orientation.WorkItem, name string) string {
	for _, check := range work.Checks {
		if check.Name == name {
			return check.Status
		}
	}
	return ""
}

func TestPartialProjectInformationDoesNotEraseEstablishedReadiness(t *testing.T) {
	for _, tc := range []struct {
		name, manifest string
		extra          map[string]string
		eligible       bool
		excluded       string
	}{
		{"missing goals", strings.Replace(committedManifest("work.md"), "## Goals\n", "", 1), nil, true, ""},
		{"incomplete commitments", strings.Replace(committedManifest("work.md"), "## Open decisions", "## Current commitments\nChoose something later.\n## Open decisions", 1), nil, true, ""},
		{"incomplete inventory", committedManifest("work.md"), map[string]string{"broken.md": "---\ntype: [broken\n---\n"}, false, "inventory_incomplete"},
		{"invalid project identity", strings.Replace(committedManifest("work.md"), "type: Project", "type: Other", 1), nil, false, "project_identity_unknown"},
		{"unrelated duplicate", committedManifest("work.md"), map[string]string{"duplicate-a.md": workItem("duplicate", "unstarted", ""), "duplicate-b.md": workItem("duplicate", "unstarted", "")}, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"project.md": tc.manifest, "work.md": workItem("work", "unstarted", ""), "uncommitted.md": workItem("later", "unstarted", "")}
			for path, body := range tc.extra {
				files[path] = body
			}
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, files)})
			if err != nil {
				t.Fatal(err)
			}
			work := findWork(t, got, "work.md")
			if got.Complete || work.Readiness != "ready" || work.Eligible != tc.eligible || work.Committed == nil || !*work.Committed {
				t.Fatalf("work=%+v complete=%v diagnostics=%+v", work, got.Complete, got.Diagnostics)
			}
			if tc.excluded != "" && !hasExclusion(work, tc.excluded) {
				t.Fatalf("suppression reason missing: %+v", work.ExclusionReasons)
			}
			if tc.name == "incomplete commitments" {
				other := findWork(t, got, "uncommitted.md")
				if other.Committed != nil || other.Eligible || !hasExclusion(other, "commitment_unknown") {
					t.Fatalf("unproven membership: %+v", other)
				}
			}
			if tc.name == "unrelated duplicate" {
				for _, path := range []string{"duplicate-a.md", "duplicate-b.md"} {
					duplicate := findWork(t, got, path)
					if duplicate.Readiness != "unknown" || duplicate.Eligible {
						t.Fatalf("ambiguous readiness: %+v", duplicate)
					}
				}
			}
		})
	}
}

func TestMissingPickupStateDoesNotChangeReadiness(t *testing.T) {
	for _, tc := range []struct{ name, body, manifest, reason string }{
		{"execution", strings.Replace(workItem("work", "unstarted", ""), "execution: unstarted\n", "", 1), committedManifest("work.md"), "execution_unknown"},
		{"invalid execution", strings.Replace(workItem("work", "unstarted", ""), "execution: unstarted", "execution: ' unstarted '", 1), committedManifest("work.md"), "execution_unknown"},
		{"triage", strings.Replace(workItem("work", "unstarted", ""), "triage: ready-for-agent\n", "", 1), committedManifest("work.md"), "triage_unknown"},
		{"commitment", workItem("work", "unstarted", ""), strings.Replace(emptyManifest(), "## Current commitments\nNone\n", "", 1), "commitment_unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, map[string]string{"project.md": tc.manifest, "work.md": tc.body})})
			if err != nil {
				t.Fatal(err)
			}
			work := findWork(t, got, "work.md")
			if got.Complete || work.Readiness != "ready" || work.Eligible || !hasExclusion(work, tc.reason) {
				t.Fatalf("work=%+v complete=%v diagnostics=%+v", work, got.Complete, got.Diagnostics)
			}
		})
	}
}

func hasExclusion(work orientation.WorkItem, code string) bool {
	for _, reason := range work.ExclusionReasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}

func TestRequiredContextUsesAuthorizedAvailableDocuments(t *testing.T) {
	for _, tc := range []struct {
		name, content, link, code string
		allowed, ready            bool
	}{
		{"available", "Requirements. [Ordinary prose](not-selected.md)", "../requirements.txt", "", true, true},
		{"malformed", "---\ntitle: [broken\n---\n", "../requirements.txt", "invalid_frontmatter", true, false},
		{"encoding", string([]byte{0xff}), "../requirements.txt", "invalid_source_encoding", true, false},
		{"unauthorized", "Private requirements", "../requirements.txt", "source_outside_scope", false, false},
		{"missing", "Requirements", "../absent.txt", "source_missing", true, false},
		{"remote", "Requirements", "https://example.test/requirements", "unsupported_source", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			project := filepath.Join(parent, "bundle")
			if err := os.Mkdir(project, 0700); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{"project.md": committedManifest("work.md"), "work.md": workItem("work", "unstarted", "## Context\n[Requirements]("+tc.link+")\n")}
			for path, content := range files {
				if err := os.WriteFile(filepath.Join(project, path), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(parent, "requirements.txt"), []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			request := orientation.Request{ProjectDir: project}
			if tc.allowed {
				request.AllowedSourceDirs = []string{parent}
			}
			got, err := orientation.Orient(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			work := findWork(t, got, "work.md")
			if got.Complete != tc.ready || !got.InventoryComplete || work.Eligible != tc.ready || (checkStatus(work, "required_context") == "pass") != tc.ready {
				t.Fatalf("context: work=%+v complete=%v inventory=%v diagnostics=%+v", work, got.Complete, got.InventoryComplete, got.Diagnostics)
			}
			if tc.code != "" && !hasCode(got, tc.code) {
				t.Fatalf("missing diagnostic %s: %+v", tc.code, got.Diagnostics)
			}
		})
	}
}

func TestRelationshipEmptySentinelPunctuation(t *testing.T) {
	for _, tc := range []struct {
		text  string
		valid bool
	}{
		{"None", true},
		{"None.", true},
		{"  None.  ", true},
		{"None..", false},
		{"none.", false},
		{"None. Ask later.", false},
		{"[Missing][unresolved]", false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			for _, section := range []string{"Blocked by", "Blocked by decisions", "Spec", "Context"} {
				t.Run(section, func(t *testing.T) {
					body := workItem("work", "unstarted", "## Spec\nNone\n## Context\nNone\n")
					body = strings.Replace(body, "## "+section+"\nNone\n", "## "+section+"\n"+tc.text+"\n", 1)
					got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, map[string]string{
						"project.md": committedManifest("work.md"), "work.md": body,
					})})
					if err != nil {
						t.Fatal(err)
					}
					work := findWork(t, got, "work.md")
					if got.Complete != tc.valid || work.Eligible != tc.valid {
						t.Fatalf("complete=%v eligible=%v diagnostics=%+v", got.Complete, work.Eligible, got.Diagnostics)
					}
				})
			}
			manifest := "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\nObserve.\n## Current commitments\n" + tc.text + "\n## Open decisions\n" + tc.text + "\n"
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, map[string]string{"project.md": manifest})})
			if err != nil || got.Complete != tc.valid {
				t.Fatalf("project complete=%v diagnostics=%+v err=%v", got.Complete, got.Diagnostics, err)
			}
		})
	}
}
