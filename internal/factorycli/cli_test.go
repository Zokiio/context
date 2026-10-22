package factorycli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	cli "github.com/urfave/cli/v3"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, stderr bytes.Buffer
	root := &cli.Command{Name: "ctx", Writer: &stderr, ErrWriter: &stderr, Commands: []*cli.Command{Command(&out, &stderr)}, ExitErrHandler: func(context.Context, *cli.Command, error) {}}
	e := root.Run(context.Background(), append([]string{"ctx", "factory"}, args...))
	return out.String(), e
}
func tokenFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if e := os.WriteFile(path, []byte("private-test-token\n"), 0600); e != nil {
		t.Fatal(e)
	}
	return path
}
func TestCLIUsesActualHTTPAndServiceRejection(t *testing.T) {
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer private-test-token" {
			t.Error("missing bearer header")
		}
		if strings.Contains(r.URL.String(), "private-test-token") {
			t.Error("token entered URL")
		}
		if r.Method != http.MethodGet || r.URL.Path != "/v1/projects" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"code":"forbidden","message":"project access denied"}`)
	}))
	defer s.Close()
	out, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "projects", "list")
	if e == nil || out != "" || requests != 1 {
		t.Fatalf("denied CLI output=%q error=%v requests=%d", out, e, requests)
	}
	if strings.Contains(e.Error(), "private-test-token") {
		t.Fatal("token in error")
	}
}
func TestCLIProfilesPreserveLiteralArgv(t *testing.T) {
	var got factory.RegisterExecutorRequest
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/executors" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL)
		}
		if e := json.NewDecoder(r.Body).Decode(&got); e != nil {
			t.Fatal(e)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"key":"native"}`)
	}))
	defer s.Close()
	literal := "$(touch should-never-exist); && `echo x`,literal"
	out, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "executors", "register", "--key", "native", "--program", "/existing/agent", "--arg", literal, "--request-id", "profile-1")
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Command) != 2 || got.Command[1] != literal || got.RequestID != "profile-1" || !strings.Contains(out, "native") {
		t.Fatalf("profile %+v output %s", got, out)
	}
}
func TestCLIAcceptSendsExactCandidateAndLeavesDecisionToService(t *testing.T) {
	var got factory.AcceptRequest
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/jobs/job-1/acceptance" {
			t.Fatal(r.URL.Path)
		}
		if e := json.NewDecoder(r.Body).Decode(&got); e != nil {
			t.Fatal(e)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		io.WriteString(w, `{"code":"not_ready","message":"required check unavailable"}`)
	}))
	defer s.Close()
	out, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "jobs", "accept", "--job", "job-1", "--candidate", "candidate-1", "--requirements", "requirements-1", "--if-version", "7", "--request-id", "accept-1")
	if e == nil || out != "" {
		t.Fatalf("missing verification accepted: %q %v", out, e)
	}
	if got.CandidateRevision != "candidate-1" || got.RequirementFingerprint != "requirements-1" || got.IfVersion != 7 || got.RequestID != "accept-1" {
		t.Fatalf("accept %+v", got)
	}
}
func TestCLISubmitUsesAdminPolicyAndExactDeadline(t *testing.T) {
	var got factory.SubmitRequest
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e := json.NewDecoder(r.Body).Decode(&got); e != nil {
			t.Fatal(e)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"job-1","status":"queued"}`)
	}))
	defer s.Close()
	deadline := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	out, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "jobs", "submit", "--project", "project-1", "--work-item", "work-1", "--base", "HEAD", "--executor", "native", "--repair-limit", "2", "--deadline", deadline.Format(time.RFC3339), "--request-id", "admit-1")
	if e != nil || !strings.Contains(out, "queued") {
		t.Fatalf("submit %q %v", out, e)
	}
	if !got.Deadline.Equal(deadline) || got.ProjectID != "project-1" || got.WorkItemID != "work-1" || got.RepairLimit != 2 {
		t.Fatalf("submit %+v", got)
	}
}
func TestTokenInputMustBePrivateAndNotSymlinked(t *testing.T) {
	path := tokenFile(t)
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := run(t, "--token-file", path, "me"); e == nil {
		t.Fatal("public token file allowed")
	}
	if e := os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
	link := filepath.Join(t.TempDir(), "link")
	if e := os.Symlink(path, link); e != nil {
		t.Fatal(e)
	}
	if _, e := run(t, "--token-file", link, "me"); e == nil {
		t.Fatal("symlink token allowed")
	}
}
func TestNonLoopbackServeRequiresTLSBeforeOpeningState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	_, e := run(t, "serve", "--data-dir", dir, "--listen", "0.0.0.0:8741")
	if e == nil || !strings.Contains(e.Error(), "TLS") && !strings.Contains(e.Error(), "tls-cert") {
		t.Fatalf("non-loopback %v", e)
	}
	if _, e = os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("invalid listener opened operational state")
	}
	for _, tc := range []struct {
		addr, cert, key string
		valid           bool
	}{{"127.0.0.1:0", "", "", true}, {"[::1]:0", "", "", true}, {"0.0.0.0:0", "cert", "key", true}, {"localhost:0", "", "", false}, {"127.0.0.1:0", "cert", "", false}} {
		if e := validateListener(tc.addr, tc.cert, tc.key); (e == nil) != tc.valid {
			t.Errorf("listener %+v %v", tc, e)
		}
	}
}
func TestTokenIssueWritesPrivateFileWithoutSecretOutput(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/users/worker-1/tokens" {
			t.Fatal(r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"credential-1","userId":"worker-1","token":"one-time-output-token"}`)
	}))
	defer s.Close()
	dest := filepath.Join(t.TempDir(), "issued-token")
	out, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "users", "token", "--user", "worker-1", "--out", dest)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(out, "one-time-output-token") {
		t.Fatal("plaintext token in JSON output")
	}
	secret, e := readSecret(dest)
	if e != nil || secret != "one-time-output-token" {
		t.Fatalf("private output %q %v", secret, e)
	}
	if _, e = run(t, "--url", s.URL, "--token-file", tokenFile(t), "users", "token", "--user", "worker-1", "--out", dest); e == nil {
		t.Fatal("overwrote existing token")
	}
}
func TestInvalidInvocationDoesNotCallHTTP(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("invalid invocation called HTTP") }))
	defer s.Close()
	_, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "jobs", "submit", "--project", "p", "--work-item", "w", "--base", "HEAD", "--executor", "native", "--deadline", "invalid")
	if e == nil {
		t.Fatal("invalid timestamp accepted")
	}
	if !errors.Is(e, context.Canceled) && !strings.Contains(e.Error(), "RFC3339") {
		t.Fatal(e)
	}
}

func TestIncompleteProjectInspectionKeepsJSONAndFails(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"project":{"id":"p"},"result":{"complete":false,"inventoryComplete":false,"diagnostics":[{"code":"missing_source"}]}}`)
	}))
	defer s.Close()
	out, e := run(t, "--url", s.URL, "--token-file", tokenFile(t), "projects", "inspect", "--project", "p")
	if e == nil || !strings.Contains(out, "missing_source") {
		t.Fatalf("partial inspection %q %v", out, e)
	}
}
