package factory_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Zokiio/context/internal/factory"
)

func TestAPersonAddsAWorkItemThatJoinsTheBoard(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil)
	draft := factory.WorkItemDraft{Title: "Count words: fast path", Description: "The preview counts words in every heading.", Criteria: []string{"WordCount(\"a b\") is 2.", "  WordCount(\"\") is 0.  "}, BlockedBy: []string{"work-test"}}
	request := factory.CreateWorkItemRequest{RequestID: "add", ProjectID: h.project.ID, Draft: draft}
	for name, actor := range map[string]factory.Principal{"reviewer": h.reviewer, "worker": h.implementer} {
		if _, err := h.service.CreateWorkItem(ctx, actor, request); !isCode(err, factory.CodeForbidden) {
			t.Fatalf("%s added work: %v", name, err)
		}
	}
	for name, invalid := range map[string]factory.WorkItemDraft{
		"no title":           {Criteria: draft.Criteria},
		"no requirements":    {Title: "Empty"},
		"unknown dependency": {Title: "Lost", Criteria: draft.Criteria, BlockedBy: []string{"work-missing"}},
	} {
		if _, err := h.service.CreateWorkItem(ctx, h.admin, factory.CreateWorkItemRequest{RequestID: name, ProjectID: h.project.ID, Draft: invalid}); !isCode(err, factory.CodeInvalid) {
			t.Fatalf("%s = %v", name, err)
		}
	}
	created, err := h.service.CreateWorkItem(ctx, h.admin, request)
	if err != nil {
		t.Fatal(err)
	}
	if created.Path != "count-words-fast-path.md" || created.Draft.Title != draft.Title || !slices.Equal(created.Draft.Criteria, []string{"WordCount(\"a b\") is 2.", "WordCount(\"\") is 0."}) || created.Frozen {
		t.Fatalf("created = %+v", created)
	}
	if again, err := h.service.CreateWorkItem(ctx, h.admin, request); err != nil || again.WorkItemID != created.WorkItemID {
		t.Fatalf("a retried add made another item: %+v, %v", again, err)
	}
	if matches, _ := filepath.Glob(filepath.Join(h.bundle, "count-words*.md")); len(matches) != 1 {
		t.Fatalf("files written = %v", matches)
	}
	manifest, err := os.ReadFile(filepath.Join(h.bundle, "project.md"))
	if err != nil || !strings.Contains(string(manifest), "- [Count words: fast path](count-words-fast-path.md)") {
		t.Fatalf("the project does not list the new work: %s", manifest)
	}
	card := cardFor(t, h.service, h.admin, h.project.ID, created.WorkItemID)
	if card.Title != draft.Title || card.Column != factory.ColumnNext || !slices.Equal(card.BlockedBy, []string{"work-test"}) || card.Status != "Waits for Ready work" {
		t.Fatalf("new card = %+v", card)
	}
}

func TestEditingAWorkItemRewritesItsRecordUntilABuildHoldsIt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupExpectations(t, fakeAuthor(t, nil), nil)
	record, err := h.service.GetWorkItem(ctx, h.admin, h.project.ID, "work-test")
	if err != nil {
		t.Fatal(err)
	}
	if record.Draft.Title != "Ready work" || !slices.Equal(record.Draft.Criteria, []string{"The change works."}) || len(record.Draft.BlockedBy) != 0 || record.TicketSHA256 == "" {
		t.Fatalf("record = %+v", record)
	}
	edit := factory.UpdateWorkItemRequest{RequestID: "edit", ProjectID: h.project.ID, WorkItemID: "work-test", IfTicketSHA256: record.TicketSHA256, Draft: factory.WorkItemDraft{Title: "Ready work, sharpened", Description: "Why it matters.", Criteria: []string{"The change works.", "ReverseWords(\"a  b\") is \"b  a\",\nkeeping the spaces."}}}
	edited, err := h.service.UpdateWorkItem(ctx, h.admin, edit)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Draft.Title != "Ready work, sharpened" || edited.Draft.Description != "Why it matters." || !slices.Equal(edited.Draft.Criteria, []string{"The change works.", "ReverseWords(\"a  b\") is \"b  a\", keeping the spaces."}) || edited.TicketSHA256 == record.TicketSHA256 {
		t.Fatalf("edited = %+v", edited)
	}
	raw, err := os.ReadFile(filepath.Join(h.bundle, "work.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kept := range []string{"## Spec\nNone", "## Blocked by decisions\nNone", "admin edited the work item"} {
		if !strings.Contains(string(raw), kept) {
			t.Fatalf("the rewritten record lost %q:\n%s", kept, raw)
		}
	}
	stale := edit
	stale.RequestID = "stale"
	if _, err := h.service.UpdateWorkItem(ctx, h.admin, stale); !isCode(err, factory.CodeStale) {
		t.Fatalf("an edit from an old copy = %v", err)
	}
	if _, err := h.service.Submit(ctx, h.coordinator, factory.SubmitRequest{RequestID: "job", ProjectID: h.project.ID, WorkItemID: "work-test", BaseRevision: head(t, h.repository, "HEAD"), ExecutorKey: "fixture-agent", Deadline: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	frozen, err := h.service.GetWorkItem(ctx, h.admin, h.project.ID, "work-test")
	if err != nil || !frozen.Frozen {
		t.Fatalf("a work item a build holds = %+v, %v", frozen, err)
	}
	held := edit
	held.RequestID, held.IfTicketSHA256 = "held", frozen.TicketSHA256
	if _, err := h.service.UpdateWorkItem(ctx, h.admin, held); !isCode(err, factory.CodeConflict) {
		t.Fatalf("an edit while a build holds the item = %v", err)
	}
}
