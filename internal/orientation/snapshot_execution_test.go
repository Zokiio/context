package orientation_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
)

func TestSnapshotMissingExecution(t *testing.T) {
	for _, tc := range []struct {
		name, metadata string
		valid          bool
	}{
		{"https", "sourceURL: https://tracker.example/issues/1\n", true},
		{"http", "sourceURL: http://tracker.example/issues/1\n", true},
		{"uppercase scheme", "sourceURL: HTTPS://tracker.example/issues/1\n", true},
		{"native", "", false},
		{"relative", "sourceURL: issues/1\n", false},
		{"protocol relative", "sourceURL: //tracker.example/issues/1\n", false},
		{"no host", "sourceURL: https:///issues/1\n", false},
		{"port only", "sourceURL: https://:443/issues/1\n", false},
		{"wrong scheme", "sourceURL: ftp://tracker.example/1\n", false},
		{"malformed", "sourceURL: https://tracker.example/%zz\n", false},
		{"empty marker", "sourceURL: ''\n", false},
		{"null marker", "sourceURL: null\n", false},
		{"nonstring marker", "sourceURL: [https://tracker.example/1]\n", false},
		{"empty execution", "sourceURL: https://tracker.example/1\nexecution: ''\n", false},
		{"null execution", "sourceURL: https://tracker.example/1\nexecution: null\n", false},
		{"invalid execution", "sourceURL: https://tracker.example/1\nexecution: open\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(workItem("work", "unstarted", ""), "execution: unstarted\n", tc.metadata, 1)
			got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, map[string]string{"project.md": committedManifest("work.md"), "work.md": body})})
			if err != nil {
				t.Fatal(err)
			}
			work := findWork(t, got, "work.md")
			if got.Complete != tc.valid || hasCode(got, "invalid_profile") == tc.valid {
				t.Fatalf("complete=%v diagnostics=%+v", got.Complete, got.Diagnostics)
			}
			if work.Execution != nil || work.Eligible || len(got.Shortlist) != 0 {
				t.Fatalf("unknown execution became eligible: %+v", work)
			}
			found := false
			for _, reason := range work.ExclusionReasons {
				if reason.Code == "execution_unknown" {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing execution_unknown: %+v", work.ExclusionReasons)
			}
		})
	}
}

func TestSnapshotExecutionDoesNotBypassReadiness(t *testing.T) {
	for _, broken := range []bool{false, true} {
		body := strings.Replace(workItem("work", "unstarted", ""), "execution: unstarted\n", "execution: unstarted\nsourceURL: https://tracker.example/1\n", 1)
		if broken {
			body += "## Context\n[Missing](missing.md)\n"
		}
		got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, map[string]string{"project.md": committedManifest("work.md"), "work.md": body})})
		if err != nil {
			t.Fatal(err)
		}
		if work := findWork(t, got, "work.md"); work.Eligible == broken {
			t.Fatalf("broken=%v work=%+v", broken, work)
		}
	}
}

func TestSnapshotWithoutExecutionCannotSatisfyDependency(t *testing.T) {
	leaf := strings.Replace(workItem("leaf", "unstarted", ""), "execution: unstarted\n", "sourceURL: https://tracker.example/1\n", 1)
	got, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: writeProject(t, map[string]string{
		"project.md": committedManifest("dependent.md"), "leaf.md": leaf, "dependent.md": blockedWork("dependent", "unstarted", "leaf.md"),
	})})
	if err != nil {
		t.Fatal(err)
	}
	work := findWork(t, got, "dependent.md")
	if hasCode(got, "invalid_profile") || work.Eligible || checkStatus(work, "dependencies") != "unknown" || !hasCheckFinding(work, "dependencies", "dependency_execution_unknown") {
		t.Fatalf("work=%+v diagnostics=%+v", work, got.Diagnostics)
	}
}
