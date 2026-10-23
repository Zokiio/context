package factoryhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factoryclient"
	"github.com/Zokiio/context/internal/factoryhttp"
)

type installation struct {
	service *factory.Service
	server  *httptest.Server
	admin   factory.Principal
	token   string
	dataDir string
}

func start(t *testing.T) *installation {
	t.Helper()
	dir := t.TempDir()
	service, err := factory.Open(dir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	boot, err := service.Bootstrap(context.Background(), factory.BootstrapRequest{Username: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	actor, err := service.Authenticate(context.Background(), boot.Token)
	if err != nil {
		t.Fatal(err)
	}
	instance := &installation{service: service, server: httptest.NewServer(factoryhttp.NewHandler(service)), admin: actor, token: boot.Token, dataDir: dir}
	t.Cleanup(func() { instance.server.Close(); _ = instance.service.Close() })
	return instance
}

func call(t *testing.T, base, token, method, path, body string, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, raw
}

func TestAuthenticationBoundedJSONAndBrowserPolicy(t *testing.T) {
	instance := start(t)
	base := instance.server.URL
	for _, test := range []struct {
		name, token, method, path, body string
		headers                         map[string]string
		status                          int
	}{
		{name: "no identity", method: "GET", path: "/v1/me", status: 401},
		{name: "invalid identity", token: "unrecognized-secret", method: "GET", path: "/v1/me", status: 401},
		{name: "URL credential", token: instance.token, method: "GET", path: "/v1/me?token=unrecognized-secret", status: 400},
		{name: "cross origin", token: instance.token, method: "GET", path: "/v1/me", headers: map[string]string{"Origin": "https://attacker.invalid"}, status: 403},
		{name: "actor spoof", token: instance.token, method: "POST", path: "/v1/users", body: `{"requestId":"spoof","username":"bad","kind":"human","actor":{"installationAdmin":true}}`, status: 400},
		{name: "multiple JSON values", token: instance.token, method: "POST", path: "/v1/users", body: `{} {}`, status: 400},
		{name: "malformed JSON", token: instance.token, method: "POST", path: "/v1/users", body: `{`, status: 400},
		{name: "null body", token: instance.token, method: "POST", path: "/v1/users", body: `null`, status: 400},
		{name: "wrong media type", token: instance.token, method: "POST", path: "/v1/users", body: `{}`, headers: map[string]string{"Content-Type": "text/plain"}, status: 415},
		{name: "oversized JSON", token: instance.token, method: "POST", path: "/v1/users", body: `{"username":"` + strings.Repeat("x", 2<<20) + `"}`, status: 413},
		{name: "subject mismatch", token: instance.token, method: "POST", path: "/v1/jobs/path-job/actions", body: `{"requestId":"mismatch","jobId":"other-job","ifVersion":1,"action":"cancel"}`, status: 400},
		{name: "invalid page limit", token: instance.token, method: "GET", path: "/v1/jobs?limit=-1", status: 400},
		{name: "unknown endpoint", token: instance.token, method: "GET", path: "/v1/nonexistent", status: 404},
		{name: "wrong method", token: instance.token, method: "DELETE", path: "/v1/users", status: 405},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, headers, raw := call(t, base, test.token, test.method, test.path, test.body, test.headers)
			if status != test.status {
				t.Fatalf("status = %d, want %d, body = %s", status, test.status, raw)
			}
			var failure factory.Error
			if err := json.Unmarshal(raw, &failure); err != nil || failure.Code == "" {
				t.Fatalf("not a JSON error: %s, %v", raw, err)
			}
			if bytes.Contains(raw, []byte("unrecognized-secret")) || bytes.Contains(raw, []byte(instance.token)) {
				t.Fatal("error echoed a credential")
			}
			if headers.Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("API permits cross-origin access")
			}
		})
	}
	for _, path := range []string{"/", "/assets/app.js", "/assets/app.css"} {
		status, headers, raw := call(t, base, "", "GET", path, "", nil)
		if status != 200 || len(raw) == 0 {
			t.Fatalf("missing asset %s: %d", path, status)
		}
		if !strings.Contains(headers.Get("Content-Security-Policy"), "frame-ancestors 'none'") || headers.Get("X-Content-Type-Options") != "nosniff" || headers.Get("Cache-Control") != "no-store" {
			t.Fatalf("missing browser protection on %s", path)
		}
		if path == "/assets/app.js" && bytes.Contains(raw, []byte("innerHTML")) {
			t.Fatal("browser permits untrusted HTML insertion")
		}
	}
}

func TestSelfHostedFontsAndLicenses(t *testing.T) {
	instance := start(t)
	for _, family := range []string{"plus-jakarta-sans", "jetbrains-mono"} {
		for _, subset := range []string{"latin", "latin-ext"} {
			path := "/assets/fonts/" + family + "-" + subset + ".woff2"
			status, headers, raw := call(t, instance.server.URL, "", "GET", path, "", nil)
			if status != 200 || headers.Get("Content-Type") != "font/woff2" || !bytes.HasPrefix(raw, []byte("wOF2")) {
				t.Fatalf("font not served as WOFF2 on %s: status %d, type %q", path, status, headers.Get("Content-Type"))
			}
			if !strings.Contains(headers.Get("Content-Security-Policy"), "font-src 'self';") || headers.Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("missing same-origin font policy on %s", path)
			}
		}
		path := "/assets/fonts/" + family + "-OFL.txt"
		status, headers, raw := call(t, instance.server.URL, "", "GET", path, "", nil)
		if status != 200 || headers.Get("Content-Type") != "text/plain; charset=utf-8" || !bytes.Contains(raw, []byte("SIL OPEN FONT LICENSE Version 1.1")) {
			t.Fatalf("font license missing on %s: status %d", path, status)
		}
	}
}

func TestHTTPUserTokensAndDisableEnforceCurrentIdentity(t *testing.T) {
	instance := start(t)
	ctx := context.Background()
	admin, err := factoryclient.New(instance.server.URL, instance.token)
	if err != nil {
		t.Fatal(err)
	}
	u, err := admin.CreateUser(ctx, factory.CreateUserRequest{RequestID: "create-user", Username: "teammate", Kind: factory.UserKindHuman})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := admin.IssueToken(ctx, factory.IssueTokenRequest{RequestID: "issue-token", UserID: u.ID, Label: "browser"})
	if err != nil {
		t.Fatal(err)
	}
	client, err := factoryclient.New(instance.server.URL, issued.Token)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := client.Me(ctx)
	if err != nil || identity.UserID != u.ID || identity.InstallationAdmin {
		t.Fatalf("unexpected authenticated identity: %+v, %v", identity, err)
	}
	if _, err := client.CreateUser(ctx, factory.CreateUserRequest{RequestID: "escalate", Username: "intruder", Kind: factory.UserKindHuman, InstallationAdmin: true}); err == nil {
		t.Fatal("ordinary identity created an administrator")
	}
	if _, err := admin.SetUserDisabled(ctx, factory.SetUserDisabledRequest{RequestID: "disable", UserID: u.ID, Disabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Me(ctx); err == nil {
		t.Fatal("disabled user kept API access")
	}
	if _, err := admin.SetUserDisabled(ctx, factory.SetUserDisabledRequest{RequestID: "enable", UserID: u.ID, Disabled: false}); err != nil {
		t.Fatal(err)
	}
	if err := admin.RevokeToken(ctx, factory.RevokeTokenRequest{RequestID: "revoke", TokenID: issued.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Me(ctx); err == nil {
		t.Fatal("revoked credential kept API access")
	}
	if _, err := admin.Me(ctx); err != nil {
		t.Fatalf("unrelated administrator token stopped working: %v", err)
	}
}

func user(t *testing.T, instance *installation, name string) (factory.User, factory.Principal, string) {
	t.Helper()
	ctx := context.Background()
	u, err := instance.service.CreateUser(ctx, instance.admin, factory.CreateUserRequest{RequestID: "create-" + name, Username: name, Kind: factory.UserKindHuman})
	if err != nil {
		t.Fatal(err)
	}
	token, err := instance.service.IssueToken(ctx, instance.admin, factory.IssueTokenRequest{RequestID: "token-" + name, UserID: u.ID, Label: "test"})
	if err != nil {
		t.Fatal(err)
	}
	actor, err := instance.service.Authenticate(ctx, token.Token)
	if err != nil {
		t.Fatal(err)
	}
	return u, actor, token.Token
}

func project(t *testing.T, instance *installation, id string) factory.Project {
	t.Helper()
	repo := t.TempDir()
	bundle := filepath.Join(repo, "records")
	if err := os.MkdirAll(bundle, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("---\ntype: Project\nid: %s\ntitle: Project %s\n---\n\n## Goals\n\nExercise the factory.\n\n## Current commitments\n\n- [Task](work.md)\n\n## Open decisions\n\nNone\n", id, id)
	ticket := "---\ntype: WorkItem\nid: work-1\ntitle: A fixture task\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n\n- Produce a verified candidate.\n\n## Blocked by\n\nNone\n\n## Blocked by decisions\n\nNone\n"
	for name, text := range map[string]string{"project.md": manifest, "work.md": ticket} {
		if err := os.WriteFile(filepath.Join(bundle, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture input"}} {
		command := exec.Command("git", args...)
		command.Dir = repo
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		if raw, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, raw)
		}
	}
	p, err := instance.service.RegisterProject(context.Background(), instance.admin, factory.RegisterProjectRequest{RequestID: "project-" + id, BundlePath: bundle, RepositoryPath: repo, AllowedSourceRoots: []string{repo}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"fixture-check"}, AllowedExecutors: []string{"fixture-executor"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 3600, Fixture: true}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProjectIsolationAdmissionReplayStaleAndRestart(t *testing.T) {
	instance := start(t)
	ctx := context.Background()
	_, err := instance.service.RegisterExecutor(ctx, instance.admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-executor", DisplayName: "Fixture", Command: []string{"fixture"}, Fixture: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = instance.service.RegisterCheck(ctx, instance.admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Fixture check", Command: []string{"fixture"}, Environment: "offline", Fixture: true})
	if err != nil {
		t.Fatal(err)
	}
	p1 := project(t, instance, "project-one")
	p2 := project(t, instance, "project-two")
	reader, _, readerToken := user(t, instance, "reader")
	coordinator, _, coordinatorToken := user(t, instance, "coordinator")
	for _, grant := range []factory.GrantProjectRequest{{RequestID: "read", ProjectID: p1.ID, UserID: reader.ID, Capabilities: []factory.Capability{factory.CapabilityRead}}, {RequestID: "coordinate", ProjectID: p1.ID, UserID: coordinator.ID, Capabilities: []factory.Capability{factory.CapabilityRead, factory.CapabilityCoordinate}}} {
		if _, err := instance.service.GrantProject(ctx, instance.admin, grant); err != nil {
			t.Fatal(err)
		}
	}
	readerClient, err := factoryclient.New(instance.server.URL, readerToken)
	if err != nil {
		t.Fatal(err)
	}
	visible, err := readerClient.ListProjects(ctx)
	if err != nil || len(visible) != 1 || visible[0].ID != p1.ID {
		t.Fatalf("visible projects = %v, err = %v", visible, err)
	}
	for _, path := range []string{"/v1/projects/" + string(p2.ID) + "/orientation", "/v1/projects/" + string(p2.ID) + "/capabilities"} {
		status, _, raw := call(t, instance.server.URL, readerToken, "GET", path, "", nil)
		if status != 404 || bytes.Contains(raw, []byte(p2.Title)) || bytes.Contains(raw, []byte(p2.BundlePath)) {
			t.Fatalf("project isolation failed: %d %s", status, raw)
		}
	}
	status, _, _ := call(t, instance.server.URL, readerToken, "GET", "/v1/users", "", nil)
	if status != 403 {
		t.Fatalf("reader can list users: %d", status)
	}
	submission := factory.SubmitRequest{RequestID: "same-submission", ProjectID: p1.ID, WorkItemID: "work-1", BaseRevision: "HEAD", ExecutorKey: "fixture-executor", RepairLimit: 1, Deadline: time.Now().Add(5 * time.Minute).UTC()}
	if _, err := readerClient.Submit(ctx, submission); err == nil {
		t.Fatal("reader submitted work")
	} else {
		var denied *factoryclient.APIError
		if !errors.As(err, &denied) || denied.Status != http.StatusForbidden {
			t.Fatalf("expected permission denial, got %v", err)
		}
	}
	coordinatorClient, err := factoryclient.New(instance.server.URL, coordinatorToken)
	if err != nil {
		t.Fatal(err)
	}
	job, err := coordinatorClient.Submit(ctx, submission)
	if err != nil {
		t.Fatal(err)
	}
	if !job.Admission.ContextComplete || job.Admission.ContextSHA256 == "" || len(job.Admission.Sources) == 0 {
		t.Fatal("admission did not retain complete context")
	}
	replay, err := coordinatorClient.Submit(ctx, submission)
	if err != nil || replay.ID != job.ID {
		t.Fatalf("submission replay duplicated job: %v, %v", replay.ID, err)
	}
	conflicting := submission
	conflicting.RequestID = "competing-claim"
	if _, err := coordinatorClient.Submit(ctx, conflicting); err == nil {
		t.Fatal("conflicting claim succeeded")
	}
	_, err = coordinatorClient.Action(ctx, factory.JobActionRequest{RequestID: "stale-action", JobID: job.ID, IfVersion: job.Version + 100, Action: factory.ActionCancel})
	var stale *factoryclient.APIError
	if !errors.As(err, &stale) || stale.Status != 409 {
		t.Fatalf("stale mutation was not rejected: %v", err)
	}
	instance.server.Close()
	if err := instance.service.Close(); err != nil {
		t.Fatal(err)
	}
	instance.service, err = factory.Open(instance.dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	instance.server = httptest.NewServer(factoryhttp.NewHandler(instance.service))
	restarted, err := factoryclient.New(instance.server.URL, coordinatorToken)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := restarted.GetJob(ctx, job.ID)
	if err != nil || retained.ID != job.ID || retained.Admission.ContextSHA256 != job.Admission.ContextSHA256 {
		t.Fatalf("restart lost admitted job: %+v, %v", retained, err)
	}
	replayed, err := restarted.Submit(ctx, submission)
	if err != nil || replayed.ID != job.ID {
		t.Fatalf("restart lost request receipt: %v, %v", replayed.ID, err)
	}
	status, _, _ = call(t, instance.server.URL, readerToken, "POST", "/v1/users", `{"requestId":"forbidden","username":"intruder","kind":"human","installationAdmin":true}`, nil)
	if status != 403 {
		t.Fatalf("reader escalated permissions: %d", status)
	}
}

func TestBoardAskPlanAndApproveRoutes(t *testing.T) {
	instance := start(t)
	ctx := context.Background()
	if _, err := instance.service.RegisterExecutor(ctx, instance.admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-executor", DisplayName: "Fixture", Command: []string{"fixture"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := instance.service.RegisterCheck(ctx, instance.admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Fixture", Command: []string{"fixture"}, Environment: "local", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	p := project(t, instance, "routes")
	base, path := instance.server.URL, "/v1/projects/"+string(p.ID)
	status, _, raw := call(t, base, instance.token, "GET", path+"/board", "", nil)
	var board factory.ProjectBoard
	if status != 200 || json.Unmarshal(raw, &board) != nil || len(board.Cards) != 1 || board.Cards[0].Column != factory.ColumnNext || board.Cards[0].Decision != factory.CardDecisionPlan {
		t.Fatalf("board = %d %s", status, raw)
	}
	if status, _, raw = call(t, base, instance.token, "PUT", path+"/ask", `{"requestId":"other","projectId":"elsewhere","beforeBuild":"always","afterBuild":"always"}`, nil); status != 400 {
		t.Fatalf("ask policy for another project = %d %s", status, raw)
	}
	status, _, raw = call(t, base, instance.token, "PUT", path+"/ask", `{"requestId":"ask","beforeBuild":"when_unclear","afterBuild":"always","autopilot":true}`, nil)
	var saved factory.Project
	if status != 200 || json.Unmarshal(raw, &saved) != nil || saved.Policy.Ask.Before() != factory.AskBeforeWhenUnclear || !saved.Policy.Ask.Autopilot {
		t.Fatalf("ask policy = %d %s", status, raw)
	}
	status, _, raw = call(t, base, instance.token, "POST", path+"/plan", `{"requestId":"plan","workItemId":"work-1"}`, nil)
	var plan factory.PlanResult
	if status != 200 || json.Unmarshal(raw, &plan) != nil || plan.Job == nil {
		t.Fatalf("plan = %d %s", status, raw)
	}
	body := fmt.Sprintf(`{"requestId":"approve","jobId":"elsewhere","ifVersion":%d,"candidateRevision":"x","requirementFingerprint":"y"}`, plan.Job.Version)
	if status, _, raw = call(t, base, instance.token, "POST", "/v1/jobs/"+string(plan.Job.ID)+"/approve", body, nil); status != 400 {
		t.Fatalf("approval naming another job = %d %s", status, raw)
	}
}
