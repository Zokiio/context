package factorycli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func TestDisputeCommandsPreserveEvidenceAndVersion(t *testing.T) {
	var report factory.ReportDisputeRequest
	var resolve factory.ResolveDisputeRequest
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/jobs/job-1/disputes":
			if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusAccepted)
			io.WriteString(w, `{"id":"dispute-1","status":"judging","nextOwner":"judge"}`)
		case "/v1/disputes/dispute-1/resolve":
			if err := json.NewDecoder(r.Body).Decode(&resolve); err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusConflict)
			io.WriteString(w, `{"code":"conflict","message":"dispute version changed"}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer s.Close()
	token := tokenFile(t)
	out, err := run(t, "--url", s.URL, "--token-file", token, "disputes", "report", "--job", "job-1", "--if-version", "7", "--request-id", "report-1", "--summary", "Test contradicts criterion", "--observed-failure", "wanted 2, got 1", "--criterion", "1", "--criterion", "3", "--test-path", "test,one_test.go", "--evidence", "command-1.log:22", "--evidence", "test,one_test.go:40", "--proposed-change", "Correct the expected count")
	if err != nil || !strings.Contains(out, "judge") {
		t.Fatalf("report: %s %v", out, err)
	}
	if report.RequestID != "report-1" || report.JobID != "job-1" || report.IfVersion != 7 || !reflect.DeepEqual(report.Dispute.CriterionRefs, []int{1, 3}) || !reflect.DeepEqual(report.Dispute.TestPaths, []string{"test,one_test.go"}) || !reflect.DeepEqual(report.Dispute.EvidenceRefs, []string{"command-1.log:22", "test,one_test.go:40"}) {
		t.Fatalf("report changed: %+v", report)
	}
	out, err = run(t, "--url", s.URL, "--token-file", token, "disputes", "resolve", "--id", "dispute-1", "--if-version", "9", "--request-id", "resolve-1", "--action", "accept_tests", "--rationale", "The corrected test preserves the criterion")
	if err == nil || out != "" || requests != 2 {
		t.Fatalf("stale resolution retried or succeeded: %q %v, requests=%d", out, err, requests)
	}
	if resolve.RequestID != "resolve-1" || resolve.DisputeID != "dispute-1" || resolve.IfVersion != 9 || resolve.Action != "accept_tests" {
		t.Fatalf("resolution changed: %+v", resolve)
	}
}

func TestDisputeReportRejectsMissingOrInvalidEvidenceBeforeHTTP(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid report reached HTTP") }))
	defer s.Close()
	for _, extra := range [][]string{{"--criterion", "0", "--evidence", "log"}, {"--criterion", "bad", "--evidence", "log"}, {"--criterion", "1"}, {"--evidence", "log"}} {
		args := []string{"--url", s.URL, "--token-file", tokenFile(t), "disputes", "report", "--job", "job-1", "--if-version", "7", "--summary", "Blocked", "--observed-failure", "Failure"}
		if _, err := run(t, append(args, extra...)...); err == nil {
			t.Fatalf("invalid report allowed: %v", extra)
		}
	}
}

func TestWatchReturnsWhenTheJobHandsOffToDisputeWorkflow(t *testing.T) {
	for _, status := range []string{"disputed", "superseded"} {
		t.Run(status, func(t *testing.T) {
			requests := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				if requests > 1 {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, `{"code":"invalid","message":"watch should have returned"}`)
					return
				}
				io.WriteString(w, `{"id":"job-1","version":7,"status":"`+status+`"}`)
			}))
			defer s.Close()
			out, err := run(t, "--url", s.URL, "--token-file", tokenFile(t), "jobs", "watch", "--job", "job-1", "--interval", "1ms")
			if err == nil || !strings.Contains(err.Error(), "inspect its disputes") || requests != 1 || !strings.Contains(out, status) {
				t.Fatalf("watch hid handoff: %q, %v, requests=%d", out, err, requests)
			}
		})
	}
}
