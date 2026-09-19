package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/resumption"
)

func TestResumeDirectBundleTextAndJSON(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	environment := scopeEnvironment(root, home)
	environment.Input = scopeForbiddenInput{t}
	for _, asJSON := range []bool{false, true} {
		args := []string{"ctx", "resume", "--bundle", "records", "--checkout", "checkout", "--ticket", "task.md"}
		if asJSON {
			args = append(args, "--json")
		}
		var stdout, stderr bytes.Buffer
		status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, environment)
		if status != 0 || stderr.Len() != 0 {
			t.Fatalf("json=%t status=%d stdout=%q stderr=%q", asJSON, status, stdout.String(), stderr.String())
		}
		if asJSON {
			var report resumption.Result
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Kind != "task-resumption" || !report.Complete || report.Recovery.Status != "absent" || report.Scope.WorkingDirectory != checkout {
				t.Fatalf("JSON report: %+v, %v", report, err)
			}
		} else {
			for _, want := range []string{"Task resumption", "Recovery: absent", "Baseline: unavailable", "Current task:", "Current context sources:"} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("text report missing %q: %s", want, stdout.String())
				}
			}
		}
	}
}

func TestResumeCheckoutSelection(t *testing.T) {
	root := scopeTempDir(t)
	home, checkout, nested := filepath.Join(root, "home"), filepath.Join(root, "checkout"), filepath.Join(root, "checkout", "nested")
	records, override := filepath.Join(root, "records"), filepath.Join(root, "override")
	for _, directory := range []string{home, nested, override} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	writeScopeFile(t, filepath.Join(checkout, ".context", "config.md"), fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nproject:\n  records: %q\n---\n", records))
	for _, tc := range []struct {
		name    string
		cwd     string
		args    []string
		working string
	}{
		{"binding", nested, nil, checkout},
		{"relative override", nested, []string{"--checkout", "../../override"}, override},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			called := false
			operation := func(_ context.Context, request resumption.Request) (resumption.Result, error) {
				called = true
				if request.RecordsDirectory != records || request.WorkingDirectory != tc.working || request.TicketPath != "task.md" {
					t.Fatalf("request: %+v", request)
				}
				return emptyResumeResult(true), nil
			}
			args := append([]string{"ctx", "resume", "--ticket", "task.md"}, tc.args...)
			status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Resume: operation}, scopeEnvironment(tc.cwd, home))
			if status != 0 || !called || stderr.Len() != 0 {
				t.Fatalf("status=%d called=%t stderr=%q", status, called, stderr.String())
			}
		})
	}
}

func TestResumeAllowsRepeatedSourceRoots(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	first, second := filepath.Join(root, "first"), filepath.Join(root, "second")
	for _, directory := range []string{home, checkout, first, second} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	called := false
	operation := func(_ context.Context, request resumption.Request) (resumption.Result, error) {
		called = true
		want := []string{first, second}
		if fmt.Sprint(request.AllowedSourceDirs) != fmt.Sprint(want) {
			t.Fatalf("allowed roots: got %v want %v", request.AllowedSourceDirs, want)
		}
		return emptyResumeResult(true), nil
	}
	args := []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md", "--allow-source", first, "--allow-source", second}
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Resume: operation}, scopeEnvironment(root, home))
	if status != 0 || !called || stderr.Len() != 0 {
		t.Fatalf("status=%d called=%t stdout=%q stderr=%q", status, called, stdout.String(), stderr.String())
	}
}

func TestResumeBareProjectMarkerUsesMarkerDirectory(t *testing.T) {
	root := scopeTempDir(t)
	home, records := filepath.Join(root, "home"), filepath.Join(root, "records")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	writeResumeCLIProject(t, records, true)
	called := false
	operation := func(_ context.Context, request resumption.Request) (resumption.Result, error) {
		called = true
		if request.RecordsDirectory != records || request.WorkingDirectory != records {
			t.Fatalf("marker scope: %+v", request)
		}
		return emptyResumeResult(true), nil
	}
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "resume", "--ticket", "task.md"}, &stdout, &stderr, cli.Operations{Resume: operation}, scopeEnvironment(records, home))
	if status != 0 || !called || stderr.Len() != 0 {
		t.Fatalf("status=%d called=%t stdout=%q stderr=%q", status, called, stdout.String(), stderr.String())
	}
}

func TestResumeDirectBundleRequiresCheckout(t *testing.T) {
	root := scopeTempDir(t)
	home, records := filepath.Join(root, "home"), filepath.Join(root, "records")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	writeResumeCLIProject(t, records, true)
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "resume", "--bundle", records, "--ticket", "task.md"}, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, scopeEnvironment(root, home))
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "requires --checkout") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestResumeAliasWithUnavailableBindingNeedsAccessibleCheckout(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	writeScopeFile(t, filepath.Join(home, ".context", "config.md"), fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nprojects:\n  - key: saved\n    alias: saved\n    directory: %q\n    records: %q\n---\n", filepath.Join(root, "missing"), records))
	environment := cli.Environment{
		WorkingDirectory: func() (string, error) { return "", errors.New("cwd was removed") },
		HomeDirectory:    func() (string, error) { return home, nil },
	}
	for _, tc := range []struct {
		name       string
		checkout   string
		wantStatus int
	}{
		{"inferred missing", "", 2},
		{"absolute override", checkout, 0},
		{"relative without cwd", "checkout", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"ctx", "resume", "--project", "@saved", "--ticket", "task.md"}
			if tc.checkout != "" {
				args = append(args, "--checkout", tc.checkout)
			}
			var stdout, stderr bytes.Buffer
			status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, environment)
			if status != tc.wantStatus {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			if status == 0 && stderr.Len() != 0 {
				t.Fatalf("successful override: %q", stderr.String())
			}
		})
	}
}

func TestResumeRejectsRepeatedScalarFlagsAndInvalidArguments(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	base := []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md", "--json"}
	duplicates := map[string][]string{
		"ticket":          {"--ticket", "task.md"},
		"project":         {"--project", root, "--project", root},
		"workspace":       {"--workspace", root, "--workspace", root},
		"bundle":          {"--bundle", records},
		"checkout":        {"--checkout", checkout},
		"json":            {"--json"},
		"explain-scope":   {"--explain-scope", "--explain-scope"},
		"max-files":       {"--max-files", "2", "--max-files", "2"},
		"max-bytes":       {"--max-bytes", "20", "--max-bytes", "20"},
		"max-cache-files": {"--max-cache-files", "2", "--max-cache-files", "2"},
		"max-cache-bytes": {"--max-cache-bytes", "20", "--max-cache-bytes", "20"},
	}
	for name, extra := range duplicates {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := cli.RunWithEnvironment(context.Background(), append(append([]string{}, base...), extra...), &stdout, &stderr, cli.Operations{}, scopeEnvironment(root, home))
			if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "may only be specified once") {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
	for name, args := range map[string][]string{
		"empty ticket":          {"--ticket="},
		"empty checkout":        {"--ticket", "task.md", "--checkout="},
		"position":              {"--ticket", "task.md", "extra"},
		"zero cache files":      {"--ticket", "task.md", "--max-cache-files", "0"},
		"negative cache bytes":  {"--ticket", "task.md", "--max-cache-bytes", "-1"},
		"malformed cache files": {"--ticket", "task.md", "--max-cache-files", "many"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			full := append([]string{"ctx", "resume", "--bundle", records, "--checkout", checkout}, args...)
			status := cli.RunWithEnvironment(context.Background(), full, &stdout, &stderr, cli.Operations{}, scopeEnvironment(root, home))
			if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestResumeRejectsWorkspaceButReportsIncompleteHistory(t *testing.T) {
	root := scopeTempDir(t)
	home, checkout, records := filepath.Join(root, "home"), filepath.Join(root, "checkout"), filepath.Join(root, "records")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	writeScopeFile(t, filepath.Join(checkout, ".context", "config.md"), "---\ntype: ContextConfig\nversion: 1\nworkspace:\n  id: workspace\n  title: Workspace\n  members: []\n---\n")
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "resume", "--ticket", "task.md"}, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, scopeEnvironment(checkout, home))
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "requires project scope") {
		t.Fatalf("workspace status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}

	observations := filepath.Join(checkout, ".context-cache", "resume-v1", shaKey("project"), shaKey("task"), "observations", "entry")
	if err := os.MkdirAll(observations, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(filepath.Dir(observations), "second"), 0700); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	status = cli.RunWithEnvironment(context.Background(), []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md"}, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, scopeEnvironment(root, home))
	if status != 1 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Recovery: unknown; graph: incomplete") {
		t.Fatalf("nonempty status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestResumePartialIdentityReturnsStatusOne(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, false)
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md", "--json"}, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, scopeEnvironment(root, home))
	if status != 1 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) || !strings.Contains(stdout.String(), `"status":"unknown"`) || !strings.Contains(stdout.String(), `"taskId":null`) {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestResumeMalformedCacheReturnsAttributedPartialJSON(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	if err := os.WriteFile(filepath.Join(checkout, ".context-cache"), []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md", "--json"}, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, scopeEnvironment(root, home))
	if status != 1 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	var result resumption.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Complete || !result.Context.Complete || !result.Orientation.Complete || result.Recovery.GraphStatus != "incomplete" || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "recovery_source_omitted" {
		t.Fatalf("partial cache report: %+v", result)
	}
}

func TestResumeOperationAndOutputFailuresReturnStatusTwo(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, directory := range []string{home, checkout} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	args := []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md", "--json"}
	for _, tc := range []struct {
		name      string
		writer    io.Writer
		operation cli.ResumeFunc
	}{
		{"operation", new(bytes.Buffer), func(context.Context, resumption.Request) (resumption.Result, error) {
			return resumption.Result{}, errors.New("resume failed")
		}},
		{"writer", failingWriter{}, func(context.Context, resumption.Request) (resumption.Result, error) {
			return emptyResumeResult(true), nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			status := cli.RunWithEnvironment(context.Background(), args, tc.writer, &stderr, cli.Operations{Resume: tc.operation}, scopeEnvironment(root, home))
			if status != 2 || stderr.Len() == 0 {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
		})
	}
}

func emptyResumeResult(complete bool) resumption.Result {
	return resumption.Result{SchemaVersion: 1, Kind: "task-resumption", Complete: complete,
		Recovery:   resumption.Recovery{Observations: []resumption.Observation{}, Candidates: []string{}},
		Comparison: resumption.Comparison{Candidates: []resumption.CandidateComparison{}}, Diagnostics: []resumption.Diagnostic{}}
}

func writeResumeCLIProject(t *testing.T, records string, identity bool) {
	t.Helper()
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\nGoal.\n## Current commitments\n[Task](task.md)\n## Open decisions\nNone\n")
	id := ""
	if identity {
		id = "id: task\n"
	}
	writeScopeFile(t, filepath.Join(records, "task.md"), "---\ntype: WorkItem\n"+id+"title: Task\ntriage: ready-for-agent\nexecution: unstarted\n---\n## Acceptance criteria\n- Resume it.\n## Blocked by\nNone\n## Blocked by decisions\nNone\n")
}

func shaKey(value string) string {
	// Keep this helper independent of unexported production namespace code.
	return fmt.Sprintf("%x", sha256Sum([]byte(value)))
}

func sha256Sum(value []byte) [32]byte {
	return sha256.Sum256(value)
}
