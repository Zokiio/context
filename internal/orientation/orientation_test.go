package orientation_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
)

func TestEmptyProjectOverview(t *testing.T) {
	project := writeProject(t, map[string]string{"project.md": "---\ntype: Project\nid: independent-project\ntitle: Independent project\n---\n## Goals\nKeep authored records local.\n## Current commitments\nNone\n## Open decisions\nNone\n"})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Complete || !got.InventoryComplete || got.SchemaVersion != 1 || got.Project == nil || got.Project.ID != "independent-project" || got.Project.Title != "Independent project" {
		t.Fatalf("overview: %+v", got)
	}
	if len(got.Goals) != 1 || got.Goals[0].Text != "Keep authored records local." || len(got.Sources) != 1 || got.Sources[0].Path != filepath.Join(project, "project.md") {
		t.Fatalf("authored goals and manifest source: %+v", got)
	}
	if got.CurrentCommitments == nil || got.WorkItems == nil || got.Shortlist == nil || got.InProgress == nil || got.Backlog == nil || got.Decisions == nil || got.Diagnostics == nil {
		t.Fatal("collection fields must be arrays")
	}
}

func TestWorkInventoryAndCommitments(t *testing.T) {
	project := writeProject(t, map[string]string{
		"project.md":    "---\ntype: Project\nid: project\ntitle: Independent project\n---\n## Goals\nFirst authored goal.\n## Current commitments\n- [Chosen](z-first.md)\n## Goals\n[More detail](goal.md#direction)\n## Current commitments\n- [Active](a-second.md)\n## Open decisions\n- [Choice](decision.md)\n",
		"z-first.md":    workItem("a", "unstarted", "## Spec\n[Requirements](spec.md)\n"),
		"a-second.md":   workItem("b", "in-progress", ""),
		"backlog.md":    workItem("c", "unstarted", ""),
		"done.md":       workItem("d", "completed", ""),
		"goal.md":       "Authored direction. [Do not select](absent.md)\n",
		"spec.md":       "Requirements.\n",
		"decision.md":   "---\ntype: Decision\nid: decision\ntitle: A choice\ndecisionState: open\n---\n",
		"acceptance.md": "---\ntype: Acceptance\nid: acceptance\ntitle: A decision\n---\n",
		"other.md":      "---\ntype: Other\nid: other\ntitle: Other\n---\n",
		".gitignore":    "backlog.md\n",
	})
	t.Chdir(t.TempDir())
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete || !got.InventoryComplete || len(got.WorkItems) != 4 || len(got.Shortlist) != 0 {
		t.Fatalf("inventory: %+v", got)
	}
	if len(got.Goals) != 2 || len(got.Goals[1].References) != 1 || got.Goals[1].References[0].Link != "goal.md#direction" {
		t.Fatalf("goals: %+v", got.Goals)
	}
	if len(got.CurrentCommitments) != 2 || *got.CurrentCommitments[0].ID != "a" || *got.CurrentCommitments[1].ID != "b" {
		t.Fatalf("commitments: %+v", got.CurrentCommitments)
	}
	for i, want := range []string{"a", "b", "c", "d"} {
		work := got.WorkItems[i]
		if work.ID == nil || *work.ID != want || work.Committed == nil || *work.Committed != (i < 2) || work.Readiness != "unknown" || work.Eligible || len(work.Checks) != 4 || work.TicketSHA256 != nil {
			t.Fatalf("work: %+v", work)
		}
		for _, check := range work.Checks {
			if check.Status != "unknown" {
				t.Fatalf("unsupported check: %+v", check)
			}
		}
		if work.Metadata["custom"] == nil {
			t.Fatal("unknown metadata lost")
		}
	}
	if len(got.InProgress) != 1 || *got.InProgress[0].ID != "b" || len(got.Backlog) != 1 || *got.Backlog[0].ID != "c" {
		t.Fatalf("progress: %+v", got)
	}
	if len(got.WorkItems[0].Specifications) != 1 || len(got.Decisions) != 1 {
		t.Fatalf("references: %+v", got)
	}
	if got.Sources[0].Path != filepath.Join(project, "project.md") || len(got.Sources) != 10 {
		t.Fatalf("sources: %+v", got.Sources)
	}
	for _, d := range got.Diagnostics {
		if d.Code != "unsupported_check" {
			t.Fatalf("unexpected diagnostic: %+v", d)
		}
	}
	for _, source := range got.Sources {
		if strings.HasSuffix(source.Path, "absent.md") {
			t.Fatal("ordinary document prose expanded")
		}
	}
}

func workItem(id, execution, extra string) string {
	return "---\ntype: WorkItem\nid: " + id + "\ntitle: Work " + id + "\ntriage: ready-for-agent\nexecution: " + execution + "\ncustom: {nested: [one, 2]}\n---\n## Acceptance criteria\n- A useful result.\n## Blocked by\nNone\n## Blocked by decisions\nNone\n" + extra
}

func TestIncompleteProfilesAndAmbiguousIdentitiesRetainFacts(t *testing.T) {
	project := writeProject(t, map[string]string{
		"project.md":         "---\ntype: Project\nid: project\ntitle: Partial project\n---\n## Goals\nKnown direction.\n## Current commitments\n[Known](good.md)\n## Current commitments\nWe should probably work on another ticket.\n",
		"good.md":            workItem("known", "unstarted", ""),
		"not-chosen.md":      workItem("other", "unstarted", ""),
		"missing-profile.md": "---\ntype: WorkItem\ntitle: Incomplete\nstatus: stable\nexecution: stable\n---\n",
		"duplicate.md":       workItem("duplicate", "unstarted", ""),
		"decision.md":        "---\ntype: Decision\nid: duplicate\ntitle: Ambiguous choice\n---\n",
		"broken.md":          "---\ntitle: [broken\n---\n",
	})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if got.Complete || got.InventoryComplete || got.Project == nil || got.Project.CommitmentsKnown || got.Project.OpenDecisionsKnown || len(got.WorkItems) != 4 || len(got.Goals) != 1 {
		t.Fatalf("partial facts: %+v", got)
	}
	var known, other, incomplete *orientation.WorkItem
	for i := range got.WorkItems {
		w := &got.WorkItems[i]
		if w.ID == nil {
			incomplete = w
		} else if *w.ID == "known" {
			known = w
		} else if *w.ID == "other" {
			other = w
		}
	}
	if known == nil || known.Committed == nil || !*known.Committed || other == nil || other.Committed != nil || incomplete == nil || incomplete.Execution != nil || incomplete.Triage != nil {
		t.Fatalf("unknown states: %+v", got.WorkItems)
	}
	for _, code := range []string{"invalid_frontmatter", "invalid_profile", "duplicate_identity", "missing_required_section", "invalid_relationship_section"} {
		if !hasCode(got, code) {
			t.Fatalf("missing %s: %+v", code, got.Diagnostics)
		}
	}
	if _, err := json.Marshal(got); err != nil {
		t.Fatal(err)
	}
}

func hasCode(result orientation.Result, code string) bool {
	for _, d := range result.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestRepeatedGoalsRetainTheirOwnReferences(t *testing.T) {
	manifest := strings.Replace(emptyManifest(), "## Goals\n", "## Goals\n[Shared](goal.txt)\n## Goals\n[Shared](goal.txt)\n", 1)
	project := writeProject(t, map[string]string{"project.md": manifest, "goal.txt": "One shared source.\n"})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || !got.Complete || len(got.Goals) != 2 {
		t.Fatalf("goals=%+v diagnostics=%+v err=%v", got.Goals, got.Diagnostics, err)
	}
	for _, goal := range got.Goals {
		if len(goal.References) != 1 || goal.References[0].Link != "goal.txt" {
			t.Fatalf("section references multiplied: %+v", goal)
		}
	}
	if len(got.Sources) != 2 || len(got.Sources[1].Reasons) != 1 {
		t.Fatalf("source reasons should remain distinct: %+v", got.Sources)
	}
}

func TestManifestFailuresKeepKnownWork(t *testing.T) {
	for _, tc := range []struct {
		name, manifest, code string
		inventory            bool
	}{
		{"missing", "", "source_missing", true},
		{"wrong type", "# A plain document\n", "invalid_profile", true},
		{"malformed", "---\ntype: [broken\n---\n", "invalid_frontmatter", false},
		{"missing title", "---\ntype: Project\nid: project\n---\n", "invalid_profile", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{"work.md": workItem("work", "unstarted", "")}
			if tc.manifest != "" {
				files["project.md"] = tc.manifest
			}
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, files)})
			if err != nil || got.Complete || got.Project != nil || got.InventoryComplete != tc.inventory || len(got.WorkItems) != 1 || !hasCode(got, tc.code) {
				t.Fatalf("manifest failure: project=%+v inventory=%v work=%d diagnostics=%+v error=%v", got.Project, got.InventoryComplete, len(got.WorkItems), got.Diagnostics, err)
			}
		})
	}
}

func TestDuplicateProjectIdentityIsUnavailable(t *testing.T) {
	project := writeProject(t, map[string]string{"project.md": emptyManifest(), "duplicate.md": workItem("project", "unstarted", "")})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || got.Complete || got.Project != nil || len(got.WorkItems) != 1 || !got.WorkItems[0].IdentityAmbiguous || !hasCode(got, "duplicate_identity") {
		t.Fatalf("identity: %+v, %v", got, err)
	}
}

func TestOperationFailuresAreSeparateFromPartialReports(t *testing.T) {
	project := writeProject(t, map[string]string{"project.md": emptyManifest()})
	for _, request := range []orientation.Request{
		{}, {ProjectDir: filepath.Join(project, "missing")}, {ProjectDir: filepath.Join(project, "project.md")},
		{ProjectDir: project, AllowedSourceDirs: []string{""}}, {ProjectDir: project, AllowedSourceDirs: []string{filepath.Join(project, "missing")}},
		{ProjectDir: project, MaxFiles: -1}, {ProjectDir: project, MaxBytes: -1},
	} {
		if _, err := orientation.Orient(context.Background(), request); err == nil {
			t.Fatalf("invalid request accepted: %+v", request)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := orientation.Orient(ctx, orientation.Request{ProjectDir: project}); err != context.Canceled {
		t.Fatalf("cancel: %v", err)
	}
}

func TestManifestSectionsUseMarkdownStructure(t *testing.T) {
	manifest := "---\ntype: Project\nid: project\ntitle: Project\n---\nGoals\n-----\nFirst authored goal.\n### More\n[Shared][goal]\n```md\n## Current commitments\n[Fake](absent.md)\n```\n## Current commitments\nNone\n## Open decisions\nNone\n\n[goal]: goal.txt\n"
	project := writeProject(t, map[string]string{"project.md": manifest, "goal.txt": "Goal source"})
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project})
	if err != nil || !got.Complete || len(got.Goals) != 1 || len(got.Goals[0].References) != 1 || len(got.CurrentCommitments) != 0 {
		t.Fatalf("Markdown sections: goals=%+v diagnostics=%+v error=%v", got.Goals, got.Diagnostics, err)
	}
}

func writeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return project
}
