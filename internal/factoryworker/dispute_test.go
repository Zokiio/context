package factoryworker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

const disputeFixture = `{"summary":"The locked assertion contradicts the empty-input criterion.","criterionRefs":[3],"testPaths":["strutil/wordfreq_test.go"],"observedFailure":"TestEmpty expects nil; criterion 3 requires a non-nil empty map.","evidenceRefs":["criterion 3","strutil/wordfreq_test.go:TestEmpty"],"proposedChange":"Require a non-nil empty map in TestEmpty."}`

func TestDisputeFileIsBoundedConfinedAndStructured(t *testing.T) {
	for _, tc := range []struct{ name, data string }{
		{"unknown field", strings.TrimSuffix(disputeFixture, "}") + `,"extra":true}`},
		{"trailing object", disputeFixture + `{}`},
		{"missing evidence", `{"summary":"conflict","observedFailure":"wrong","criterionRefs":[1],"evidenceRefs":[]}`},
		{"malformed", `{"summary":`},
		{"oversized", strings.Repeat(" ", maxDisputeBytes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "dispute.json")
			if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := readDispute(path); err == nil {
				t.Fatal("invalid dispute file was accepted")
			}
		})
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(disputeFixture), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dispute.json")
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := readDispute(path); err == nil {
		t.Fatal("dispute symlink escaped the output directory")
	}
}

func TestAgentDisputeStopsBeforeCandidateCommit(t *testing.T) {
	output := filepath.Join(t.TempDir(), "dispute.json")
	t.Setenv("CTX_FACTORY_DISPUTE", output)
	workspace, assignment := agentWorkspace(t)
	dir := t.TempDir()
	binary := filepath.Join(dir, "claude")
	promptPath := filepath.Join(dir, "prompt.txt")
	script := "#!/bin/sh\ncat > " + promptPath + "\nprintf unfinished > implementation.txt\ncat > \"$CTX_FACTORY_DISPUTE\" <<'JSON'\n" + disputeFixture + "\nJSON\necho '{\"type\":\"result\",\"is_error\":false,\"result\":\"reported conflict\"}'\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if out, err := runAgent(t, binary, assignment); err != nil || !strings.Contains(out, "without committing a candidate") {
		t.Fatalf("agent dispute = %s, %v", out, err)
	}
	prompt, err := os.ReadFile(promptPath)
	if err != nil || !strings.Contains(string(prompt), "CTX_FACTORY_DISPUTE for this attempt is "+filepath.Join(workspace, ".waymark-dispute-")) || strings.Contains(string(prompt), output) {
		t.Fatalf("native agent lacked a confined writable report path: %s, %v", prompt, err)
	}
	if head, err := git(workspace, "rev-parse", "HEAD"); err != nil || head != assignment.BaseRevision {
		t.Fatalf("dispute committed a candidate: %s, %v", head, err)
	}
	dispute, err := readDispute(output)
	if err != nil || dispute == nil || dispute.CriterionRefs[0] != 3 {
		t.Fatalf("durable report = %+v, %v", dispute, err)
	}
	if matches, _ := filepath.Glob(filepath.Join(workspace, ".waymark-dispute-*")); len(matches) != 0 {
		t.Fatalf("native report staging leaked into workspace: %v", matches)
	}
	if _, err := os.Stat(filepath.Join(workspace, "implementation.txt")); err != nil {
		t.Fatal("unfinished implementation evidence was removed")
	}
}

func TestWorkerDisputeReceiptReplaysWithoutRerunning(t *testing.T) {
	dir, base := repo(t)
	client := &fakeClient{a: canonical(dir, base), failTerminal: 1}
	client.a.Command = helper("dispute").Argv
	process := &Process{ReceiptDir: t.TempDir()}
	worker := Worker{Client: client, Process: process, Stage: factory.StageImplementation}
	receipt, err := worker.Once(context.Background())
	if err != nil || receipt.State != "disputed" || receipt.Dispute == nil || receipt.Candidate != "" {
		t.Fatalf("disputed outcome = %+v, %v", receipt, err)
	}
	first := append([]factory.WorkerReportRequest{}, client.reports...)
	if len(first) != 4 || first[2].Outcome != factory.OutcomeDisputed || first[2].Dispute == nil || first[2].CandidateRevision != "" || !reflect.DeepEqual(first[2], first[3]) {
		t.Fatalf("dispute report/retry = %+v", first)
	}
	var saved Receipt
	if err := json.Unmarshal([]byte(first[2].TerminalReceipt), &saved); err != nil || !reflect.DeepEqual(saved.Dispute, receipt.Dispute) {
		t.Fatalf("terminal receipt lost dispute: %+v, %v", saved, err)
	}
	// Replay uses only the durable receipt, even if the transient output is gone.
	if err := os.Remove(process.disputePath(string(client.a.AttemptID))); err != nil {
		t.Fatal(err)
	}
	worker.Process = &Process{ReceiptDir: process.ReceiptDir}
	again, err := worker.Once(context.Background())
	if err != nil || !reflect.DeepEqual(receipt, again) || len(client.reports) != 5 || !reflect.DeepEqual(first[2], client.reports[4]) {
		t.Fatalf("dispute replay = %+v, %v; reports=%+v", again, err, client.reports)
	}
}

func TestDisputeCannotOverrideUncertainOrCancelledExecution(t *testing.T) {
	for _, tc := range []struct{ command, mode, want string }{
		{"dispute-wait", "fixture", "cancelled"},
		{"dispute-wait", "command", "unknown"},
		{"dispute-descendant", "fixture", "unknown"},
		{"dispute-prose", "fixture", "failed"},
	} {
		t.Run(tc.command+"/"+tc.mode, func(t *testing.T) {
			dir, base := repo(t)
			a := assignment(dir, base)
			a.Command, a.Mode = helper(tc.command), tc.mode
			process := &Process{ReceiptDir: t.TempDir(), StopGrace: 50 * time.Millisecond}
			ctx := context.Background()
			var cancel context.CancelFunc
			if tc.command == "dispute-wait" {
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				go func() {
					deadline := time.Now().Add(5 * time.Second)
					for time.Now().Before(deadline) {
						if _, err := os.Stat(process.disputePath(a.AttemptID)); err == nil {
							cancel()
							return
						}
						time.Sleep(10 * time.Millisecond)
					}
					cancel()
				}()
			}
			receipt, err := process.Execute(ctx, a)
			if err != nil || receipt.State != tc.want || receipt.Dispute != nil || (tc.want != "failed" && receipt.Candidate != "") {
				t.Fatalf("dispute overrode execution: %+v, %v", receipt, err)
			}
		})
	}
}
