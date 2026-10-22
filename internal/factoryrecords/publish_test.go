package factoryrecords

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
)

func plan(t *testing.T) Plan {
	t.Helper()
	root := t.TempDir()
	bundle := filepath.Join(root, "records")
	if err := os.MkdirAll(bundle, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ntype: Project\nid: test-project\ntitle: Test\n---\n\n## Goals\n\nTest publication.\n\n## Current commitments\n\n- [Work](work.md)\n\n## Open decisions\n\nNone\n"
	ticket := []byte("---\n# retain this authored comment\ntype: WorkItem\nid: test-work\ntitle: Test publication\ntriage: ready-for-agent\nexecution: in-progress # keep the progress note\n---\n\n## Acceptance criteria\n\n- Retain the accepted criterion bytes.\n\n```text\n## Acceptance\nnot a real section\n```\n\n## Blocked by\n\nNone\n\n## Blocked by decisions\n\nNone\n\n## Acceptance\n\n")
	for name, data := range map[string][]byte{"project.md": []byte(manifest), "work.md": ticket} {
		if err := os.WriteFile(filepath.Join(bundle, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return Plan{BundlePath: bundle, AllowedSourceRoots: []string{root}, ProjectID: "test-project", WorkItemID: "test-work", TicketPath: "work.md", ExpectedTicket: ticket, AcceptancePath: "accepted/one.md", AcceptanceBytes: []byte("---\ntype: Acceptance\nid: acceptance-one\ntitle: Acceptance\nprojectId: test-project\nworkItemId: test-work\n---\n\n## Requirements\n\n## Evidence\n"), EvidencePath: filepath.Join(root, "evidence", "one.md"), EvidenceBytes: []byte("Observed the real check.\n"), LockPath: filepath.Join(root, "locks", "work.lock")}
}

func fingerprints(t *testing.T, p Plan) (string, string) {
	t.Helper()
	v, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: p.BundlePath, AllowedSourceDirs: p.AllowedSourceRoots})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range v.WorkItems {
		if w.ID != nil && *w.ID == p.WorkItemID && w.TicketSHA256 != nil && w.CriteriaSHA256 != nil {
			return *w.TicketSHA256, *w.CriteriaSHA256
		}
	}
	t.Fatal("no work-item fingerprints")
	return "", ""
}

func TestPublicationPreservesRequirementsAndIsReplayable(t *testing.T) {
	p := plan(t)
	beforeTicket, beforeCriteria := fingerprints(t, p)
	// A crash after retaining evidence must not require repeating observations.
	if err := immutable(p.EvidencePath, p.EvidenceBytes); err != nil {
		t.Fatal(err)
	}
	first, err := Reconcile(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Publish(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatalf("non-idempotent publication: %+v %+v", first, again)
	}
	afterTicket, afterCriteria := fingerprints(t, p)
	if beforeTicket != afterTicket || beforeCriteria != afterCriteria {
		t.Fatal("acceptance changed the admitted requirement fingerprints")
	}
	bytesAfter, err := os.ReadFile(first.TicketPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := recordread.ParseDocument(bytesAfter)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Metadata["execution"] != "completed" {
		t.Fatal("execution not recorded")
	}
	if !bytes.Contains(bytesAfter, []byte("retain this authored comment")) || !bytes.Contains(bytesAfter, []byte("keep the progress note")) {
		t.Fatal("authored metadata comments lost")
	}
	if !bytes.Contains(doc.Body, []byte("not a real section")) {
		t.Fatal("fenced content was changed")
	}
}

func TestChangedTicketOrImmutableRecordIsNeverOverwritten(t *testing.T) {
	for _, where := range []string{"ticket", "evidence", "acceptance"} {
		t.Run(where, func(t *testing.T) {
			p := plan(t)
			target := filepath.Join(p.BundlePath, p.TicketPath)
			switch where {
			case "evidence":
				target = p.EvidencePath
			case "acceptance":
				target = filepath.Join(p.BundlePath, p.AcceptancePath)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				t.Fatal(err)
			}
			changed := []byte("somebody else's retained bytes\n")
			if err := os.WriteFile(target, changed, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Publish(context.Background(), p); !errors.Is(err, ErrConflict) {
				t.Fatalf("want conflict, got %v", err)
			}
			actual, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, changed) {
				t.Fatal("conflicting source was overwritten")
			}
		})
	}
}

func TestConcurrentPublicationHasOneImmutableResult(t *testing.T) {
	p := plan(t)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := Publish(context.Background(), p); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestPriorAcceptanceRemainsHistoryWithoutChangingRequirements(t *testing.T) {
	p := plan(t)
	p.ExpectedTicket = append(p.ExpectedTicket, []byte("- [Earlier decision](old.md)\n\n")...)
	if err := os.WriteFile(filepath.Join(p.BundlePath, p.TicketPath), p.ExpectedTicket, 0600); err != nil {
		t.Fatal(err)
	}
	before, criteria := fingerprints(t, p)
	if _, err := Publish(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	after, afterCriteria := fingerprints(t, p)
	if before != after || criteria != afterCriteria {
		t.Fatal("history changed requirements")
	}
	data, err := os.ReadFile(filepath.Join(p.BundlePath, p.TicketPath))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := recordread.ParseDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	comments := recordread.Sections(doc.Body, "", map[string]string{"Comments": "comments"})
	if len(comments.Sections) != 1 || !strings.Contains(comments.Sections[0].Content, "old.md") {
		t.Fatal("prior acceptance was not retained in history")
	}
}

func TestPublicationRejectsEscapedEvidenceAndWrongIdentity(t *testing.T) {
	p := plan(t)
	p.ProjectID = "wrong"
	if _, err := Publish(context.Background(), p); err == nil {
		t.Fatal("wrong project accepted")
	}
	p = plan(t)
	outside := t.TempDir()
	link := filepath.Join(filepath.Dir(p.BundlePath), "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip(err)
	}
	p.EvidencePath = filepath.Join(link, "evidence.md")
	if _, err := Publish(context.Background(), p); err == nil {
		t.Fatal("symlink escaped the authorized source root")
	}
	if _, err := os.Stat(filepath.Join(outside, "evidence.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("wrote outside authorized roots")
	}
}
