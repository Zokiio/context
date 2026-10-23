package factoryintegration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
	"github.com/Zokiio/context/internal/factorycli"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// One person runs the whole flow against solo mode: register, submit, let the
// in-process workers implement and verify, then review and accept their own job.
func TestSoloModeRunsOnePersonsWholeFlow(t *testing.T) {
	data := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	stdout, stderr := &lockedBuffer{}, &lockedBuffer{}
	done := make(chan error, 1)
	go func() {
		done <- factorycli.Command(stdout, stderr).Run(ctx, []string{"factory", "solo", "--data-dir", data, "--listen", "127.0.0.1:0", "--poll", "200ms"})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("solo mode did not stop")
		}
	})
	var endpoint string
	waitFor(t, 15*time.Second, func() bool {
		match := regexp.MustCompile(`at (http://127\.0\.0\.1:\d+)`).FindStringSubmatch(stdout.String())
		if match != nil {
			endpoint = match[1]
		}
		return endpoint != ""
	}, func() string { return stdout.String() + stderr.String() })
	token, err := os.ReadFile(filepath.Join(data, "solo", "operator.token"))
	if err != nil {
		t.Fatal(err)
	}
	operator := client(t, endpoint, strings.TrimSpace(string(token)))

	repository, bundle, base := repositoryFixture(t)
	base = strings.TrimSpace(base)
	const workID = "ec352583-bd7e-4a59-9576-46e129c76c6b"
	call[factory.ExecutorProfile](t, operator, "POST", "/v1/executors", factory.RegisterExecutorRequest{RequestID: "executor", Key: "offline", DisplayName: "Offline fixture", Fixture: true, Command: []string{os.Args[0], "-test.run=^TestFactoryFixtureProcess$", "--", "factory-fixture-implement"}})
	call[factory.CheckProfile](t, operator, "POST", "/v1/checks", factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Fixture check", Environment: "local", Fixture: true, Command: []string{os.Args[0], "-test.run=^TestFactoryFixtureProcess$", "--", "factory-fixture-verify"}})
	project := call[factory.Project](t, operator, "POST", "/v1/projects", factory.RegisterProjectRequest{RequestID: "project", BundlePath: bundle, RepositoryPath: repository, AllowedSourceRoots: []string{repository}, Policy: factory.ProjectPolicy{RequiredChecks: []string{"fixture-check"}, AllowedExecutors: []string{"offline"}, MaxRepairAttempts: 1, MaxJobDurationSeconds: 3600, Fixture: true, ProtectedPaths: []string{"tests/"}, AllowCoordinatorReview: true}})

	var job factory.Job
	waitFor(t, 10*time.Second, func() bool {
		var err error
		err = operator.DoJSON(context.Background(), "POST", "/v1/jobs", factory.SubmitRequest{RequestID: "submit", ProjectID: project.ID, WorkItemID: workID, BaseRevision: base, ExecutorKey: "offline", RepairLimit: 1, Deadline: time.Now().Add(30 * time.Minute)}, &job)
		return err == nil
	}, func() string { return stderr.String() })
	jobPath := "/v1/jobs/" + string(job.ID)
	waitFor(t, 60*time.Second, func() bool {
		job = call[factory.Job](t, operator, "GET", jobPath, nil)
		return job.Status == factory.JobReadyForReview
	}, func() string { return string(job.Status) + "\n" + stderr.String() })

	job = call[factory.Job](t, operator, "POST", jobPath+"/reviews", factory.ReviewRequest{RequestID: "review", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Decision: factory.ReviewApproved, Summary: "Solo review of the fixture candidate."})
	job = call[factory.Job](t, operator, "POST", jobPath+"/acceptance", factory.AcceptRequest{RequestID: "accept", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision, RequirementFingerprint: job.Admission.TicketSHA256, Summary: "Solo acceptance."})
	if job.Status != factory.JobAccepted || job.AcceptanceStatus != "valid" {
		t.Fatalf("solo job = %s acceptance %s", job.Status, job.AcceptanceStatus)
	}
	job = call[factory.Job](t, operator, "POST", jobPath+"/merge", factory.MergeRequest{RequestID: "merge", JobID: job.ID, IfVersion: job.Version, CandidateRevision: job.CandidateRevision})
	if job.Merge == nil || job.Merge.Mode != factory.MergeModeCheckout || job.Merge.TicketNote != "recorded" {
		t.Fatalf("solo merge = %+v", job.Merge)
	}
	if _, err := os.Stat(filepath.Join(repository, "waymark-fixture.txt")); err != nil {
		t.Fatalf("merged candidate is not in the project checkout: %v", err)
	}
	if tip := strings.TrimSpace(git(t, repository, "rev-parse", "HEAD")); tip != job.Merge.Revision {
		t.Fatalf("checkout HEAD %s is not the merge %s", tip, job.Merge.Revision)
	}
}

func waitFor(t *testing.T, timeout time.Duration, ready func() bool, describe func() string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting; state:\n%s", describe())
		}
		time.Sleep(100 * time.Millisecond)
	}
}
