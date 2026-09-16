package cli_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/cli"
)

func TestDiscoveryFilesRejectFIFOsWithoutWaitingForAWriter(t *testing.T) {
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("mkfifo is unavailable on this platform")
	}
	for _, scenario := range []struct {
		name, target string
		args         []string
	}{
		{"shared config orient", "shared", []string{"orient"}},
		{"shared config context", "shared", []string{"context", "--ticket", "ticket.md"}},
		{"personal config orient", "personal", []string{"orient"}},
		{"shared config setup", "shared", []string{"setup", "--records", "records", "--dry-run"}},
		{"personal config setup", "personal", []string{"setup", "--personal", "--records", "records", "--dry-run"}},
		{"project marker orient", "marker", []string{"orient", "--project", "records"}},
		{"project marker context", "marker", []string{"context", "--project", "records", "--ticket", "ticket.md"}},
		{"project marker setup", "marker", []string{"setup", "--records", "records", "--dry-run"}},
	} {
		for _, symlink := range []bool{false, true} {
			name := scenario.name
			if symlink {
				name += " symlink"
			}
			t.Run(name, func(t *testing.T) {
				root := scopeTempDir(t)
				home, records := filepath.Join(root, "home"), filepath.Join(root, "records")
				if err := os.Mkdir(home, 0o755); err != nil {
					t.Fatal(err)
				}
				marker := filepath.Join(records, "project.md")
				writeScopeFile(t, marker, "---\ntype: Project\nid: project\ntitle: Project\n---\n")
				target := filepath.Join(root, ".context", "config.md")
				switch scenario.target {
				case "personal":
					target = filepath.Join(home, ".context", "config.md")
				case "marker":
					target = marker
					if err := os.Remove(marker); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				fifo := target
				if symlink {
					fifo = filepath.Join(root, "pipe")
				}
				if output, err := exec.Command(mkfifo, fifo).CombinedOutput(); err != nil {
					t.Fatalf("create FIFO: %v: %s", err, output)
				}
				if symlink {
					if err := os.Symlink(fifo, target); err != nil {
						t.Fatal(err)
					}
				}
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
					if got.status != 2 || got.stdout != "" || !strings.Contains(got.stderr, target) || !strings.Contains(got.stderr, "regular file") {
						t.Fatalf("file rejection: %+v", got)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("discovery waited for a FIFO writer")
				}
			})
		}
	}
}
