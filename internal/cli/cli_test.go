package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"ctx": func() { os.Exit(cli.Run(context.Background(), os.Args, os.Stdout, os.Stderr, taskcontext.Assemble)) }})
}

func TestCLI(t *testing.T) {
	testscript.Run(t, testscript.Params{Dir: "testdata", Cmds: map[string]func(*testscript.TestScript, bool, []string){"exit": func(ts *testscript.TestScript, neg bool, args []string) {
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
	}}})
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
			status := cli.Run(context.Background(), []string{"ctx", "context", "--project", ".", "--ticket", "x"}, tc.writer, &stderr, tc.operation)
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
		args := append([]string{"ctx", "context", "--project", ".", "--ticket", "root.md"}, tc.args...)
		status := cli.Run(context.Background(), args, &stdout, &stderr, func(_ context.Context, request taskcontext.Request) (taskcontext.Result, error) {
			called = true
			if request.MaxFiles != tc.files || request.MaxBytes != tc.bytes {
				t.Fatalf("unexpected limits: %+v", request)
			}
			return taskcontext.Result{Complete: true}, nil
		})
		if status != 0 || !called || stderr.Len() != 0 {
			t.Fatalf("status=%d called=%v stderr=%q", status, called, stderr.String())
		}
	}
}
