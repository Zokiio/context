package factoryworker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

type fakeClient struct {
	mu                sync.Mutex
	a                 factory.Assignment
	reports           []factory.WorkerReportRequest
	heartbeatCount    int
	cancelAfter       int
	failTerminal      int
	failRegistration  bool
	registrationCheck func()
}

func (f *fakeClient) Claim(context.Context, factory.WorkerClaimRequest) (factory.Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.a, nil
}
func (f *fakeClient) Heartbeat(_ context.Context, r factory.HeartbeatRequest) (factory.HeartbeatResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.AttemptID != f.a.AttemptID || r.DispatchID != f.a.DispatchID {
		return factory.HeartbeatResult{}, errors.New("wrong heartbeat identity")
	}
	f.heartbeatCount++
	if f.cancelAfter > 0 && f.heartbeatCount >= f.cancelAfter {
		return factory.HeartbeatResult{AttemptID: r.AttemptID, Directive: factory.DirectiveCancel}, nil
	}
	return factory.HeartbeatResult{}, errors.New("brief service interruption")
}
func (f *fakeClient) Report(_ context.Context, r factory.WorkerReportRequest) (factory.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reports = append(f.reports, r)
	if r.AttemptID != f.a.AttemptID || r.DispatchID != f.a.DispatchID {
		return factory.Job{}, errors.New("wrong report identity")
	}
	if r.Kind == factory.ReportStartRegistered {
		if f.registrationCheck != nil {
			f.registrationCheck()
		}
		if f.failRegistration {
			return factory.Job{}, errors.New("registration rejected")
		}
	}
	if r.Kind == factory.ReportTerminal && f.failTerminal > 0 {
		f.failTerminal--
		return factory.Job{}, errors.New("terminal response lost")
	}
	return factory.Job{}, nil
}
func canonical(dir, base string) factory.Assignment {
	return factory.Assignment{AttemptID: "attempt-1", DispatchID: "dispatch-1", JobID: "job-1", ProjectID: "fixture-project", WorkItemID: "fixture-item", Stage: factory.StageImplementation, WorkerID: "fixture-worker", WorkspacePath: dir, BaseRevision: base, RequirementFingerprint: "requirements-1", ContextSHA256: "context-1", ContextSources: []factory.ContextSource{{Path: "ticket.md", Text: "immutable fixture requirements"}}, ProfileKey: "offline-fixture", Command: helper("commit", "real offline candidate").Argv, Fixture: true, Deadline: time.Now().UTC().Add(time.Minute), LeaseExpiresAt: time.Now().UTC().Add(10 * time.Second)}
}
func TestWorkerReplayIgnoresOnlyRenewedLeaseAndRetriesReport(t *testing.T) {
	dir, base := repo(t)
	f := &fakeClient{a: canonical(dir, base), failTerminal: 1}
	f.registrationCheck = func() {
		if _, e := os.Stat(filepath.Join(dir, "fixture.txt")); !os.IsNotExist(e) {
			t.Error("execution preceded start registration")
		}
	}
	p := &Process{ReceiptDir: t.TempDir()}
	w := Worker{Client: f, Process: p, Stage: factory.StageImplementation}
	r, e := w.Once(context.Background())
	if e != nil || r.State != "succeeded" {
		t.Fatalf("once %+v %v", r, e)
	}
	f.mu.Lock()
	firstReports := append([]factory.WorkerReportRequest(nil), f.reports...)
	f.a.LeaseExpiresAt = f.a.LeaseExpiresAt.Add(time.Hour)
	f.mu.Unlock()
	if len(firstReports) != 4 || firstReports[0].Kind != factory.ReportStartRegistered || firstReports[1].Kind != factory.ReportStarted || firstReports[2].Kind != factory.ReportTerminal {
		t.Fatalf("protocol %+v", firstReports)
	}
	if !reflect.DeepEqual(firstReports[2], firstReports[3]) {
		t.Fatal("report retry changed payload")
	}
	// A new Process object models restarting the worker with its receipt store.
	w.Process = &Process{ReceiptDir: p.ReceiptDir}
	again, e := w.Once(context.Background())
	if e != nil || again.Candidate != r.Candidate || !again.StartedAt.Equal(r.StartedAt) {
		t.Fatalf("replay %+v %v", again, e)
	}
	f.mu.Lock()
	for _, report := range f.reports[4:] {
		if report.Kind != factory.ReportTerminal {
			t.Fatalf("terminal receipt replay started another process: %+v", report)
		}
	}
	raw, e := os.ReadFile(filepath.Join(filepath.Dir(p.receiptPath(string(f.a.AttemptID))), "assignment.json"))
	if e != nil {
		t.Fatal(e)
	}
	var retained Assignment
	if e = json.Unmarshal(raw, &retained); e != nil {
		t.Fatal(e)
	}
	var original factory.Assignment
	if e = json.Unmarshal(retained.Raw, &original); e != nil {
		t.Fatal(e)
	}
	if original.LeaseExpiresAt.Equal(f.a.LeaseExpiresAt) {
		t.Fatal("retained original assignment changed on replay")
	}
	f.a.Command = helper("fail").Argv
	f.mu.Unlock()
	if _, e = w.Once(context.Background()); e == nil {
		t.Fatal("changed command reused same attempt")
	}
}
func TestWorkerCancellationAndHeartbeatInterruptionDoNotDuplicateExecution(t *testing.T) {
	dir, base := repo(t)
	f := &fakeClient{a: canonical(dir, base), cancelAfter: 3}
	f.a.Command = helper("wait").Argv
	w := Worker{Client: f, Process: &Process{ReceiptDir: t.TempDir(), StopGrace: 50 * time.Millisecond}, Stage: factory.StageImplementation, HeartbeatInterval: 20 * time.Millisecond}
	r, e := w.Once(context.Background())
	if e != nil || r.State != "cancelled" {
		t.Fatalf("cancel %+v %v", r, e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	counts := map[factory.WorkerReportKind]int{}
	for _, report := range f.reports {
		counts[report.Kind]++
	}
	if counts[factory.ReportStarted] != 1 || counts[factory.ReportCancelAcknowledged] != 1 || counts[factory.ReportTerminal] != 1 {
		t.Fatalf("reports %+v", counts)
	}
	if f.heartbeatCount < 3 {
		t.Fatal("no interrupted heartbeats")
	}
}
func TestUnacknowledgedRegistrationLeavesUnknownReceiptAndDoesNotLaunch(t *testing.T) {
	dir, base := repo(t)
	f := &fakeClient{a: canonical(dir, base), failRegistration: true}
	p := &Process{ReceiptDir: t.TempDir()}
	w := Worker{Client: f, Process: p, Stage: factory.StageImplementation, PermanentError: func(error) bool { return true }}
	if _, e := w.Once(context.Background()); e == nil {
		t.Fatal("unacknowledged registration ran")
	}
	if _, e := os.Stat(filepath.Join(dir, "fixture.txt")); !os.IsNotExist(e) {
		t.Fatal("process launched")
	}
	f.mu.Lock()
	f.failRegistration = false
	f.mu.Unlock()
	r, e := w.Once(context.Background())
	if e != nil || r.State != "unknown" {
		t.Fatalf("uncertain intent %+v %v", r, e)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reports[len(f.reports)-1].Kind != factory.ReportUnknown {
		t.Fatal("unfinished receipt not reported unknown")
	}
}
