package factoryworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestWorkerHelper(t *testing.T) {
	if os.Getenv("CTX_FACTORY_ATTEMPT") == "" {
		return
	}
	var a Assignment
	var input json.RawMessage
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	_ = json.Unmarshal(input, &a)
	if a.Workspace == "" {
		var canonical factory.Assignment
		_ = json.Unmarshal(input, &canonical)
		a.Workspace = canonical.WorkspacePath
	}
	i := 0
	for i < len(os.Args) && os.Args[i] != "--" {
		i++
	}
	args := os.Args[i+1:]
	switch args[0] {
	case "standard-fixture":
		if e := Fixture(context.Background(), bytes.NewReader(input), os.Stdout, 0, false); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(12)
		}
	case "standard-check":
		if e := FixtureCheck(bytes.NewReader(input), os.Stdout); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(13)
		}
	case "commit":
		if err := os.WriteFile("fixture.txt", []byte(args[1]), 0600); err != nil {
			os.Exit(3)
		}
		if _, err := git(a.Workspace, "add", "fixture.txt"); err != nil {
			os.Exit(4)
		}
		if _, err := git(a.Workspace, "commit", "-m", "offline fixture candidate"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(5)
		}
		fmt.Println("fixture command complete")
	case "fail":
		fmt.Println("observed failure")
		os.Exit(7)
	case "pass":
		fmt.Println("observed passing check")
	case "result":
		if err := os.WriteFile(os.Getenv("CTX_FACTORY_RESULT"), []byte(`{"costUsd":1}`), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(14)
		}
	case "dispute", "dispute-wait", "dispute-descendant":
		if err := os.WriteFile(os.Getenv("CTX_FACTORY_DISPUTE"), []byte(disputeFixture), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(15)
		}
		if args[0] == "dispute-wait" {
			time.Sleep(30 * time.Second)
		}
		if args[0] == "dispute-descendant" {
			raw, _ := json.Marshal(a)
			child := exec.Command(os.Args[0], "-test.run=^TestWorkerHelper$", "--", "short-wait")
			child.Env, child.Stdin = os.Environ(), bytes.NewReader(raw)
			if err := child.Start(); err != nil {
				os.Exit(16)
			}
			_ = child.Process.Release()
		}
	case "dispute-prose":
		fmt.Println(disputeFixture)
	case "descendant":
		raw, _ := json.Marshal(a)
		child := exec.Command(os.Args[0], "-test.run=^TestWorkerHelper$", "--", "short-wait")
		child.Env = os.Environ()
		child.Stdin = bytes.NewReader(raw)
		if e := child.Start(); e != nil {
			os.Exit(11)
		}
		_ = child.Process.Release()
	case "short-wait":
		time.Sleep(750 * time.Millisecond)
	case "wait":
		if err := os.WriteFile(filepath.Join(a.Workspace, "started"), []byte("started"), 0600); err != nil {
			os.Exit(6)
		}
		time.Sleep(30 * time.Second)
	default:
		os.Exit(9)
	}
	os.Exit(0)
}
func repo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.name", "Fixture"}, {"config", "user.email", "fixture@example.invalid"}} {
		if _, e := git(dir, args...); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0600); e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]string{{"add", "base.txt"}, {"commit", "-m", "base"}} {
		if _, e := git(dir, args...); e != nil {
			t.Fatal(e)
		}
	}
	base, e := git(dir, "rev-parse", "HEAD")
	if e != nil {
		t.Fatal(e)
	}
	return dir, base
}
func helper(name string, args ...string) Command {
	argv := []string{os.Args[0], "-test.run=^TestWorkerHelper$", "--", name}
	return Command{Name: name, Argv: append(argv, args...)}
}
func assignment(dir, base string) Assignment {
	return Assignment{AttemptID: "attempt-1", JobID: "job-1", WorkerID: "fixture-worker", Role: "implementer", Workspace: dir, Base: base, Requirements: "sha256:fixture", Context: json.RawMessage(`{"sources":[{"text":"offline fixture"}]}`), Mode: "fixture", FixtureProject: true}
}
func TestLiteralArgvRealCommitAndRepeatedReceipt(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	literal := "literal ; $(touch INJECTED) `touch INJECTED` && &"
	a.Command = helper("commit", literal)
	p := Process{ReceiptDir: t.TempDir()}
	r, e := p.Execute(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	if r.State != "succeeded" || r.Candidate == base || r.Candidate == "" {
		t.Fatalf("receipt %+v", r)
	}
	b, e := os.ReadFile(filepath.Join(dir, "fixture.txt"))
	if e != nil || string(b) != literal {
		t.Fatalf("literal argv: %q, %v", b, e)
	}
	if _, e = os.Stat(filepath.Join(dir, "INJECTED")); !os.IsNotExist(e) {
		t.Fatal("metacharacters executed")
	}
	again, e := (&Process{ReceiptDir: p.ReceiptDir}).Execute(context.Background(), a)
	if e != nil || again.Candidate != r.Candidate || again.StartedAt != r.StartedAt {
		t.Fatalf("repeated delivery: %+v, %v", again, e)
	}
	a.Command = helper("fail")
	if _, e = p.Execute(context.Background(), a); e == nil {
		t.Fatal("changed delivery reused receipt")
	}
}
func TestCommandsInTheirWorkspaceReachARelativeReceiptDirectory(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Command = helper("result")
	t.Chdir(t.TempDir())
	p := Process{ReceiptDir: "receipts"}
	if _, e := p.Execute(context.Background(), a); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(p.resultPath(a.AttemptID))
	if e != nil || string(b) != `{"costUsd":1}` {
		t.Fatalf("result written from the workspace: %q, %v", b, e)
	}
}
func TestVerifierRunsEveryRequiredCommandAndRecordsFailure(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Role = "verifier"
	a.Candidate = base
	a.Checks = []Command{helper("fail"), helper("pass")}
	p := Process{ReceiptDir: t.TempDir()}
	r, e := p.Execute(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	if r.State != "failed" || len(r.Observations) != 2 || r.Observations[0].ExitCode != 7 || r.Observations[1].Status != "passed" {
		t.Fatalf("receipt %+v", r)
	}
	if !strings.Contains(r.Observations[0].Log, "observed failure") {
		t.Fatal("missing actual log")
	}
}
func TestUnfinishedReceiptAfterRestartNeverStartsProcess(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Command = helper("commit", "must not run")
	p := Process{ReceiptDir: t.TempDir()}
	// An interruption during context retention leaves the prepared durable intent.
	path := p.receiptPath(a.AttemptID)
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(filepath.Join(filepath.Dir(path), "assignment.json"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e := p.Execute(context.Background(), a); e == nil {
		t.Fatal("expected retention failure")
	}
	r, e := (&Process{ReceiptDir: p.ReceiptDir}).Execute(context.Background(), a)
	if e != nil || r.State != "unknown" {
		t.Fatalf("restart %+v %v", r, e)
	}
	if _, e = os.Stat(filepath.Join(dir, "fixture.txt")); !os.IsNotExist(e) {
		t.Fatal("unknown attempt ran")
	}
}
func TestCancellationWaitsForObservedProcessExit(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Command = helper("wait")
	p := Process{ReceiptDir: t.TempDir(), StopGrace: 50 * time.Millisecond}
	type result struct {
		r Receipt
		e error
	}
	done := make(chan result, 1)
	go func() { r, e := p.Execute(context.Background(), a); done <- result{r, e} }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, e := os.Stat(filepath.Join(dir, "started")); e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !p.Cancel(a.AttemptID) {
		t.Fatal("owned invocation missing")
	}
	got := <-done
	if got.e != nil || got.r.State != "cancelled" {
		t.Fatalf("cancellation %+v %v", got.r, got.e)
	}
	if got.r.PID <= 0 || got.r.FinishedAt.IsZero() {
		t.Fatal("no observed terminal execution")
	}
	if p.Cancel(a.AttemptID) {
		t.Fatal("terminal process still owned")
	}
}
func TestFixtureDeniedForNormalProject(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Command = helper("pass")
	a.FixtureProject = false
	if _, e := (&Process{ReceiptDir: t.TempDir()}).Execute(context.Background(), a); e == nil {
		t.Fatal("fixture allowed")
	}
}
func TestVerifierRejectsDirtyOrChangedCandidate(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Role = "verifier"
	a.Candidate = base
	a.Checks = []Command{helper("pass")}
	if e := os.WriteFile(filepath.Join(dir, "untracked"), []byte("x"), 0600); e != nil {
		t.Fatal(e)
	}
	r, e := (&Process{ReceiptDir: t.TempDir()}).Execute(context.Background(), a)
	if e != nil || r.State != "failed" || len(r.Observations) != 0 {
		t.Fatalf("dirty verifier %+v %v", r, e)
	}
}

func TestLeaderExitWithLiveDescendantIsUnknown(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Command = helper("descendant")
	r, e := (&Process{ReceiptDir: t.TempDir()}).Execute(context.Background(), a)
	if e != nil || r.State != "unknown" {
		t.Fatalf("live group %+v %v", r, e)
	}
	time.Sleep(time.Second)
}
func TestGenericCancellationCannotProveExternalExecutionStopped(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Mode = "command"
	a.FixtureProject = false
	a.Command = helper("wait")
	ctx, cancel := context.WithCancel(context.Background())
	p := Process{ReceiptDir: t.TempDir()}
	done := make(chan Receipt, 1)
	go func() { r, _ := p.Execute(ctx, a); done <- r }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, e := os.Stat(filepath.Join(dir, "started")); e == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no process start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	r := <-done
	if r.State != "unknown" {
		t.Fatalf("generic cancellation %+v", r)
	}
}

func TestUnreadableReceiptIsUnknownAndNeverLaunches(t *testing.T) {
	dir, base := repo(t)
	a := assignment(dir, base)
	a.Command = helper("commit", "must not run")
	p := Process{ReceiptDir: t.TempDir()}
	path := p.receiptPath(a.AttemptID)
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, []byte("interrupted JSON"), 0600); e != nil {
		t.Fatal(e)
	}
	r, e := p.Execute(context.Background(), a)
	if e != nil || r.State != "unknown" {
		t.Fatalf("unreadable receipt %+v %v", r, e)
	}
	if _, e = os.Stat(filepath.Join(dir, "fixture.txt")); !os.IsNotExist(e) {
		t.Fatal("unreadable receipt launched command")
	}
}
