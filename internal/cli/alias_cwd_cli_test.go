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
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestReaderAliasesWorkWhenCwdLookupFails(t *testing.T) {
	root := scopeTempDir(t)
	home, records := filepath.Join(root, "home"), filepath.Join(root, "records")
	docs, extra := filepath.Join(root, "docs"), filepath.Join(root, "extra")
	for _, directory := range []string{docs, extra} {
		if err := os.Mkdir(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: saved\ntitle: Saved project\n---\n")
	writeScopeFile(t, filepath.Join(home, ".context", "config.md"), fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nprojects:\n  - key: saved\n    alias: saved\n    directory: %q\n    records: %q\n    allowSources: [%q]\nworkspaces:\n  - key: team\n    alias: team\n    id: saved-workspace\n    title: Saved workspace\n    members: []\n---\n", filepath.Join(root, "missing-checkout"), records, docs))
	environment := cli.Environment{
		WorkingDirectory: func() (string, error) { return "", errors.New("cwd was removed") },
		HomeDirectory:    func() (string, error) { return home, nil },
	}
	for _, command := range []string{"context", "orient"} {
		for _, addRoot := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/absolute-root=%t", command, addRoot), func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				var gotRecords string
				var gotRoots []string
				operations := cli.Operations{
					Assemble: func(_ context.Context, request taskcontext.Request) (taskcontext.Result, error) {
						gotRecords, gotRoots = request.ProjectDir, request.AllowedSourceDirs
						return taskcontext.Result{Complete: true}, nil
					},
					Orient: func(_ context.Context, request orientation.Request) (orientation.Result, error) {
						gotRecords, gotRoots = request.ProjectDir, request.AllowedSourceDirs
						return orientation.Result{Complete: true}, nil
					},
				}
				args := []string{"ctx", command, "--project", "@saved"}
				if command == "context" {
					args = append(args, "--ticket", "ticket.md")
				} else {
					args = append(args, "--json")
				}
				wantRoots := []string{docs}
				if addRoot {
					args = append(args, "--allow-source", extra)
					wantRoots = append(wantRoots, extra)
				}
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, operations, environment)
				if status != 0 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) || gotRecords != records || !reflect.DeepEqual(gotRoots, wantRoots) {
					t.Fatalf("status=%d records=%q roots=%v stdout=%q stderr=%q", status, gotRecords, gotRoots, stdout.String(), stderr.String())
				}
			})
		}
	}
	t.Run("workspace retains project selection guidance", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "context", "--workspace", "@team", "--ticket", "ticket.md", "--explain-scope"}, &stdout, &stderr, cli.Operations{}, environment)
		if status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "Scope: workspace") || !strings.Contains(stderr.String(), "saved-workspace") || !strings.Contains(stderr.String(), "requires project scope") || strings.Contains(stderr.String(), "cwd was removed") {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
		}
	})
}

func TestReadCommandsKeepCwdErrorsWhenPathsNeedCwd(t *testing.T) {
	for _, command := range []string{"context", "orient"} {
		for _, flags := range [][]string{
			nil,
			{"--project", "relative"}, {"--project", "/absolute"},
			{"--workspace", "relative"}, {"--workspace", "/absolute"},
			{"--bundle", "relative"}, {"--bundle", "/absolute"},
			{"--project", "./@literal"},
			{"--project", "@saved", "--allow-source", "relative"},
			{"--workspace", "@team", "--allow-source", "relative"},
		} {
			t.Run(command+"/"+strings.Join(flags, " "), func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				environment := cli.Environment{
					WorkingDirectory: func() (string, error) { return "", errors.New("cwd was removed") },
					HomeDirectory: func() (string, error) {
						t.Fatal("a required cwd must not be replaced with home")
						return "", nil
					},
				}
				args := append([]string{"ctx", command}, flags...)
				if command == "context" {
					args = append(args, "--ticket", "ticket.md")
				}
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{}, environment)
				if status != 2 || stdout.Len() != 0 || stderr.String() != "read working directory: cwd was removed\n" {
					t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
				}
			})
		}
	}
}
