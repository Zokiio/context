package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/discovery"
)

func setupCLIEnvironment(t *testing.T) (Environment, string, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home, cwd, records := filepath.Join(root, "home"), filepath.Join(root, "checkout"), filepath.Join(root, "records")
	for _, path := range []string{home, cwd, records} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(records, "project.md"), []byte("---\ntype: Project\nid: setup-cli\ntitle: Setup CLI\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return Environment{
		WorkingDirectory: func() (string, error) { return cwd, nil },
		HomeDirectory:    func() (string, error) { return home, nil },
		Input:            strings.NewReader(""),
		IsTerminal:       func() bool { return false },
	}, cwd, records
}

func runSetupCLI(environment Environment, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := setupCommand(&stdout, &stderr, environment).Run(context.Background(), append([]string{"setup"}, args...))
	return stdout.String(), stderr.String(), err
}

func TestSetupCLIPrintsExactDryRunWithoutCreatingConfiguration(t *testing.T) {
	environment, cwd, records := setupCLIEnvironment(t)
	stdout, stderr, err := runSetupCLI(environment, "--records", records, "--allow-source", cwd, "--dry-run")
	if err != nil || stderr != "" {
		t.Fatalf("dry-run: %v, %s", err, stderr)
	}
	_, document, found := strings.Cut(stdout, "Proposed configuration:\n")
	home, _ := environment.HomeDirectory()
	plan, err := discovery.PrepareSetup(context.Background(), discovery.SetupRequest{Cwd: cwd, Home: home, Records: records, AllowSources: []string{cwd}})
	if err != nil || !found || document != plan.Document() {
		t.Fatalf("dry-run does not show exact proposal: %s, %v", stdout, err)
	}
	for _, value := range []string{"Project binding: create", cwd, records, filepath.Join(cwd, ".context", "config.md"), "Allowed sources:"} {
		if !strings.Contains(stdout, value) {
			t.Fatalf("summary lacks %q: %s", value, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(cwd, ".context")); !os.IsNotExist(err) {
		t.Fatalf("dry-run changed configuration directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".context")); !os.IsNotExist(err) {
		t.Fatalf("dry-run changed personal configuration directory: %v", err)
	}
}

type setupOutputWriter struct {
	write func([]byte) (int, error)
}

func (w setupOutputWriter) Write(data []byte) (int, error) { return w.write(data) }

func TestSetupCLIPrintsSummaryBeforeWritingWithoutPrompting(t *testing.T) {
	environment, cwd, records := setupCLIEnvironment(t)
	environment.IsTerminal = func() bool { t.Fatal("fully specified setup checked terminal"); return true }
	path := filepath.Join(cwd, ".context", "config.md")
	var output bytes.Buffer
	writes := 0
	writer := setupOutputWriter{write: func(data []byte) (int, error) {
		writes++
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("configuration written before summary: %v", err)
		}
		return output.Write(data)
	}}
	var stderr bytes.Buffer
	err := setupCommand(writer, &stderr, environment).Run(context.Background(), []string{"setup", "--records", records})
	if err != nil || writes != 1 || stderr.Len() != 0 {
		t.Fatalf("setup: %v, writes %d, stderr %s", err, writes, stderr.String())
	}
	if _, err := discovery.ReadConfig(path, discovery.SharedConfig); err != nil {
		t.Fatal(err)
	}
}

func TestSetupCLIOutputFailurePreventsAllWrites(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "short"}[short], func(t *testing.T) {
			environment, cwd, records := setupCLIEnvironment(t)
			writer := setupOutputWriter{write: func([]byte) (int, error) {
				if short {
					return 1, nil
				}
				return 0, errors.New("output unavailable")
			}}
			var stderr bytes.Buffer
			err := setupCommand(writer, &stderr, environment).Run(context.Background(), []string{"setup", "--records", records})
			if err == nil || !strings.Contains(err.Error(), "write setup output") {
				t.Fatalf("output failure = %v", err)
			}
			if _, err := os.Stat(filepath.Join(cwd, ".context")); !os.IsNotExist(err) {
				t.Fatalf("failed summary changed files: %v", err)
			}
		})
	}
}

func TestSetupCLIGuidesOnlyTerminalInput(t *testing.T) {
	environment, cwd, records := setupCLIEnvironment(t)
	stdout, stderr, err := runSetupCLI(environment)
	if err == nil || !strings.Contains(err.Error(), "--records") || stdout != "" || stderr != "" {
		t.Fatalf("noninteractive invocation: %v, %q, %q", err, stdout, stderr)
	}
	environment.IsTerminal = func() bool { return true }
	environment.Input = strings.NewReader(records + "\n\nyes\nmine\n" + cwd + "\n\n")
	stdout, stderr, err = runSetupCLI(environment)
	if err != nil || !strings.Contains(stdout, "Personal alias: mine") || !strings.Contains(stderr, "Existing records directory:") || !strings.Contains(stderr, "Binding directory") {
		t.Fatalf("guided invocation: %v, %s, %s", err, stdout, stderr)
	}
	home, _ := environment.HomeDirectory()
	config, err := discovery.ReadConfig(filepath.Join(home, ".context", "config.md"), discovery.PersonalConfig)
	if err != nil || config.Projects[0].Alias != "mine" || config.Projects[0].Directory.Canonical != cwd || len(config.Projects[0].AllowSources) != 1 {
		t.Fatalf("guided registration = %#v, %v", config, err)
	}
}

func TestSetupCLIGuidedEOFAndInvalidAnswerNeverWrite(t *testing.T) {
	for _, answer := range []string{"", "not-a-boolean\n"} {
		environment, cwd, records := setupCLIEnvironment(t)
		environment.IsTerminal = func() bool { return true }
		environment.Input = strings.NewReader(records + "\n\n" + answer)
		_, _, err := runSetupCLI(environment)
		if err == nil {
			t.Fatal("incomplete guided input accepted")
		}
		if _, err := os.Stat(filepath.Join(cwd, ".context")); !os.IsNotExist(err) {
			t.Fatalf("incomplete guided setup wrote files: %v", err)
		}
	}
}

func TestSetupCLIRejectsInvalidFlagsBeforePrompts(t *testing.T) {
	for _, args := range [][]string{
		{"--records="}, {"--directory="}, {"--alias="}, {"--records", "a", "--records", "b"},
		{"--directory", "a", "--directory", "b"}, {"--personal", "--alias", "a", "--alias", "b"},
		{"--alias", "mine"}, {"--replace", "--replace"}, {"extra"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			environment, _, _ := setupCLIEnvironment(t)
			environment.IsTerminal = func() bool { t.Fatal("invalid flags prompted"); return true }
			stdout, stderr, err := runSetupCLI(environment, args...)
			if err == nil || stdout != "" || stderr != "" {
				t.Fatalf("invalid flags: %v, %q, %q", err, stdout, stderr)
			}
		})
	}
}

func TestSetupCLIReplacementReportsRemovedRootsAndRetainsAlias(t *testing.T) {
	environment, cwd, records := setupCLIEnvironment(t)
	if _, _, err := runSetupCLI(environment, "--records", records, "--personal", "--alias", "mine", "--allow-source", cwd); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runSetupCLI(environment, "--records", records, "--personal", "--replace")
	if err != nil || stderr != "" || !strings.Contains(stdout, "Removes allowed source: "+cwd) || !strings.Contains(stdout, "Personal alias: mine") || !strings.Contains(stdout, "Project binding: update") {
		t.Fatalf("replacement: %v, %s, %s", err, stdout, stderr)
	}
	stdout, _, err = runSetupCLI(environment, "--records", records, "--personal")
	if err != nil || !strings.Contains(stdout, "Project binding: unchanged") {
		t.Fatalf("idempotent invocation: %v, %s", err, stdout)
	}
}

var _ io.Writer = setupOutputWriter{}
