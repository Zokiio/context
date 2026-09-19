package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/cli"
	"github.com/Zokiio/context/internal/resumption"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestResumeRetainedCheckpointCLIContract(t *testing.T) {
	root := scopeTempDir(t)
	home, records, checkout := filepath.Join(root, "home"), filepath.Join(root, "records"), filepath.Join(root, "checkout")
	for _, path := range []string{home, checkout} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeResumeCLIProject(t, records, true)
	retained, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: records, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := json.Marshal(retained)
	if err != nil {
		t.Fatal(err)
	}
	id := "281d6632-b93d-43ce-a5b7-796721964024"
	directory := filepath.Join(checkout, ".context-cache", "resume-v1", shaKey("project"), shaKey("task"), "observations", id)
	note := fmt.Sprintf("---\ntype: RecoveryNote\nversion: 1\nid: %s\nprojectId: project\ntaskId: task\nobservedAt: '2026-09-19T12:00:00Z'\nactor: retained-fixture\npredecessors: []\nticketPath: task.md\ncheckoutRevision: null\ncontextFile: context.json\ncontextSHA256: %x\n---\n## Approach\nRead.\n## Completed\nOne local check.\n## Remaining\nDevice trial.\n## Checks\nReported command: go test. Revision: previous-checkout. Environment: local.\n## Questions\nNone\n## Failed approaches\nNone\n## Next step\nRefresh.\n", id, sha256.Sum256(snapshot))
	writeScopeFile(t, filepath.Join(directory, "note.md"), note)
	writeScopeFile(t, filepath.Join(directory, "context.json"), string(snapshot))
	environment := scopeEnvironment(root, home)
	environment.Input = scopeForbiddenInput{t}
	args := []string{"ctx", "resume", "--bundle", records, "--checkout", checkout, "--ticket", "task.md"}
	var stdout, stderr bytes.Buffer
	status := cli.RunWithEnvironment(context.Background(), args, &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, environment)
	if status != 0 || stderr.Len() != 0 {
		t.Fatalf("text: %d %s %s", status, &stdout, &stderr)
	}
	for _, fragment := range []string{"Historical observation", "completion not established by this note", "previous-checkout", "Current checks:", "Source text comparison", "unchanged"} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("missing %q in %s", fragment, &stdout)
		}
	}
	for _, corrupt := range []bool{false, true} {
		if corrupt {
			writeScopeFile(t, filepath.Join(directory, "context.json"), "bad snapshot")
		}
		stdout.Reset()
		stderr.Reset()
		status = cli.RunWithEnvironment(context.Background(), append(args, "--json"), &stdout, &stderr, cli.Operations{Resume: resumption.Resume}, environment)
		want := 0
		if corrupt {
			want = 1
		}
		if status != want || stderr.Len() != 0 {
			t.Fatalf("JSON: %d %s %s", status, &stdout, &stderr)
		}
		var report map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		recovery := report["recovery"].(map[string]any)
		if recovery["status"] != "available" || recovery["graphStatus"] != "valid" {
			t.Fatal(recovery)
		}
		observation := recovery["observations"].([]any)[0].(map[string]any)
		requireKeys(t, observation, "id", "projectId", "taskId", "observedAt", "actor", "predecessors", "ticketPath", "checkoutRevision", "source", "body", "metadata", "snapshot")
		if observation["checkoutRevision"] != nil || len(observation["predecessors"].([]any)) != 0 {
			t.Fatal(observation)
		}
		reference := observation["source"].(map[string]any)
		requireKeys(t, reference, "path", "sha256")
		snapshotReport := observation["snapshot"].(map[string]any)
		requireKeys(t, snapshotReport, "path", "recordedSHA256", "observedSHA256", "status", "sourceCount")
		candidate := report["comparison"].(map[string]any)["candidates"].([]any)[0].(map[string]any)
		requireKeys(t, candidate, "observationId", "baselineAvailable", "complete", "sources")
		if corrupt {
			if snapshotReport["status"] != "invalid" || snapshotReport["sourceCount"] != nil || candidate["baselineAvailable"] != false || len(candidate["sources"].([]any)) != 0 {
				t.Fatal(candidate, snapshotReport)
			}
			diagnostic := report["diagnostics"].([]any)[0].(map[string]any)
			requireKeys(t, diagnostic, "code", "severity", "message", "path", "from", "link", "observationId")
			if diagnostic["from"] != nil || diagnostic["link"] != nil || diagnostic["observationId"] != id {
				t.Fatal(diagnostic)
			}
		} else {
			difference := candidate["sources"].([]any)[0].(map[string]any)
			requireKeys(t, difference, "status", "previous", "current")
			for _, side := range []string{"previous", "current"} {
				source := difference[side].(map[string]any)
				requireKeys(t, source, "path", "sha256", "text", "availability")
				if source["availability"] != "available" || source["text"] == nil {
					t.Fatal(source)
				}
			}
		}
	}
}
func requireKeys(t *testing.T, object map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, exists := object[key]; !exists {
			t.Fatalf("missing required property %q: %+v", key, object)
		}
	}
}
