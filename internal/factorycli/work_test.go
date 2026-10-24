package factorycli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func TestWorkDraftIsExplicitAndLegacyCreationStillQueues(t *testing.T) {
	for _, draft := range []bool{false, true} {
		var got factory.CreateWorkItemRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/v1/projects/p/work-items" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"workItemId":"work"}`)
		}))
		args := []string{"--url", server.URL, "--token-file", tokenFile(t), "work", "add", "--project", "p", "--title", "Draft", "--criterion", "One check", "--spec", "spec.md", "--context", "context.md"}
		if draft {
			args = append(args, "--draft")
		}
		_, err := run(t, args...)
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		if draft && (got.Queue == nil || *got.Queue) {
			t.Fatal("draft entered the queue")
		}
		if !draft && got.Queue != nil {
			t.Fatal("legacy creation changed its default")
		}
		if len(got.Draft.Spec) != 1 || len(got.Draft.Context) != 1 {
			t.Fatalf("links lost: %+v", got.Draft)
		}
	}
}

func TestWorkDispositionCarriesTheExpectedRevision(t *testing.T) {
	var got factory.SetWorkItemDispositionRequest
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v1/projects/p/work-items/w/disposition" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		io.WriteString(w, `{"code":"stale","message":"the task changed"}`)
	}))
	defer server.Close()
	_, err := run(t, "--url", server.URL, "--token-file", tokenFile(t), "work", "archive", "--project", "p", "--work-item", "w", "--if-ticket-sha256", "opened-revision", "--request-id", "archive-once")
	if err == nil || requests != 1 || got.Action != "archive" || got.IfTicketSHA256 != "opened-revision" || got.RequestID != "archive-once" {
		t.Fatalf("unsafe disposition: %+v, %v", got, err)
	}
}

func TestWorkEditCanClearDocumentLinks(t *testing.T) {
	var got factory.UpdateWorkItemRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			io.WriteString(w, `{"projectId":"p","workItemId":"w","ticketSHA256":"opened-revision","draft":{"title":"Task","criteria":["Check"],"spec":["spec.md"],"context":["context.md"]}}`)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		io.WriteString(w, `{}`)
	}))
	defer server.Close()
	_, err := run(t, "--url", server.URL, "--token-file", tokenFile(t), "work", "edit", "--project", "p", "--work-item", "w", "--spec", "", "--context", "")
	if err != nil || got.Draft.Spec == nil || len(got.Draft.Spec) != 0 || got.Draft.Context == nil || len(got.Draft.Context) != 0 || got.IfTicketSHA256 != "opened-revision" {
		t.Fatalf("explicit link clearing lost: %+v, %v", got, err)
	}
}
