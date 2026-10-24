package factory_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

// Pause the existing fixture after it has read the actual worker input. The
// marker proves the process started; the release file avoids timing assertions.
func gateProposalFixture(t *testing.T, command []string) ([]string, string, func()) {
	t.Helper()
	raw, err := os.ReadFile(command[0])
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitN(string(raw), "\n", 3)
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "cat > ") {
		t.Fatal("proposal fixture must read its input before the gate")
	}
	gate := ": > \"$1\"\nwhile [ ! -f \"$2\" ]; do sleep 0.01; done\n"
	if err := os.WriteFile(command[0], []byte(lines[0]+"\n"+lines[1]+"\n"+gate+lines[2]), 0o700); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	started, release := filepath.Join(dir, "started"), filepath.Join(dir, "release")
	return append(append([]string{}, command...), started, release), started, func() {
		if err := os.WriteFile(release, nil, 0o600); err != nil {
			t.Error(err)
		}
	}
}

func waitProposalFixtureStarted(t *testing.T, marker string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("proposal worker did not consume its input")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertTaskChangesBlockedByProposal(t *testing.T, service *factory.Service, actor factory.Principal, project factory.Project) factory.UpdateWorkItemRequest {
	t.Helper()
	ctx := context.Background()
	record, err := service.GetWorkItem(ctx, actor, project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	if !record.Frozen || record.FrozenReason == "" || len(record.Actions) != 0 {
		t.Fatalf("running proposal did not freeze task actions: %+v", record)
	}
	paths := []string{filepath.Join(project.BundlePath, record.Path), filepath.Join(project.BundlePath, "project.md")}
	before := make([][]byte, len(paths))
	for i, path := range paths {
		before[i], err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	draft := record.Draft
	draft.Description = "A task edit attempted while its proposal worker is running."
	request := factory.UpdateWorkItemRequest{RequestID: "edit-during-proposal", ProjectID: project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Draft: draft}
	if _, err := service.UpdateWorkItem(ctx, actor, request); !isCode(err, factory.CodeConflict) {
		t.Fatalf("edit while proposal is running = %v", err)
	}
	for _, action := range []string{"archive", "unqueue"} {
		if _, err := service.SetWorkItemDisposition(ctx, actor, factory.SetWorkItemDispositionRequest{RequestID: action + "-during-proposal", ProjectID: project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Action: action}); !isCode(err, factory.CodeConflict) {
			t.Fatalf("%s while proposal is running = %v", action, err)
		}
	}
	for i, path := range paths {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before[i], after) {
			t.Fatalf("refused change altered %s: %v", path, err)
		}
	}
	return request
}

func TestTaskChangesWaitForRunningProposalWorkers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("refinement", func(t *testing.T) {
		fixture, _ := fakeRefiner(t, "")
		command, marker, release := gateProposalFixture(t, fixture)
		h := setupRefine(t, command)
		t.Cleanup(release)
		started, err := h.service.StartRefinement(ctx, h.admin, factory.StartRefinementRequest{RequestID: "active-refinement", ProjectID: h.project.ID, WorkItemID: "work-test"})
		if err != nil || started.Status != factory.RefinementRunning {
			t.Fatalf("start refinement = %+v, %v", started, err)
		}
		waitProposalFixtureStarted(t, marker)
		request := assertTaskChangesBlockedByProposal(t, h.service, h.admin, h.project)
		release()
		if completed := h.waitProposed(t, started.ID); completed.Status != factory.RefinementProposed {
			t.Fatalf("refinement did not finish normally: %+v", completed)
		}
		if updated, err := h.service.UpdateWorkItem(ctx, h.admin, request); err != nil || updated.Frozen {
			t.Fatalf("completed refinement still blocked the edit: %+v, %v", updated, err)
		}
	})
	t.Run("expectation", func(t *testing.T) {
		fixture := fakeAuthor(t, map[string]string{"tests/holdout.txt": "checks the requested result"})
		command, marker, release := gateProposalFixture(t, fixture)
		h := setupExpectations(t, command, nil)
		t.Cleanup(release)
		started, err := h.service.StartExpectations(ctx, h.admin, factory.StartExpectationsRequest{RequestID: "active-expectation", ProjectID: h.project.ID, WorkItemID: "work-test"})
		if err != nil || started.Status != factory.ExpectationRunning {
			t.Fatalf("start expectations = %+v, %v", started, err)
		}
		waitProposalFixtureStarted(t, marker)
		request := assertTaskChangesBlockedByProposal(t, h.service, h.admin, h.project)
		release()
		if completed := h.waitExpectation(t, started.ID); completed.Status != factory.ExpectationProposed {
			t.Fatalf("expectation did not finish normally: %+v", completed)
		}
		if updated, err := h.service.UpdateWorkItem(ctx, h.admin, request); err != nil || updated.Frozen {
			t.Fatalf("completed expectation still blocked the edit: %+v, %v", updated, err)
		}
	})
}
