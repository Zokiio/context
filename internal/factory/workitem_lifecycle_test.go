package factory_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func draftTask(t *testing.T, h jobHarness, title string) factory.WorkItemRecord {
	t.Helper()
	queue := false
	record, err := h.service.CreateWorkItem(context.Background(), h.admin, factory.CreateWorkItemRequest{RequestID: "draft-" + title, ProjectID: h.project.ID, Queue: &queue, Draft: factory.WorkItemDraft{Title: title, Criteria: []string{"The requested result is observable."}}})
	if err != nil {
		t.Fatal(err)
	}
	return record
}
func taskDisposition(t *testing.T, h jobHarness, record factory.WorkItemRecord, action string) factory.WorkItemRecord {
	t.Helper()
	changed, err := h.service.SetWorkItemDisposition(context.Background(), h.admin, factory.SetWorkItemDispositionRequest{RequestID: action + "-" + record.WorkItemID + "-" + record.TicketSHA256, ProjectID: h.project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Action: action})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}
func TestTaskDraftQueueArchiveRestoreAndRestrictedDeletion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil).jobHarness
	draft := draftTask(t, h, "Draft lifecycle")
	if draft.Queued || draft.Archived || !draft.CanDelete || draft.Frozen {
		t.Fatalf("draft = %+v", draft)
	}
	if card := cardFor(t, h.service, h.admin, h.project.ID, draft.WorkItemID); card.Status != "Draft" {
		t.Fatalf("draft card = %+v", card)
	}
	if _, err := h.service.StartExpectations(ctx, h.admin, factory.StartExpectationsRequest{RequestID: "draft-tests", ProjectID: h.project.ID, WorkItemID: draft.WorkItemID}); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("draft author = %v", err)
	}
	if _, err := h.service.Submit(ctx, h.admin, factory.SubmitRequest{RequestID: "draft-submit", ProjectID: h.project.ID, WorkItemID: draft.WorkItemID, ExecutorKey: "fixture-agent", BaseRevision: head(t, h.repository, "HEAD"), Deadline: time.Now().Add(time.Hour)}); !isCode(err, factory.CodeNotReady) {
		t.Fatalf("draft submit = %v", err)
	}
	queued := taskDisposition(t, h, draft, "queue")
	if !queued.Queued || queued.CanDelete {
		t.Fatalf("queued = %+v", queued)
	}
	archived := taskDisposition(t, h, queued, "archive")
	if !archived.Archived || archived.Queued || archived.CanDelete {
		t.Fatalf("archived = %+v", archived)
	}
	board, err := h.service.Board(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range board.Cards {
		if card.WorkItemID == draft.WorkItemID {
			t.Fatal("archive remained on board")
		}
	}
	records, err := h.service.ListWorkItems(ctx, h.admin, h.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range records {
		if record.WorkItemID == draft.WorkItemID && record.Archived {
			found = true
		}
	}
	if !found {
		t.Fatal("archive disappeared from task inventory")
	}
	restored := taskDisposition(t, h, archived, "restore")
	if restored.Queued || restored.Archived || !restored.CanDelete {
		t.Fatalf("restore = %+v", restored)
	}
	request := factory.SetWorkItemDispositionRequest{RequestID: "delete-draft", ProjectID: h.project.ID, WorkItemID: restored.WorkItemID, IfTicketSHA256: restored.TicketSHA256, Action: "delete"}
	for i := 0; i < 2; i++ {
		deleted, err := h.service.SetWorkItemDisposition(ctx, h.admin, request)
		if err != nil || !deleted.Deleted {
			t.Fatalf("delete replay %d = %+v %v", i, deleted, err)
		}
	}
	if _, err := os.Stat(filepath.Join(h.bundle, draft.Path)); !os.IsNotExist(err) {
		t.Fatalf("deleted file = %v", err)
	}
}

func TestDraftCannotRunUnderAutopilotUntilQueued(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupPilot(t, nil, nil)
	existing, err := h.service.GetWorkItem(ctx, h.admin, h.project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	taskDisposition(t, h, existing, "unqueue")
	if _, err := h.service.SetAskPolicy(ctx, h.admin, factory.SetAskPolicyRequest{RequestID: "auto", ProjectID: h.project.ID, BeforeBuild: factory.AskBeforeNever, AfterBuild: factory.AskAfterAlways, Autopilot: true}); err != nil {
		t.Fatal(err)
	}
	draft := draftTask(t, h, "Do not launch on save")
	for i := 0; i < 2; i++ {
		h.service.PilotOnce(ctx)
	}
	jobs, err := h.service.Jobs(ctx, h.admin, factory.JobFilter{ProjectID: h.project.ID})
	if err != nil || len(jobs) != 0 {
		t.Fatalf("draft jobs = %+v %v", jobs, err)
	}
	taskDisposition(t, h, draft, "queue")
	h.service.PilotOnce(ctx)
	jobs, err = h.service.Jobs(ctx, h.admin, factory.JobFilter{ProjectID: h.project.ID})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("queued jobs = %+v %v", jobs, err)
	}
}

func TestTaskDeletionPreservesReferencesAndExecutionHistory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil).jobHarness
	draft := draftTask(t, h, "Referenced draft")
	other := draftTask(t, h, "Dependent draft")
	other.Draft.BlockedBy = []string{draft.WorkItemID}
	if _, err := h.service.UpdateWorkItem(ctx, h.admin, factory.UpdateWorkItemRequest{RequestID: "dependency", ProjectID: h.project.ID, WorkItemID: other.WorkItemID, IfTicketSHA256: other.TicketSHA256, Draft: other.Draft}); err != nil {
		t.Fatal(err)
	}
	referenced, err := h.service.GetWorkItem(ctx, h.admin, h.project.ID, draft.WorkItemID)
	if err != nil {
		t.Fatal(err)
	}
	if referenced.CanDelete || len(referenced.Dependents) != 1 || referenced.Dependents[0].ID != other.WorkItemID {
		t.Fatalf("references = %+v", referenced)
	}
	if _, err := h.service.SetWorkItemDisposition(ctx, h.admin, factory.SetWorkItemDispositionRequest{RequestID: "unsafe-delete", ProjectID: h.project.ID, WorkItemID: draft.WorkItemID, IfTicketSHA256: referenced.TicketSHA256, Action: "delete"}); !isCode(err, factory.CodeConflict) {
		t.Fatalf("delete referenced = %v", err)
	}
	// An ordinary link in an otherwise unknown section is still a reference.
	plain := draftTask(t, h, "Linked in prose")
	source := filepath.Join(h.repository, "guide.md")
	if err := os.WriteFile(source, []byte("## Notes\n\n[Keep this]("+filepath.ToSlash(filepath.Join("records", plain.Path))+")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// readyTestProject uses records as its bundle directory.
	rel, err := filepath.Rel(h.repository, filepath.Join(h.bundle, plain.Path))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("## Notes\n\n[Keep this]("+filepath.ToSlash(rel)+")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linked, err := h.service.GetWorkItem(ctx, h.admin, h.project.ID, plain.WorkItemID)
	if err != nil || linked.CanDelete || !strings.Contains(linked.DeleteReason, "referenced") {
		t.Fatalf("prose reference = %+v %v", linked, err)
	}
	queued := taskDisposition(t, h, plain, "queue")
	job, err := h.service.Submit(ctx, h.admin, factory.SubmitRequest{RequestID: "history", ProjectID: h.project.ID, WorkItemID: plain.WorkItemID, ExecutorKey: "fixture-agent", BaseRevision: head(t, h.repository, "HEAD"), Deadline: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.service.SetWorkItemDisposition(ctx, h.admin, factory.SetWorkItemDispositionRequest{RequestID: "archive-active", ProjectID: h.project.ID, WorkItemID: plain.WorkItemID, IfTicketSHA256: queued.TicketSHA256, Action: "archive"}); !isCode(err, factory.CodeConflict) {
		t.Fatalf("archive active = %v", err)
	}
	if _, err := h.service.Action(ctx, h.admin, factory.JobActionRequest{RequestID: "stop-history", JobID: job.ID, IfVersion: job.Version, Action: factory.ActionCancel}); err != nil {
		t.Fatal(err)
	}
	current, err := h.service.GetWorkItem(ctx, h.admin, h.project.ID, plain.WorkItemID)
	if err != nil {
		t.Fatal(err)
	}
	unqueued := taskDisposition(t, h, current, "unqueue")
	if unqueued.CanDelete || !strings.Contains(unqueued.DeleteReason, "history") {
		t.Fatalf("history deletion = %+v", unqueued)
	}
}

func TestTaskMarkdownLinksAndUnknownSectionsSurviveEdits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil).jobHarness
	record := draftTask(t, h, "Rich description")
	path := filepath.Join(h.bundle, record.Path)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	unknown := "\n## Local knowledge\n\nKeep **exactly** this layout.\n\n- [x] A fact\n"
	if err := os.WriteFile(path, append(raw, unknown...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.bundle, "details.md"), []byte("# Details\nA result.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	record, err = h.service.GetWorkItem(ctx, h.admin, h.project.ID, record.WorkItemID)
	if err != nil {
		t.Fatal(err)
	}
	record.Draft.Description = "# Context\n\nA **formatted** description.\n\n## Examples\n\n```go\nfmt.Println(1)\n```"
	record.Draft.Spec = []string{"details.md"}
	record.Draft.Context = []string{"details.md#details"}
	edited, err := h.service.UpdateWorkItem(ctx, h.admin, factory.UpdateWorkItemRequest{RequestID: "markdown", ProjectID: h.project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Draft: record.Draft})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Draft.Description != record.Draft.Description || !slices.Equal(edited.Draft.Spec, record.Draft.Spec) {
		t.Fatalf("markdown round trip = %+v", edited.Draft)
	}
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), unknown) {
		t.Fatalf("unknown section changed: %s %v", content, err)
	}
	for i, link := range []string{"https://example.com/test.md", "missing.md", "../../../../private.md"} {
		draft := edited.Draft
		draft.Spec = []string{link}
		if _, err := h.service.UpdateWorkItem(ctx, h.admin, factory.UpdateWorkItemRequest{RequestID: fmt.Sprintf("bad-link-%d", i), ProjectID: h.project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: edited.TicketSHA256, Draft: draft}); !isCode(err, factory.CodeInvalid) {
			t.Fatalf("unsafe link %q = %v", link, err)
		}
	}
	readOnly, err := h.service.GetWorkItem(ctx, h.reviewer, h.project.ID, record.WorkItemID)
	if err != nil || len(readOnly.Actions) != 0 {
		t.Fatalf("read-only actions = %+v %v", readOnly.Actions, err)
	}
}

func TestConcurrentTaskEditsKeepOneWinnerAndRejectStaleDisposition(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil).jobHarness
	record := draftTask(t, h, "Concurrent edits")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			draft := record.Draft
			draft.Title = fmt.Sprintf("Edit %d", i)
			_, err := h.service.UpdateWorkItem(ctx, h.admin, factory.UpdateWorkItemRequest{RequestID: fmt.Sprintf("race-%d", i), ProjectID: h.project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Draft: draft})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	passed, stale := 0, 0
	for err := range results {
		if err == nil {
			passed++
		} else if isCode(err, factory.CodeStale) {
			stale++
		} else {
			t.Fatal(err)
		}
	}
	if passed != 1 || stale != 1 {
		t.Fatalf("race passed=%d stale=%d", passed, stale)
	}
	if _, err := h.service.SetWorkItemDisposition(ctx, h.admin, factory.SetWorkItemDispositionRequest{RequestID: "stale-archive", ProjectID: h.project.ID, WorkItemID: record.WorkItemID, IfTicketSHA256: record.TicketSHA256, Action: "archive"}); !isCode(err, factory.CodeStale) {
		t.Fatalf("stale archive = %v", err)
	}
}

func TestTaskPublicationReconcilesAfterRestartWithoutOverwritingChanges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil).jobHarness
	request := factory.CreateWorkItemRequest{RequestID: "recover-create", ProjectID: h.project.ID, Draft: factory.WorkItemDraft{Title: "Recover publication", Criteria: []string{"The result is retained."}}}
	created, err := h.service.CreateWorkItem(ctx, h.admin, request)
	if err != nil {
		t.Fatal(err)
	}
	if err = h.service.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(h.dataDir, "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	var encoded string
	if err = db.QueryRow(`SELECT plan_json FROM work_item_writes WHERE request_id='recover-create'`).Scan(&encoded); err != nil {
		t.Fatal(err)
	}
	var publication struct {
		Files []struct {
			Path          string
			Before, After []byte
		}
	}
	if err = json.Unmarshal([]byte(encoded), &publication); err != nil {
		t.Fatal(err)
	}
	if len(publication.Files) != 2 {
		t.Fatalf("publication = %s", encoded)
	}
	// Model a crash after creating the task but before its project link and receipt.
	manifest := publication.Files[1]
	if err = os.WriteFile(manifest.Path, manifest.Before, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE work_item_writes SET state='prepared' WHERE request_id='recover-create'`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`DELETE FROM command_receipts WHERE operation='create_work_item' AND request_id='recover-create'`); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	changed := append(append([]byte{}, manifest.Before...), []byte("\nAn external edit.\n")...)
	if err = os.WriteFile(manifest.Path, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	service, err := factory.Open(h.dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	held, err := service.GetWorkItem(ctx, h.admin, h.project.ID, created.WorkItemID)
	if err != nil || !held.Frozen {
		t.Fatalf("partial publication = %+v %v", held, err)
	}
	if _, err := service.Submit(ctx, h.admin, factory.SubmitRequest{RequestID: "pending-submit", ProjectID: h.project.ID, WorkItemID: created.WorkItemID, ExecutorKey: "fixture-agent", BaseRevision: head(t, h.repository, "HEAD"), Deadline: time.Now().Add(time.Hour)}); !isCode(err, factory.CodeConflict) {
		t.Fatalf("pending admission = %v", err)
	}
	if _, err = service.CreateWorkItem(ctx, h.admin, request); !isCode(err, factory.CodeStale) {
		t.Fatalf("conflicting recovery = %v", err)
	}
	retained, err := os.ReadFile(manifest.Path)
	if err != nil || string(retained) != string(changed) {
		t.Fatal("recovery overwrote external changes")
	}
	if err = os.WriteFile(manifest.Path, manifest.Before, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReconcileWorkItemPublications(ctx, h.reviewer, h.project.ID); !isCode(err, factory.CodeForbidden) {
		t.Fatalf("reader reconciled save: %v", err)
	}
	recoveredRecords, err := service.ReconcileWorkItemPublications(ctx, h.admin, h.project.ID)
	if err != nil || len(recoveredRecords) != 1 {
		t.Fatalf("reconcile = %+v %v", recoveredRecords, err)
	}
	recovered := recoveredRecords[0]
	if recovered.WorkItemID != created.WorkItemID || !recovered.Queued || recovered.Frozen {
		t.Fatalf("recovered = %+v", recovered)
	}
	if replay, err := service.CreateWorkItem(ctx, h.admin, request); err != nil || replay.WorkItemID != created.WorkItemID {
		t.Fatalf("replay = %+v %v", replay, err)
	}
	if matches, _ := filepath.Glob(filepath.Join(h.bundle, "recover-publication*.md")); len(matches) != 1 {
		t.Fatalf("duplicate publication: %v", matches)
	}
	// A crash after both file effects but before receipt commit reconciles itself
	// during startup, without a browser retaining the original request ID.
	if err = service.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", filepath.Join(h.dataDir, "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE work_item_writes SET state='prepared' WHERE request_id='recover-create'`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`DELETE FROM command_receipts WHERE operation='create_work_item' AND request_id='recover-create'`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	restarted, err := factory.Open(h.dataDir, factory.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	afterRestart, err := restarted.GetWorkItem(ctx, h.admin, h.project.ID, created.WorkItemID)
	if err != nil || afterRestart.PublicationPending || afterRestart.Frozen || !afterRestart.Queued {
		t.Fatalf("automatic recovery = %+v %v", afterRestart, err)
	}
}
