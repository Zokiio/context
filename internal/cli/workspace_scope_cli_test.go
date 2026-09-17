package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestWorkspaceScopeSelectionNeverInvokesProjectReaders(t *testing.T) {
	root := scopeTempDir(t)
	home := filepath.Join(root, "home")
	checkout := filepath.Join(root, "checkout")
	cwd := filepath.Join(checkout, "src")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	sharedFile := filepath.Join(checkout, ".context", "config.md")
	personalFile := filepath.Join(home, ".context", "config.md")
	memberRecords := filepath.Join(root, "unavailable-member")
	writeScopeFile(t, sharedFile, fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nworkspace:\n  id: workspace-1\n  title: Selected workspace\n  members:\n    - key: sole-member\n      records: %q\n---\n", memberRecords))
	writeScopeFile(t, personalFile, fmt.Sprintf("---\ntype: ContextConfig\nversion: 1\nworkspaces:\n  - key: saved\n    alias: saved-workspace\n    directory: %q\n    id: workspace-1\n    title: Selected workspace\n    members:\n      - key: sole-member\n        records: %q\n---\n", checkout, memberRecords))
	for _, command := range []string{"context", "orient"} {
		for _, selection := range []struct {
			name, cwd, origin string
			flags             []string
		}{
			{"cwd", cwd, sharedFile, nil},
			{"path", root, sharedFile, []string{"--workspace", "checkout/src"}},
			{"alias", root, personalFile, []string{"--workspace", "@saved-workspace"}},
		} {
			t.Run(command+"/"+selection.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				operations := cli.Operations{
					Assemble: func(context.Context, taskcontext.Request) (taskcontext.Result, error) {
						t.Fatal("workspace scope must not choose its sole member for task context")
						return taskcontext.Result{}, nil
					},
					Orient: func(context.Context, orientation.Request) (orientation.Result, error) {
						t.Fatal("workspace scope must not invoke project orientation")
						return orientation.Result{}, nil
					},
				}
				args := append([]string{"ctx", command, "--explain-scope"}, selection.flags...)
				if command == "context" {
					args = append(args, "--ticket", "ticket.md")
				} else {
					args = append(args, "--json")
				}
				status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, operations, scopeEnvironment(selection.cwd, home))
				if status != 2 || stdout.Len() != 0 {
					t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
				}
				for _, fact := range []string{"Scope: workspace", "workspace-1", checkout, selection.origin, "--project", "--bundle"} {
					if !strings.Contains(stderr.String(), fact) {
						t.Errorf("missing selected workspace or selection guidance %q in %s", fact, stderr.String())
					}
				}
				if command == "context" && !strings.Contains(stderr.String(), "requires project scope") {
					t.Fatalf("context must explain its project requirement: %s", stderr.String())
				}
			})
		}
	}
}
