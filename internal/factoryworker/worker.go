package factoryworker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

// Client is implemented by factoryclient.Client. Every worker call is authorized
// and correlated by the service against the immutable attempt and dispatch.
type Client interface {
	Claim(context.Context, factory.WorkerClaimRequest) (factory.Assignment, error)
	Heartbeat(context.Context, factory.HeartbeatRequest) (factory.HeartbeatResult, error)
	Report(context.Context, factory.WorkerReportRequest) (factory.Job, error)
}

type Worker struct {
	Client            Client
	Process           *Process
	Stage             factory.Stage
	ProjectID         *factory.ProjectID
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	PermanentError    func(error) bool
}

// Once claims at most one assignment, runs it once, and retains the final receipt
// even if the service becomes unavailable before the report is acknowledged.
func (w *Worker) Once(ctx context.Context) (Receipt, error) {
	if err := platformSupport(); err != nil {
		return Receipt{}, err
	}
	if w.Client == nil || w.Process == nil {
		return Receipt{}, errors.New("worker requires client and receipt store")
	}
	if w.Stage != factory.StageImplementation && w.Stage != factory.StageVerification {
		return Receipt{}, errors.New("worker stage must be implementation or verification")
	}
	claimID := requestID("claim", fmt.Sprint(time.Now().UnixNano()))
	a, e := w.Client.Claim(ctx, factory.WorkerClaimRequest{RequestID: claimID, Stages: []factory.Stage{w.Stage}, ProjectID: w.ProjectID})
	if e != nil {
		return Receipt{}, e
	}
	raw, e := json.Marshal(a)
	if e != nil {
		return Receipt{}, e
	}
	sources, e := json.Marshal(a.ContextSources)
	if e != nil {
		return Receipt{}, e
	}
	role := "implementer"
	if a.Stage == factory.StageVerification {
		role = "verifier"
	}
	mode := "command"
	if a.Fixture {
		mode = "fixture"
	}
	local := Assignment{AttemptID: string(a.AttemptID), DispatchID: a.DispatchID, JobID: string(a.JobID), WorkerID: string(a.WorkerID), Role: role, Workspace: a.WorkspacePath, Base: a.BaseRevision, Candidate: a.CandidateRevision, Requirements: a.RequirementFingerprint, Context: sources, Command: Command{Name: a.ProfileKey, Argv: a.Command}, Mode: mode, FixtureProject: a.Fixture, Raw: raw}
	if role == "verifier" {
		local.Checks = []Command{local.Command}
	}
	native := requestID("process", a.DispatchID)
	report := func(kind factory.WorkerReportKind, extra func(*factory.WorkerReportRequest)) error {
		r := factory.WorkerReportRequest{RequestID: requestID(string(kind), a.DispatchID), AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: kind, ObservedAt: time.Now().UTC(), NativeRunID: native}
		if extra != nil {
			extra(&r)
		}
		return w.report(ctx, r)
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	doneHeartbeat := make(chan struct{})
	go func() {
		defer close(doneHeartbeat)
		interval := w.HeartbeatInterval
		if interval <= 0 {
			interval = 2 * time.Second
		}
		timer := time.NewTicker(interval)
		defer timer.Stop()
		cancelAck := false
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-timer.C:
				callCtx, cancel := context.WithTimeout(heartbeatCtx, 5*time.Second)
				h, err := w.Client.Heartbeat(callCtx, factory.HeartbeatRequest{RequestID: requestID("heartbeat", a.DispatchID+fmt.Sprint(time.Now().UnixNano())), AttemptID: a.AttemptID, DispatchID: a.DispatchID, Progress: latestProgress(w.Process.progressPath(string(a.AttemptID)))})
				cancel()
				if err == nil && h.Directive == factory.DirectiveCancel {
					// Acknowledgement says only that the wrapper forwarded the request.
					if w.Process.Cancel(string(a.AttemptID)) && !cancelAck {
						_ = report(factory.ReportCancelAcknowledged, nil)
						cancelAck = true
					}
				}
			}
		}
	}()
	hooks := Hooks{Register: func(context.Context, Receipt) error { return report(factory.ReportStartRegistered, nil) }, Started: func(_ context.Context, pid int) error {
		return report(factory.ReportStarted, func(r *factory.WorkerReportRequest) { r.Summary = fmt.Sprintf("spawned process pid %d", pid) })
	}}
	runCtx := ctx
	var deadlineCancel context.CancelFunc
	if !a.Deadline.IsZero() {
		runCtx, deadlineCancel = context.WithDeadline(ctx, a.Deadline)
		defer deadlineCancel()
	}
	receipt, runerr := w.Process.execute(runCtx, local, hooks)
	stopHeartbeat()
	<-doneHeartbeat
	if runerr != nil {
		return receipt, runerr
	}
	kind := factory.ReportTerminal
	if receipt.State == "unknown" {
		kind = factory.ReportUnknown
	}
	terminalBytes, e := json.Marshal(receipt)
	if e != nil {
		return receipt, e
	}
	final := factory.WorkerReportRequest{RequestID: requestID("outcome", a.DispatchID), AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: kind, ObservedAt: receipt.FinishedAt, NativeRunID: native, Outcome: factory.WorkerOutcome(receipt.State), CandidateRevision: receipt.Candidate, Summary: receipt.Error, TerminalReceipt: string(terminalBytes), Artifacts: []factory.ArtifactInput{{Name: "process-receipt.json", MediaType: "application/json", Content: string(terminalBytes)}}}
	if receipt.State == "disputed" {
		final.Dispute = receipt.Dispute
		data, err := json.Marshal(receipt.Dispute)
		if err != nil {
			return receipt, err
		}
		final.Artifacts = append(final.Artifacts, factory.ArtifactInput{Name: "dispute.json", MediaType: "application/json", Content: string(data)})
	}
	if final.ObservedAt.IsZero() {
		final.ObservedAt = time.Now().UTC()
	}
	for i, o := range receipt.Observations {
		final.Artifacts = append(final.Artifacts, factory.ArtifactInput{Name: fmt.Sprintf("command-%d.log", i+1), MediaType: "text/plain", Content: o.Log})
	}
	if result, err := os.ReadFile(w.Process.resultPath(string(a.AttemptID))); err == nil && len(result) <= 64<<10 && json.Valid(result) {
		final.Artifacts = append(final.Artifacts, factory.ArtifactInput{Name: "agent-result.json", MediaType: "application/json", Content: string(result)})
	}
	if a.Stage == factory.StageVerification {
		final.CandidateRevision = a.CandidateRevision
		final.CheckStatus = factory.CheckUnknown
		if len(receipt.Observations) == 1 {
			switch status := factory.CheckStatus(receipt.Observations[0].Status); status {
			case factory.CheckPassed, factory.CheckFailed, factory.CheckSkipped, factory.CheckUnavailable, factory.CheckUnknown:
				final.CheckStatus = status
			}
		}
		if receipt.State == "failed" && final.CheckStatus == factory.CheckPassed {
			final.CheckStatus = factory.CheckFailed
		}
	}
	e = w.report(ctx, final)
	return receipt, e
}

func (w *Worker) Loop(ctx context.Context, observe func(Receipt, error)) error {
	interval := w.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	for {
		r, e := w.Once(ctx)
		if observe != nil {
			observe(r, e)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e != nil && w.PermanentError != nil && w.PermanentError(e) {
			return e
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (w *Worker) report(ctx context.Context, r factory.WorkerReportRequest) error {
	var err error
	for n := 0; n < 4; n++ {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = w.Client.Report(callCtx, r)
		cancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if w.PermanentError != nil && w.PermanentError(err) {
			return err
		}
		timer := time.NewTimer(time.Duration(n+1) * 200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
func requestID(prefix, input string) string {
	h := sha256.Sum256([]byte(input))
	return prefix + "-" + hex.EncodeToString(h[:16])
}

// latestProgress returns the last nonempty line of a command's progress file, reading
// at most its final 4 KiB.
func latestProgress(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	offset := info.Size() - 4096
	if offset < 0 {
		offset = 0
	}
	buf := make([]byte, info.Size()-offset)
	if _, err := f.ReadAt(buf, offset); err != nil && !errors.Is(err, io.EOF) {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(buf), "\r\n"), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
