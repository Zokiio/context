package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestSetupCommandConnectsSubsequentReaderInvocation(t *testing.T) {
	root := scopeTempDir(t)
	cwd, home, records := filepath.Join(root, "checkout"), filepath.Join(root, "home"), filepath.Join(root, "records")
	for _, directory := range []string{cwd, home} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: connected\ntitle: Connected\n---\n")
	writeScopeFile(t, filepath.Join(records, "ticket.md"), "# Connected ticket\n")
	environment := scopeEnvironment(cwd, home)
	environment.Input = scopeForbiddenInput{t}
	environment.IsTerminal = func() bool { t.Fatal("specified setup must not inspect terminal state"); return false }
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), []string{"ctx", "setup", "--records", "../records"}, &stdout, &stderr, cli.Operations{}, environment)
	if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Project binding: create") {
		t.Fatalf("setup: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	status = cli.RunWithEnvironment(context.Background(), []string{"ctx", "context", "--ticket", "ticket.md"}, &stdout, &stderr, cli.Operations{Assemble: taskcontext.Assemble}, environment)
	if status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Connected ticket") {
		t.Fatalf("subsequent context: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	stdout.Reset()
	status = cli.RunWithEnvironment(context.Background(), []string{"ctx", "setup", "--records", "missing"}, &stdout, &stderr, cli.Operations{}, environment)
	if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("failed setup: status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}
