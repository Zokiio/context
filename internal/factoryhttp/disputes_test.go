package factoryhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryclient"
)

func TestDisputeHTTPAuthorizationVersionsAndRetainedDecision(t *testing.T) {
	h := start(t)
	ctx := context.Background()
	if _, err := h.service.RegisterExecutor(ctx, h.admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-executor", DisplayName: "Fixture", Command: []string{"fixture"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.RegisterCheck(ctx, h.admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Fixture check", Command: []string{"fixture"}, Environment: "offline", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	p := project(t, h, "dispute-http")
	job, err := h.service.Submit(ctx, h.admin, factory.SubmitRequest{RequestID: "job", ProjectID: p.ID, WorkItemID: "work-1", BaseRevision: "HEAD", ExecutorKey: "fixture-executor", RepairLimit: 1, Deadline: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := h.service.CreateUser(ctx, h.admin, factory.CreateUserRequest{RequestID: "builder", Username: "builder", Kind: factory.UserKindWorker})
	if err != nil {
		t.Fatal(err)
	}
	token, err := h.service.IssueToken(ctx, h.admin, factory.IssueTokenRequest{RequestID: "builder-token", UserID: worker.ID, Label: "test"})
	if err != nil {
		t.Fatal(err)
	}
	w, err := h.service.Authenticate(ctx, token.Token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.GrantProject(ctx, h.admin, factory.GrantProjectRequest{RequestID: "worker-grant", ProjectID: p.ID, UserID: worker.ID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityImplementWorker}}); err != nil {
		t.Fatal(err)
	}
	a, err := h.service.Claim(ctx, w, factory.WorkerClaimRequest{RequestID: "claim", Stages: []factory.Stage{factory.StageImplementation}, ProjectID: &p.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []factory.WorkerReportKind{factory.ReportStartRegistered, factory.ReportStarted} {
		if _, err := h.service.Report(ctx, w, factory.WorkerReportRequest{RequestID: string(kind), AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: kind, ObservedAt: time.Now(), NativeRunID: "failed-builder"}); err != nil {
			t.Fatal(err)
		}
	}
	job, err = h.service.Report(ctx, w, factory.WorkerReportRequest{RequestID: "failed", AttemptID: a.AttemptID, DispatchID: a.DispatchID, Kind: factory.ReportTerminal, ObservedAt: time.Now(), NativeRunID: "failed-builder", Outcome: factory.OutcomeFailed, TerminalReceipt: "The builder exited and retained its error log"})
	if err != nil {
		t.Fatal(err)
	}
	reader, _, readerToken := user(t, h, "reader")
	if _, err := h.service.GrantProject(ctx, h.admin, factory.GrantProjectRequest{RequestID: "reader-grant", ProjectID: p.ID, UserID: reader.ID, Capabilities: []factory.Capability{factory.CapabilityRead}}); err != nil {
		t.Fatal(err)
	}
	_, _, outsideToken := user(t, h, "outside")
	cl, err := factoryclient.New(h.server.URL, h.token)
	if err != nil {
		t.Fatal(err)
	}
	req := factory.ReportDisputeRequest{RequestID: "dispute", JobID: job.ID, IfVersion: job.Version, Dispute: factory.DisputeInput{Summary: "Expected behavior needs review", CriterionRefs: []int{1}, ObservedFailure: "A conflicting assertion prevented implementation", EvidenceRefs: []string{"attempt error log"}}}
	raw, _ := json.Marshal(req)
	path := "/v1/jobs/" + string(job.ID) + "/disputes"
	for _, tc := range []struct {
		token  string
		status int
	}{{"", 401}, {outsideToken, 404}, {readerToken, 403}} {
		status, _, body := call(t, h.server.URL, tc.token, "POST", path, string(raw), nil)
		if status != tc.status {
			t.Fatalf("authorization: %d %s; want %d", status, body, tc.status)
		}
	}
	status, _, body := call(t, h.server.URL, h.token, "POST", path, `{"requestId":"mismatch","jobId":"other","ifVersion":1}`, nil)
	if status != 400 {
		t.Fatalf("job path binding: %d %s", status, body)
	}
	stale := req
	stale.RequestID = "stale"
	stale.IfVersion--
	if _, err := cl.ReportDispute(ctx, stale); disputeErrorCode(err) != factory.CodeConflict {
		t.Fatalf("stale report: %v", err)
	}
	d, err := cl.ReportDispute(ctx, req)
	if err != nil || d.Status != "needs_human" || d.NextOwner != "you" || len(d.AvailableActions) == 0 {
		t.Fatalf("report: %+v %v", d, err)
	}
	replay, err := cl.ReportDispute(ctx, req)
	if err != nil || !reflect.DeepEqual(replay, d) {
		t.Fatalf("report replay changed: %+v %+v %v", d, replay, err)
	}
	for _, path := range []string{"/v1/disputes/" + string(d.ID), fmt.Sprintf("/v1/projects/%s/work-items/work-1/disputes", p.ID)} {
		status, _, body = call(t, h.server.URL, outsideToken, "GET", path, "", nil)
		if status != 404 {
			t.Fatalf("read isolation: %d %s", status, body)
		}
	}
	readClient, _ := factoryclient.New(h.server.URL, readerToken)
	read, err := readClient.GetDispute(ctx, d.ID)
	if err != nil || len(read.AvailableActions) != 0 {
		t.Fatalf("reader actions: %+v %v", read, err)
	}
	list, err := cl.ListDisputes(ctx, p.ID, "work-1")
	if err != nil || len(list) != 1 || list[0].ID != d.ID {
		t.Fatalf("dispute list: %+v %v", list, err)
	}
	resolution := factory.ResolveDisputeRequest{RequestID: "return", DisputeID: d.ID, IfVersion: d.Version, Action: "return_to_planning", Rationale: "Clarify the requirement before continuing"}
	if _, err := readClient.ResolveDispute(ctx, resolution); disputeErrorCode(err) != factory.CodeForbidden {
		t.Fatalf("reader resolved: %v", err)
	}
	status, _, body = call(t, h.server.URL, h.token, "POST", "/v1/disputes/"+string(d.ID)+"/resolve", `{"requestId":"mismatch-resolution","disputeId":"other","ifVersion":1,"action":"dismiss","rationale":"mismatch"}`, nil)
	if status != 400 {
		t.Fatalf("dispute path binding: %d %s", status, body)
	}
	done, err := cl.ResolveDispute(ctx, resolution)
	if err != nil || done.Status != "resolved" || done.NextOwner != "you" {
		t.Fatalf("return to planning: %+v %v", done, err)
	}
	again, err := cl.ResolveDispute(ctx, resolution)
	if err != nil || !reflect.DeepEqual(again, done) {
		t.Fatalf("resolution replay changed: %+v %+v %v", done, again, err)
	}
	resolution.RequestID = "old-version"
	if _, err := cl.ResolveDispute(ctx, resolution); disputeErrorCode(err) != factory.CodeConflict {
		t.Fatalf("stale resolution: %v", err)
	}
}

func disputeErrorCode(err error) factory.ErrorCode {
	var rejection *factoryclient.APIError
	if errors.As(err, &rejection) {
		return factory.ErrorCode(rejection.Code)
	}
	return ""
}
