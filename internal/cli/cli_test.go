package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/resumption"
	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"ctx": func() {
		os.Exit(cli.Run(context.Background(), os.Args, os.Stdout, os.Stderr, cli.Operations{
			Assemble: taskcontext.Assemble,
			Orient:   orientation.Orient,
			Resume:   resumption.Resume,
		}))
	}})
}

func TestCLI(t *testing.T) {
	testscript.Run(t, testscript.Params{Dir: "testdata", Setup: func(environment *testscript.Env) error {
		home := filepath.Join(environment.WorkDir, "home")
		if err := os.MkdirAll(home, 0o755); err != nil {
			return err
		}
		environment.Setenv("HOME", home)
		return nil
	}, Cmds: map[string]func(*testscript.TestScript, bool, []string){"exit": func(ts *testscript.TestScript, neg bool, args []string) {
		if neg || len(args) < 2 {
			ts.Fatalf("usage: exit code command args...")
		}
		want, err := strconv.Atoi(args[0])
		ts.Check(err)
		err = ts.Exec(args[1], args[2:]...)
		code := 0
		if err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				ts.Fatalf("%v", err)
			}
			code = exitError.ExitCode()
		}
		if code != want {
			ts.Fatalf("exit code %d, want %d", code, want)
		}
	}, "jsonresult": func(ts *testscript.TestScript, neg bool, args []string) {
		if neg || len(args) < 2 || len(args) > 3 {
			ts.Fatalf("usage: jsonresult file true|false [traversalComplete]")
		}
		var result taskcontext.Result
		decoder := json.NewDecoder(bytes.NewBufferString(ts.ReadFile(args[0])))
		if err := decoder.Decode(&result); err != nil {
			ts.Fatalf("%v", err)
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			ts.Fatalf("unexpected trailing JSON: %v", err)
		}
		traversalComplete := args[1] == "true"
		if len(args) == 3 {
			traversalComplete = args[2] == "true"
		}
		if result.SchemaVersion != 1 || result.Complete != (args[1] == "true") || result.TraversalComplete != traversalComplete || result.Sources == nil || result.Diagnostics == nil {
			ts.Fatalf("unexpected JSON: %+v", result)
		}
	}, "jsonorientation": checkOrientationJSON}})
}

func checkOrientationJSON(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 2 {
		ts.Fatalf("usage: jsonorientation file true|false")
	}
	var result orientation.Result
	decoder := json.NewDecoder(bytes.NewBufferString(ts.ReadFile(args[0])))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		ts.Fatalf("%v", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		ts.Fatalf("unexpected trailing JSON: %v", err)
	}
	if result.SchemaVersion != 1 || result.Complete != (args[1] == "true") ||
		result.Goals == nil || result.CurrentCommitments == nil || result.WorkItems == nil ||
		result.Shortlist == nil || result.InProgress == nil || result.Backlog == nil ||
		result.Decisions == nil || result.Sources == nil || result.Diagnostics == nil {
		ts.Fatalf("unexpected orientation JSON: %+v", result)
	}
	for _, work := range result.WorkItems {
		if work.Dependencies == nil {
			ts.Fatalf("dependency edges must be an array: %+v", work)
		}
		for _, edge := range work.Dependencies {
			if edge.Reasons == nil {
				ts.Fatalf("dependency edge reasons must be an array: %+v", edge)
			}
		}
		if acceptance := work.Acceptance; acceptance != nil {
			if acceptance.HumanApprovals == nil || acceptance.Requirements == nil || acceptance.Evidence == nil || acceptance.Reasons == nil {
				ts.Fatalf("acceptance lists must be arrays: %+v", acceptance)
			}
			for _, snapshots := range [][]orientation.Snapshot{acceptance.Requirements, acceptance.Evidence} {
				for _, snapshot := range snapshots {
					if snapshot.Reasons == nil {
						ts.Fatalf("snapshot reasons must be an array: %+v", snapshot)
					}
				}
			}
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }
func TestExecutionFailures(t *testing.T) {
	for _, tc := range []struct {
		name      string
		writer    io.Writer
		operation cli.AssembleFunc
	}{
		{"operation", new(bytes.Buffer), func(context.Context, taskcontext.Request) (taskcontext.Result, error) {
			return taskcontext.Result{}, errors.New("operation failed")
		}},
		{"writer", failingWriter{}, func(context.Context, taskcontext.Request) (taskcontext.Result, error) {
			return taskcontext.Result{Complete: true}, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			status := cli.Run(context.Background(), []string{"ctx", "context", "--bundle", ".", "--ticket", "x"}, tc.writer, &stderr, cli.Operations{Assemble: tc.operation})
			if status != 2 || stderr.Len() == 0 {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
		})
	}
}

func TestDefaultAndExplicitLimitFlags(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		files int
		bytes int64
	}{
		{nil, 100, 1_048_576},
		{[]string{"--max-files", "7", "--max-bytes", "1234"}, 7, 1234},
	} {
		var stdout, stderr bytes.Buffer
		called := false
		args := append([]string{"ctx", "context", "--bundle", ".", "--ticket", "root.md"}, tc.args...)
		status := cli.Run(context.Background(), args, &stdout, &stderr, cli.Operations{Assemble: func(_ context.Context, request taskcontext.Request) (taskcontext.Result, error) {
			called = true
			if request.MaxFiles != tc.files || request.MaxBytes != tc.bytes {
				t.Fatalf("unexpected limits: %+v", request)
			}
			return taskcontext.Result{Complete: true}, nil
		}})
		if status != 0 || !called || stderr.Len() != 0 {
			t.Fatalf("status=%d called=%v stderr=%q", status, called, stderr.String())
		}
	}
}
