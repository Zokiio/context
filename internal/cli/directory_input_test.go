package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/cli"
)

func TestDirectoryInputsRejectFIFOsWithoutWaitingForAWriter(t *testing.T) {
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("mkfifo is unavailable on this platform")
	}
	root := scopeTempDir(t)
	home, records, fifo := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "pipe")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}
	writeScopeFile(t, filepath.Join(records, "project.md"), "---\ntype: Project\nid: project\ntitle: Project\n---\n")
	if output, err := exec.Command(mkfifo, fifo).CombinedOutput(); err != nil {
		t.Fatalf("create FIFO: %v: %s", err, output)
	}
	for _, scenario := range []struct {
		name string
		args []string
	}{
		{"direct bundle", []string{"orient", "--bundle", fifo}},
		{"invocation root", []string{"orient", "--bundle", records, "--allow-source", fifo}},
		{"project start", []string{"orient", "--project", fifo}},
		{"setup records", []string{"setup", "--records", fifo}},
		{"setup directory", []string{"setup", "--records", records, "--directory", fifo}},
		{"setup root", []string{"setup", "--records", records, "--allow-source", fifo}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			type result struct {
				status         int
				stdout, stderr string
			}
			done := make(chan result, 1)
			go func() {
				var stdout, stderr bytes.Buffer
				status := cli.RunWithEnvironment(context.Background(), append([]string{"ctx"}, scenario.args...), &stdout, &stderr, cli.Operations{}, scopeEnvironment(root, home))
				done <- result{status, stdout.String(), stderr.String()}
			}()
			select {
			case got := <-done:
				if got.status != 2 || got.stdout != "" || got.stderr == "" {
					t.Fatalf("directory rejection: %+v", got)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("directory input waited for a FIFO writer")
			}
		})
	}
}
