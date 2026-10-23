package factoryhttp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Zokiio/context/internal/factory"
)

func TestTaskWorkspaceHTTPDraftLifecycleAndIsolation(t *testing.T) {
	instance := start(t)
	ctx := context.Background()
	if _, err := instance.service.RegisterExecutor(ctx, instance.admin, factory.RegisterExecutorRequest{RequestID: "executor", Key: "fixture-executor", DisplayName: "Fixture", Command: []string{"fixture"}, Fixture: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := instance.service.RegisterCheck(ctx, instance.admin, factory.RegisterCheckRequest{RequestID: "check", Key: "fixture-check", DisplayName: "Fixture check", Command: []string{"fixture"}, Environment: "offline", Fixture: true}); err != nil {
		t.Fatal(err)
	}
	p := project(t, instance, "workspace")
	base := "/v1/projects/" + string(p.ID) + "/work-items"
	status, _, raw := call(t, instance.server.URL, instance.token, "POST", base, `{"requestId":"draft","queue":false,"draft":{"title":"Private draft","description":"## Notes\n\nA **readable** draft.","criteria":["Keep this task out of the queue."]}}`, nil)
	if status != 201 {
		t.Fatalf("create: %d %s", status, raw)
	}
	var record factory.WorkItemRecord
	if err := json.Unmarshal(raw, &record); err != nil || record.Queued || record.Archived || !record.CanDelete {
		t.Fatalf("draft disposition: %+v, %v", record, err)
	}
	itemPath := base + "/" + record.WorkItemID
	_, _, outsider := user(t, instance, "outsider")
	for _, path := range []string{base, itemPath, itemPath + "/activity"} {
		status, _, raw = call(t, instance.server.URL, outsider, "GET", path, "", nil)
		if status != 404 {
			t.Fatalf("isolation %s: %d %s", path, status, raw)
		}
	}
	status, _, raw = call(t, instance.server.URL, outsider, "POST", base+"/reconcile", `{}`, nil)
	if status != 404 {
		t.Fatalf("recovery isolation: %d %s", status, raw)
	}
	status, _, raw = call(t, instance.server.URL, instance.token, "POST", base+"/reconcile", `{}`, nil)
	if status != 200 || string(raw) != "[]\n" {
		t.Fatalf("empty recovery: %d %s", status, raw)
	}
	status, _, raw = call(t, instance.server.URL, instance.token, "GET", itemPath+"/activity", "", nil)
	var activity factory.WorkItemActivity
	if err := json.Unmarshal(raw, &activity); err != nil || status != 200 || len(activity.Jobs) != 0 || activity.WorkItemID != record.WorkItemID {
		t.Fatalf("empty draft activity: %d %s %v", status, raw, err)
	}
	status, _, raw = call(t, instance.server.URL, instance.token, "POST", itemPath+"/disposition", `{"requestId":"mismatch","workItemId":"another-item","action":"archive"}`, nil)
	if status != 400 {
		t.Fatalf("path binding: %d %s", status, raw)
	}
	for _, action := range []string{"queue", "unqueue", "archive", "restore", "delete"} {
		body := fmt.Sprintf(`{"requestId":%q,"ifTicketSHA256":%q,"action":%q}`, action, record.TicketSHA256, action)
		status, _, raw = call(t, instance.server.URL, instance.token, "POST", itemPath+"/disposition", body, nil)
		if status != 200 {
			t.Fatalf("%s: %d %s", action, status, raw)
		}
		if err := json.Unmarshal(raw, &record); err != nil {
			t.Fatal(err)
		}
		if record.Queued != (action == "queue") || record.Archived != (action == "archive") || record.Deleted != (action == "delete") {
			t.Fatalf("%s disposition: %+v", action, record)
		}
		// Retrying a command must return its retained result, including deletion.
		status, _, replay := call(t, instance.server.URL, instance.token, "POST", itemPath+"/disposition", body, nil)
		if status != 200 || string(replay) != string(raw) {
			t.Fatalf("%s replay: %d %s, first %s", action, status, replay, raw)
		}
	}
	status, _, raw = call(t, instance.server.URL, instance.token, "GET", base, "", nil)
	var records []factory.WorkItemRecord
	if err := json.Unmarshal(raw, &records); err != nil || status != 200 || len(records) != 1 || records[0].WorkItemID != "work-1" {
		t.Fatalf("deleted draft still listed: %d %s %v", status, raw, err)
	}
}
