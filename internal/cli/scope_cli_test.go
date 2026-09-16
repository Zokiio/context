package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func scopeTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func writeScopeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scopeEnvironment(cwd, home string) cli.Environment {
	return cli.Environment{
		WorkingDirectory: func() (string, error) { return cwd, nil },
		HomeDirectory:    func() (string, error) { return home, nil },
	}
}

func TestReaderDiscoveryPassesSameExplicitScope(t *testing.T) {
	root := scopeTempDir(t)
	home := filepath.Join(root, "home")
	checkout := filepath.Join(root, "checkout")
	nested := filepath.Join(checkout, "src")
	records := filepath.Join(root, "records")
	docs := filepath.Join(root, "docs")
	extra := filepath.Join(root, "extra")
	for _, directory := range []string{home, nested, docs, extra} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: project\ntitle: Project\n---\n")
	writeScopeFile(t, filepath.Join(checkout, ".context", "config.md"), "---\ntype: ContextConfig\nversion: 1\nproject:\n  records: ../../records\n  allowSources: [../../docs]\n---\n")
	writeScopeFile(t, filepath.Join(home, ".context", "config.md"), fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nprojects:\n  - key: saved\n    alias: saved-project\n    directory: %q\n    records: %q\n    allowSources: [%q]\n---\n", checkout, records, docs))
	for _, command := range []string{"context", "orient"} {
		for _, selection := range []struct {
			name, cwd string
			flags     []string
		}{
			{"cwd", nested, nil},
			{"path", root, []string{"--project", "checkout/src"}},
			{"alias", root, []string{"--project", "@saved-project"}},
		} {
			t.Run(command+"/"+selection.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				var gotProject string
				var gotRoots []string
				called := false
				operations := cli.Operations{
					Assemble: func(_ context.Context, request taskcontext.Request) (taskcontext.Result, error) {
						called = true
						gotProject, gotRoots = request.ProjectDir, request.AllowedSourceDirs
						if request.TicketPath != "issues/task.md" {
							t.Fatalf("ticket path changed: %q", request.TicketPath)
						}
						return taskcontext.Result{Complete: true}, nil
					},
					Orient: func(_ context.Context, request orientation.Request) (orientation.Result, error) {
						called = true
						gotProject, gotRoots = request.ProjectDir, request.AllowedSourceDirs
						return orientation.Result{Complete: true}, nil
					},
				}
				args := append([]string{"ctx", command}, selection.flags...)
				if command == "context" {
					args = append(args, "--ticket", "issues/task.md")
				} else {
					args = append(args, "--json")
				}
				relativeExtra, err := filepath.Rel(selection.cwd, extra)
				if err != nil {
					t.Fatal(err)
				}
				args = append(args, "--allow-source", relativeExtra)
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, operations, scopeEnvironment(selection.cwd, home))
				if status != 0 || !called || stderr.Len() != 0 || gotProject != records || !reflect.DeepEqual(gotRoots, []string{docs, extra}) {
					t.Fatalf("status=%d called=%t project=%q roots=%v stderr=%q", status, called, gotProject, gotRoots, stderr.String())
				}
			})
		}
	}
}

type scopeForbiddenInput struct{ t *testing.T }

func (input scopeForbiddenInput) Read([]byte) (int, error) {
	input.t.Fatal("reader commands must not prompt or read stdin")
	return 0, errors.New("unexpected stdin read")
}

func TestBundleBypassesHomeLookupAndDiscovery(t *testing.T) {
	root := scopeTempDir(t)
	bundle := filepath.Join(root, "bundle")
	docs := filepath.Join(root, "docs")
	writeScopeFile(t, filepath.Join(bundle, "ticket.md"), "# Manifestless ticket\n\n## Context\n[Guide](../docs/guide.md)\n")
	writeScopeFile(t, filepath.Join(docs, "guide.md"), "# Authorized guide\n")
	writeScopeFile(t, filepath.Join(root, ".context", "config.md"), "Malformed local configuration.\n")
	writeScopeFile(t, filepath.Join(root, "home", ".context", "config.md"), "Malformed personal registry.\n")
	environment := scopeEnvironment(root, filepath.Join(root, "home"))
	environment.HomeDirectory = func() (string, error) {
		t.Fatal("--bundle must not look up home")
		return "", errors.New("home unavailable")
	}
	environment.Input = scopeForbiddenInput{t}
	environment.IsTerminal = func() bool {
		t.Fatal("reader commands must not inspect terminal state")
		return false
	}
	for _, command := range []string{"context", "orient"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := []string{"ctx", command, "--bundle", "bundle", "--allow-source", "docs"}
			wantStatus := 0
			if command == "context" {
				args = append(args, "--ticket", "ticket.md")
			} else {
				args = append(args, "--json")
				wantStatus = 1
			}
			status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Assemble: taskcontext.Assemble, Orient: orientation.Orient}, environment)
			if status != wantStatus || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) {
				t.Fatalf("status=%d stderr=%q stdout=%q", status, stderr.String(), stdout.String())
			}
			if command == "context" && !strings.Contains(stdout.String(), "Authorized guide") {
				t.Fatalf("direct context lost its explicitly authorized source: %s", stdout.String())
			}
			if command == "orient" && !strings.Contains(stdout.String(), `"project":null`) {
				t.Fatalf("direct orientation must retain its partial missing-manifest report: %s", stdout.String())
			}
		})
	}
}

func TestReadCommandsRejectInvalidSelectorsBeforeDiscovery(t *testing.T) {
	for _, command := range []string{"context", "orient"} {
		for _, flags := range [][]string{
			{"--project", ""}, {"--workspace="}, {"--bundle", ""},
			{"--project", "one", "--project=one"},
			{"--workspace=one", "--workspace", "two"},
			{"--bundle", "one", "--bundle=one"},
			{"--project", "one", "--workspace", "two"},
			{"--project", "one", "--bundle", "two"},
			{"--workspace", "one", "--bundle", "two"},
		} {
			t.Run(command+"/"+strings.Join(flags, " "), func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				environment := cli.Environment{WorkingDirectory: func() (string, error) {
					t.Fatal("invalid selectors must fail before filesystem discovery")
					return "", nil
				}}
				args := append([]string{"ctx", command}, flags...)
				if command == "context" {
					args = append(args, "--ticket", "ticket.md")
				}
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{}, environment)
				if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
					t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
				}
			})
		}
	}
}

func TestDiscoveryFailuresDoNotEmitReportsOrFallBack(t *testing.T) {
	for _, command := range []string{"context", "orient"} {
		for _, failure := range []string{"no scope", "manifestless project", "malformed registry", "broken selected mapping", "conflicting mappings"} {
			t.Run(command+"/"+failure, func(t *testing.T) {
				root := scopeTempDir(t)
				home := filepath.Join(root, "home")
				checkout := filepath.Join(root, "checkout")
				cwd := filepath.Join(checkout, "src")
				records := filepath.Join(root, "records")
				for _, directory := range []string{home, cwd} {
					if err := os.MkdirAll(directory, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: same-project\ntitle: Project\n---\n")
				sharedFile := filepath.Join(checkout, ".context", "config.md")
				personalFile := filepath.Join(home, ".context", "config.md")
				args := []string{"ctx", command}
				wantDetails := []string{}
				switch failure {
				case "no scope":
					wantDetails = []string{cwd, "physical ancestors", "ctx setup", "--bundle"}
				case "manifestless project":
					manifestless := filepath.Join(root, "manifestless")
					writeScopeFile(t, filepath.Join(manifestless, "ticket.md"), "# Direct ticket\n")
					args = append(args, "--project", manifestless)
					wantDetails = []string{manifestless, "--bundle"}
				case "malformed registry":
					writeScopeFile(t, sharedFile, fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nproject:\n  records: %q\n---\n", records))
					writeScopeFile(t, personalFile, "Malformed personal configuration.\n")
					wantDetails = []string{personalFile, "frontmatter"}
				case "broken selected mapping":
					writeScopeFile(t, filepath.Join(root, ".context", "config.md"), fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nproject:\n  records: %q\n---\n", records))
					writeScopeFile(t, sharedFile, "---\ntype: ContextConfig\nversion: 1\nproject:\n  records: ../missing-records\n---\n")
					wantDetails = []string{sharedFile, "project.records", "../missing-records", filepath.Join(checkout, "missing-records")}
				case "conflicting mappings":
					other := filepath.Join(root, "other-records")
					writeScopeFile(t, filepath.Join(other, "project.md"), "---\ntype: Project\nid: same-project\ntitle: Other checkout\n---\n")
					writeScopeFile(t, sharedFile, fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nproject:\n  records: %q\n---\n", records))
					writeScopeFile(t, personalFile, fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nprojects:\n  - key: conflicting\n    directory: %q\n    records: %q\n---\n", checkout, other))
					wantDetails = []string{"conflicting", sharedFile, personalFile}
				}
				if command == "context" {
					args = append(args, "--ticket", "ticket.md")
				} else {
					args = append(args, "--json")
				}
				var stdout, stderr bytes.Buffer
				operations := cli.Operations{
					Assemble: func(context.Context, taskcontext.Request) (taskcontext.Result, error) {
						t.Fatal("failed discovery must not invoke the reader")
						return taskcontext.Result{}, nil
					},
					Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
						t.Fatal("failed discovery must not invoke the reader")
						return orientation.Result{}, nil
					},
				}
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, operations, scopeEnvironment(cwd, home))
				if status != 2 || stdout.Len() != 0 {
					t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
				}
				for _, detail := range wantDetails {
					if !strings.Contains(stderr.String(), detail) {
						t.Errorf("missing diagnostic detail %q in %s", detail, stderr.String())
					}
				}
			})
		}
	}
}

func TestBareCommandDoesNotDiscoverOrPrompt(t *testing.T) {
	var stdout, stderr bytes.Buffer
	environment := cli.Environment{
		WorkingDirectory: func() (string, error) { t.Fatal("bare ctx must not discover scope"); return "", nil },
		HomeDirectory:    func() (string, error) { t.Fatal("bare ctx must not discover scope"); return "", nil },
		Input:            scopeForbiddenInput{t},
		IsTerminal:       func() bool { t.Fatal("bare ctx must not inspect terminal state"); return false },
	}
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx"}, &stdout, &stderr, cli.Operations{}, environment)
	if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "context") || !strings.Contains(stderr.String(), "orient") {
		t.Fatalf("bare command guidance: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func scopeFilesystemSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%s %d\n", info.Mode(), info.ModTime().UnixNano())
		if !entry.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += string(content)
		}
		snapshot[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestDiscoveredReadsPreserveFilesAndInvocationRootsStayTemporary(t *testing.T) {
	root := scopeTempDir(t)
	home := filepath.Join(root, "home")
	checkout := filepath.Join(root, "checkout")
	cwd := filepath.Join(checkout, "src")
	records := filepath.Join(root, "records")
	for _, directory := range []string{home, cwd} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\nNone\n## Current commitments\nNone\n## Open decisions\nNone\n")
	writeScopeFile(t, filepath.Join(records, "ticket.md"), "---\ntype: WorkItem\nid: task\ntitle: Task\ntriage: ready-for-agent\nexecution: unstarted\n---\n## Acceptance criteria\n- [ ] Read supporting documents.\n## Blocked by\nNone\n## Blocked by decisions\nNone\n## Context\n[Configured source](../docs/configured.md)\n[Invocation source](../extra/invocation.md)\n")
	writeScopeFile(t, filepath.Join(root, "docs", "configured.md"), "# Configured documentation\n")
	writeScopeFile(t, filepath.Join(root, "extra", "invocation.md"), "# Invocation documentation\n")
	configPath := filepath.Join(checkout, ".context", "config.md")
	writeScopeFile(t, configPath, "---\ntype: ContextConfig\nversion: 1\nproject:\n  records: ../../records\n  allowSources: [../../docs]\n---\n\nPreserve this authored configuration.\n")
	before := scopeFilesystemSnapshot(t, root)
	environment := scopeEnvironment(cwd, home)
	environment.Input = scopeForbiddenInput{t}
	environment.IsTerminal = func() bool {
		t.Fatal("reader commands must not inspect terminal state")
		return false
	}
	operations := cli.Operations{Assemble: taskcontext.Assemble, Orient: orientation.Orient}
	for _, command := range []string{"context", "orient"} {
		t.Run(command, func(t *testing.T) {
			args := []string{"ctx", command}
			if command == "context" {
				args = append(args, "--ticket", "ticket.md")
			} else {
				args = append(args, "--json")
			}
			withRoots := append(append([]string{}, args...), "--allow-source", "../../extra")
			var ordinary, ordinaryErrors bytes.Buffer
			status := cli.RunWithEnvironment(context.Background(), withRoots, &ordinary, &ordinaryErrors, operations, environment)
			if status != 0 || ordinaryErrors.Len() != 0 || !json.Valid(ordinary.Bytes()) {
				t.Fatalf("ordinary report: status=%d stderr=%q stdout=%q", status, ordinaryErrors.String(), ordinary.String())
			}
			var explained, explanation bytes.Buffer
			status = cli.RunWithEnvironment(context.Background(), append(withRoots, "--explain-scope"), &explained, &explanation, operations, environment)
			if status != 0 || explained.String() != ordinary.String() {
				t.Fatalf("scope explanation changed the project JSON: status=%d stderr=%q stdout=%q", status, explanation.String(), explained.String())
			}
			for _, fact := range []string{"Scope: project", checkout, records, configPath, "field project"} {
				if !strings.Contains(explanation.String(), fact) {
					t.Errorf("missing scope explanation %q in %s", fact, explanation.String())
				}
			}
			var denied, deniedErrors bytes.Buffer
			status = cli.RunWithEnvironment(context.Background(), args, &denied, &deniedErrors, operations, environment)
			if status != 1 || deniedErrors.Len() != 0 || !strings.Contains(denied.String(), "source_outside_scope") {
				t.Fatalf("invocation root was persisted or authorization changed: status=%d stderr=%q stdout=%q", status, deniedErrors.String(), denied.String())
			}
		})
	}
	if after := scopeFilesystemSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("reader commands changed fixture paths, contents, permissions, or modification times")
	}
}

func TestDirectBundleAndInvocationRootsFollowFilesystemPaths(t *testing.T) {
	for _, selector := range []string{"bundle", "allow-source"} {
		t.Run(selector, func(t *testing.T) {
			root := scopeTempDir(t)
			physical := filepath.Join(root, "physical")
			lexical := filepath.Join(root, "lexical")
			for _, directory := range []string{filepath.Join(physical, "anchor"), lexical} {
				if err := os.MkdirAll(directory, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(lexical, "link")
			if err := os.Symlink(filepath.Join(physical, "anchor"), link); err != nil {
				t.Fatal(err)
			}
			writeScopeFile(t, filepath.Join(physical, "records", "ticket.md"), "# Physical record store\n")
			writeScopeFile(t, filepath.Join(lexical, "records", "ticket.md"), "# Wrong lexical record store\n")
			writeScopeFile(t, filepath.Join(physical, "docs", "guide.md"), "# Physical documentation\n")
			writeScopeFile(t, filepath.Join(lexical, "docs", "guide.md"), "# Wrong lexical documentation\n")
			args := []string{"ctx", "context", "--ticket", "ticket.md"}
			wantText := "Physical record store"
			if selector == "bundle" {
				args = append(args, "--bundle", link+"/../records")
			} else {
				writeScopeFile(t, filepath.Join(physical, "records", "ticket.md"), "## Context\n[Physical documentation](../docs/guide.md)\n")
				args = append(args, "--bundle", filepath.Join(physical, "records"), "--allow-source", link+"/../docs")
				wantText = "# Physical documentation"
			}
			environment := scopeEnvironment(root, filepath.Join(root, "unused-home"))
			environment.HomeDirectory = func() (string, error) { t.Fatal("direct paths must not consult home"); return "", nil }
			for _, relative := range []bool{false, true} {
				invocation := append([]string{}, args...)
				if relative {
					invocation[len(invocation)-1] = strings.TrimPrefix(invocation[len(invocation)-1], root+string(filepath.Separator))
				}
				var stdout, stderr bytes.Buffer
				status := cli.RunWithEnvironment(context.Background(), invocation, &stdout, &stderr, cli.Operations{Assemble: taskcontext.Assemble}, environment)
				if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), wantText) || strings.Contains(stdout.String(), "Wrong lexical") {
					t.Fatalf("filesystem path was normalized before following symlinks: relative=%t status=%d stdout=%q stderr=%q", relative, status, stdout.String(), stderr.String())
				}
			}
		})
	}
}

func TestDirectPathsDoNotRepairMissingOrNonDirectoryParents(t *testing.T) {
	for _, selector := range []string{"bundle", "allow-source"} {
		for _, parentKind := range []string{"missing", "file"} {
			t.Run(selector+"/"+parentKind, func(t *testing.T) {
				root := scopeTempDir(t)
				bundle := filepath.Join(root, "bundle")
				docs := filepath.Join(root, "docs")
				writeScopeFile(t, filepath.Join(bundle, "ticket.md"), "## Context\n[Guide](../docs/guide.md)\n")
				writeScopeFile(t, filepath.Join(docs, "guide.md"), "# Guide\n")
				parent := filepath.Join(root, "invalid-parent")
				if parentKind == "file" {
					writeScopeFile(t, parent, "This is a file, not a directory.\n")
				}
				args := []string{"ctx", "context", "--ticket", "ticket.md"}
				if selector == "bundle" {
					args = append(args, "--bundle", parent+"/../bundle")
				} else {
					args = append(args, "--bundle", bundle, "--allow-source", parent+"/../docs")
				}
				var stdout, stderr bytes.Buffer
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Assemble: taskcontext.Assemble}, scopeEnvironment(root, filepath.Join(root, "unused-home")))
				if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), parent) {
					t.Fatalf("unavailable authored path was repaired: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
				}
			})
		}
	}
}
