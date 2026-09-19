package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
)

type navigationCLIResult struct {
	SchemaVersion int
	Kind          string
	Complete      bool
	WorkStatus    string
	Members       []struct {
		Key           string
		Directory     string
		Records       string
		Availability  string
		SelectionArgs []string
	}
	Diagnostics []json.RawMessage
}

func TestWorkspaceCLIEnumeratesAndCopiesOnlyMemberRoots(t *testing.T) {
	root := scopeTempDir(t)
	home, checkout := filepath.Join(root, "home"), filepath.Join(root, "checkout")
	records := filepath.Join(root, "record store's files")
	docs, otherDocs := filepath.Join(root, "first docs"), filepath.Join(root, "other docs")
	for _, directory := range []string{checkout, records, docs, otherDocs} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A member's project identity does not participate in workspace enumeration.
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: same-id\ntitle: Same project\n---\n")
	missingCheckout := filepath.Join(root, "missing-checkout")
	brokenRecords := root + "/missing/../record store's files"
	config := filepath.Join(checkout, ".context", "config.md")
	source := fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nworkspace:\n  id: navigation\n  title: Navigation workspace\n  members:\n    - key: z-first\n      records: %q\n      directory: %q\n      allowSources: [%q]\n    - key: a-second\n      records: %q\n      directory: %q\n      allowSources: [%q]\n    - key: unavailable\n      records: %q\n---\n", records, missingCheckout, docs, records, filepath.Join(root, "other-checkout"), otherDocs, brokenRecords)
	writeScopeFile(t, config, source)
	var stdout, stderr bytes.Buffer
	operations := cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		t.Fatal("workspace navigation invoked project orientation")
		return orientation.Result{}, nil
	}}
	args := []string{"ctx", "orient", "--json", "--allow-source", filepath.Join(root, "invocation-only-root")}
	status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, operations, scopeEnvironment(checkout, home))
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%s stderr=%s", status, stdout.String(), stderr.String())
	}
	var result navigationCLIResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SchemaVersion != 1 || result.Kind != "workspace-navigation" || !result.Complete || result.WorkStatus != "unevaluated" || len(result.Members) != 3 || len(result.Diagnostics) != 1 {
		t.Fatalf("navigation result = %#v", result)
	}
	for i, key := range []string{"z-first", "a-second", "unavailable"} {
		if result.Members[i].Key != key {
			t.Fatalf("member order = %#v", result.Members)
		}
	}
	firstArgs := []string{"ctx", "orient", "--bundle", records, "--allow-source", docs}
	secondArgs := []string{"ctx", "orient", "--bundle", records, "--allow-source", otherDocs}
	if !reflect.DeepEqual(result.Members[0].SelectionArgs, firstArgs) || !reflect.DeepEqual(result.Members[1].SelectionArgs, secondArgs) {
		t.Fatalf("member roots leaked or changed: %#v", result.Members)
	}
	if result.Members[0].Directory != missingCheckout || result.Members[0].Availability != "available" || result.Members[2].Availability != "unavailable" || result.Members[2].Records != brokenRecords {
		t.Fatalf("availability or original paths changed: %#v", result.Members)
	}
	stdout.Reset()
	stderr.Reset()
	called := false
	copiedOperations := cli.Operations{Orient: func(_ context.Context, request orientation.Request) (orientation.Result, error) {
		called = true
		if request.ProjectDir != records || !reflect.DeepEqual(request.AllowedSourceDirs, []string{docs}) {
			t.Fatalf("copied command selected %#v", request)
		}
		return orientation.Result{Complete: true}, nil
	}}
	status = cli.RunWithEnvironment(context.Background(), append(firstArgs, "--json"), &stdout, &stderr, copiedOperations, scopeEnvironment(root, home))
	if status != 0 || !called {
		t.Fatalf("copied member command failed: %d, %s", status, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	status = cli.RunWithEnvironment(context.Background(), result.Members[2].SelectionArgs, &stdout, &stderr, operations, scopeEnvironment(root, home))
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "missing/../") {
		t.Fatalf("copied broken spelling was repaired: %d, %s, %s", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	status = cli.RunWithEnvironment(context.Background(), []string{"ctx", "orient"}, &stdout, &stderr, operations, scopeEnvironment(checkout, home))
	if status != 0 || !strings.Contains(stdout.String(), "Work status was not evaluated.") || !strings.Contains(stdout.String(), "ctx orient --bundle") {
		t.Fatalf("human navigation = %d, %s, %s", status, stdout.String(), stderr.String())
	}
	after, err := os.ReadFile(config)
	if err != nil || string(after) != source {
		t.Fatalf("navigation changed configuration: %v", err)
	}
}

func TestWorkspaceCLIEmptyEnumerationSucceeds(t *testing.T) {
	root := scopeTempDir(t)
	home, checkout := filepath.Join(root, "home"), filepath.Join(root, "checkout")
	writeScopeFile(t, filepath.Join(checkout, ".context", "config.md"), "---\ntype: ContextConfig\nversion: 1\nworkspace: {id: empty, title: Empty workspace, members: []}\n---\n")
	for _, format := range []string{"text", "detail", "json"} {
		t.Run(format, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := []string{"ctx", "orient"}
			if format == "json" {
				args = append(args, "--json")
			} else if format == "detail" {
				args = append(args, "--detail")
			}
			status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{}, scopeEnvironment(checkout, home))
			if status != 0 || stderr.Len() != 0 {
				t.Fatalf("empty navigation = %d, %s, %s", status, stdout.String(), stderr.String())
			}
			if format == "text" && !strings.Contains(stdout.String(), "Members: none") {
				t.Fatalf("empty text = %s", stdout.String())
			}
			if format == "json" && !strings.Contains(stdout.String(), "\"members\":[]") {
				t.Fatalf("empty JSON = %s", stdout.String())
			}
		})
	}
	for _, args := range [][]string{
		{"ctx", "orient", "--detail", "--detail"},
		{"ctx", "orient", "--detail", "--json"},
	} {
		var stdout, stderr bytes.Buffer
		status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{}, scopeEnvironment(checkout, home))
		if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "--detail") {
			t.Fatalf("workspace accepted invalid detail flags %v: status=%d stdout=%q stderr=%q", args, status, stdout.String(), stderr.String())
		}
	}
}

func TestWorkspaceCLIRejectsInvalidAndConflictingDeclarations(t *testing.T) {
	for _, kind := range []string{"invalid", "conflicting"} {
		t.Run(kind, func(t *testing.T) {
			root := scopeTempDir(t)
			home, checkout := filepath.Join(root, "home"), filepath.Join(root, "checkout")
			local, personal := filepath.Join(checkout, ".context", "config.md"), filepath.Join(home, ".context", "config.md")
			source := "---\ntype: ContextConfig\nversion: 1\nworkspace: {id: shared, title: Shared, members: []}\n---\n"
			if kind == "invalid" {
				source = "---\ntype: ContextConfig\nversion: 1\nworkspace: {id: shared, members: []}\n---\n"
			} else {
				writeScopeFile(t, personal, fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nworkspaces:\n  - key: saved\n    directory: %q\n    id: shared\n    title: Different title\n    members: []\n---\n", checkout))
			}
			writeScopeFile(t, local, source)
			var stdout, stderr bytes.Buffer
			status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "orient", "--json"}, &stdout, &stderr, cli.Operations{}, scopeEnvironment(checkout, home))
			if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), local) {
				t.Fatalf("bad workspace = %d, %s, %s", status, stdout.String(), stderr.String())
			}
			if kind == "conflicting" && !strings.Contains(stderr.String(), personal) {
				t.Fatalf("conflict lost personal origin: %s", stderr.String())
			}
		})
	}
}

func TestWorkspaceCLIAliasWithoutWorkingDirectory(t *testing.T) {
	root := scopeTempDir(t)
	home := filepath.Join(root, "home")
	records := filepath.Join(root, "unavailable-records")
	writeScopeFile(t, filepath.Join(home, ".context", "config.md"), fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nworkspaces:\n  - key: saved\n    alias: team\n    id: team-workspace\n    title: Team workspace\n    members:\n      - key: unavailable\n        records: %q\n---\n", records))
	environment := scopeEnvironment(root, home)
	environment.WorkingDirectory = func() (string, error) {
		return "", errors.New("working directory was removed")
	}
	operations := cli.Operations{Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
		t.Fatal("workspace alias invoked project orientation")
		return orientation.Result{}, nil
	}}
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := []string{"ctx", "orient", "--workspace", "@team"}
			if format == "json" {
				args = append(args, "--json")
			}
			status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, operations, environment)
			if status != 0 || stderr.Len() != 0 {
				t.Fatalf("workspace alias with unavailable cwd = %d, %s, %s", status, stdout.String(), stderr.String())
			}
			if format == "text" {
				for _, detail := range []string{"Team workspace", "team-workspace", "Work status was not evaluated.", "unavailable", records} {
					if !strings.Contains(stdout.String(), detail) {
						t.Fatalf("human navigation omitted %q: %s", detail, stdout.String())
					}
				}
			} else {
				var result navigationCLIResult
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Kind != "workspace-navigation" || !result.Complete || result.WorkStatus != "unevaluated" || len(result.Members) != 1 || result.Members[0].Availability != "unavailable" {
					t.Fatalf("workspace alias navigation = %+v", result)
				}
			}
		})
	}
}
