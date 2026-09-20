package resumption

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testObservationID = "281d6632-b93d-43ce-a5b7-796721964024"
	testPredecessorID = "783d15eb-6698-4668-9c13-19806d67d24c"
	testProjectID     = "project-stable-id"
	testTaskID        = "task-stable-id"
	testContextDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func TestParseNoteReturnsExactObservation(t *testing.T) {
	body := "## Approach\nKeep the byte sequence.\r\n\n## Completed\nNone\n## Remaining\n\n## Checks\n`go test ./...`\n## Questions\nNone\n## Failed approaches\nNone\n## Next step\nContinue.\n"
	note := validNote("predecessors: ["+testPredecessorID+"]\ncheckoutRevision: {origin: 'git@example.test:team/project.git', revision: abc123}\ncustom: {limits: [.nan, .inf, -.inf], nested: {1: one, two: 2}}\n", body)
	notePath := filepath.Join("tmp", "observations", testObservationID, "note.md")

	got, err := parseNote(note, notePath, testObservationID, testProjectID, testTaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != testObservationID || got.ProjectID != testProjectID || got.TaskID != testTaskID {
		t.Fatalf("identity changed: %+v", got)
	}
	if got.ObservedAt != "2026-09-17T17:00:00Z" || got.Actor != "coding-session" || got.TicketPath != "feature/issues/task.md" {
		t.Fatalf("typed fields changed: %+v", got)
	}
	if len(got.Predecessors) != 1 || got.Predecessors[0] != testPredecessorID {
		t.Fatalf("predecessors = %#v", got.Predecessors)
	}
	if got.CheckoutRevision == nil || got.CheckoutRevision.Origin != "git@example.test:team/project.git" || got.CheckoutRevision.Revision != "abc123" {
		t.Fatalf("checkout revision = %+v", got.CheckoutRevision)
	}
	if got.Body != body {
		t.Fatalf("body bytes changed:\n got %q\nwant %q", got.Body, body)
	}
	wantSourceDigest := fmt.Sprintf("%x", sha256.Sum256(note))
	if got.Source.Path != notePath || got.Source.SHA256 == nil || *got.Source.SHA256 != wantSourceDigest {
		t.Fatalf("source = %+v, want digest %s", got.Source, wantSourceDigest)
	}
	wantSnapshotPath := filepath.Join(filepath.Dir(notePath), "context.json")
	if got.Snapshot.Path != wantSnapshotPath || got.Snapshot.RecordedSHA256 != testContextDigest || got.Snapshot.Status != "not_loaded" || got.Snapshot.ObservedSHA256 != nil || got.Snapshot.SourceCount != nil {
		t.Fatalf("snapshot = %+v", got.Snapshot)
	}
	encoded, err := json.Marshal(got.Metadata)
	if err != nil {
		t.Fatalf("preserved metadata is not JSON encodable: %v", err)
	}
	for _, want := range []string{`"custom"`, `"yamlType":"float"`, `"1":"one"`, `"two":2`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("metadata lost %s: %s", want, encoded)
		}
	}
}

func TestParseNoteAcceptsTrimmedEffectiveIdentitiesAndNullRevision(t *testing.T) {
	note := validNote("projectId: '  project-stable-id  '\ntaskId: ' task-stable-id '\nactor: ' coding-session '\nticketPath: ' feature/issues/task.md '\ncheckoutRevision: null\n", validNoteBody())
	note = replaceNoteText(t, note, "id: "+testObservationID, "id: '  "+testObservationID+"  '")
	got, err := parseNote(note, "/cache/note.md", testObservationID, testProjectID, testTaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectID != testProjectID || got.TaskID != testTaskID || got.Actor != "coding-session" || got.TicketPath != "feature/issues/task.md" || got.CheckoutRevision != nil {
		t.Fatalf("effective fields = %+v", got)
	}
	if got.Predecessors == nil || len(got.Predecessors) != 0 {
		t.Fatalf("empty predecessor array must remain non-nil: %#v", got.Predecessors)
	}
}

func TestParseNoteRejectsMissingNullAndWrongTypedFields(t *testing.T) {
	tests := []struct {
		name    string
		oldText string
		newText string
	}{
		{"missing type", "type: RecoveryNote\n", ""},
		{"null type", "type: RecoveryNote", "type: null"},
		{"wrong type", "type: RecoveryNote", "type: recovery-note"},
		{"missing version", "version: 1\n", ""},
		{"null version", "version: 1", "version: null"},
		{"string version", "version: 1", "version: '1'"},
		{"float version", "version: 1", "version: 1.0"},
		{"unsupported version", "version: 1", "version: 2"},
		{"missing id", "id: " + testObservationID + "\n", ""},
		{"null id", "id: " + testObservationID, "id: null"},
		{"numeric id", "id: " + testObservationID, "id: 7"},
		{"invalid id", "id: " + testObservationID, "id: 281D6632-b93d-43ce-a5b7-796721964024"},
		{"missing project", "projectId: " + testProjectID + "\n", ""},
		{"null project", "projectId: " + testProjectID, "projectId: null"},
		{"numeric project", "projectId: " + testProjectID, "projectId: 7"},
		{"empty project", "projectId: " + testProjectID, "projectId: '  '"},
		{"missing task", "taskId: " + testTaskID + "\n", ""},
		{"null task", "taskId: " + testTaskID, "taskId: null"},
		{"numeric task", "taskId: " + testTaskID, "taskId: 7"},
		{"missing observed at", "observedAt: '2026-09-17T17:00:00Z'\n", ""},
		{"null observed at", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: null"},
		{"numeric observed at", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: 7"},
		{"invalid observed at", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: yesterday"},
		{"comma fractional seconds", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: '2026-09-17T17:00:00,5Z'"},
		{"out of range positive offset minute", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: '2026-09-17T17:00:00+00:60'"},
		{"out of range negative offset minute", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: '2026-09-17T17:00:00-23:60'"},
		{"out of range offset", "observedAt: '2026-09-17T17:00:00Z'", "observedAt: '2026-09-17T17:00:00+24:00'"},
		{"missing actor", "actor: coding-session\n", ""},
		{"null actor", "actor: coding-session", "actor: null"},
		{"numeric actor", "actor: coding-session", "actor: 7"},
		{"empty actor", "actor: coding-session", "actor: '  '"},
		{"missing predecessors", "predecessors: []\n", ""},
		{"null predecessors", "predecessors: []", "predecessors: null"},
		{"object predecessors", "predecessors: []", "predecessors: {}"},
		{"non-string predecessor", "predecessors: []", "predecessors: [7]"},
		{"invalid predecessor UUID", "predecessors: []", "predecessors: [281D6632-b93d-43ce-a5b7-796721964024]"},
		{"self predecessor", "predecessors: []", "predecessors: [" + testObservationID + "]"},
		{"duplicate predecessor", "predecessors: []", "predecessors: [" + testPredecessorID + ", " + testPredecessorID + "]"},
		{"missing ticket", "ticketPath: feature/issues/task.md\n", ""},
		{"null ticket", "ticketPath: feature/issues/task.md", "ticketPath: null"},
		{"numeric ticket", "ticketPath: feature/issues/task.md", "ticketPath: 7"},
		{"empty ticket", "ticketPath: feature/issues/task.md", "ticketPath: '  '"},
		{"missing checkout revision", "checkoutRevision: null\n", ""},
		{"scalar checkout revision", "checkoutRevision: null", "checkoutRevision: revision"},
		{"missing revision origin", "checkoutRevision: null", "checkoutRevision: {revision: abc123}"},
		{"null revision origin", "checkoutRevision: null", "checkoutRevision: {origin: null, revision: abc123}"},
		{"empty revision origin", "checkoutRevision: null", "checkoutRevision: {origin: ' ', revision: abc123}"},
		{"missing revision", "checkoutRevision: null", "checkoutRevision: {origin: repository}"},
		{"null revision", "checkoutRevision: null", "checkoutRevision: {origin: repository, revision: null}"},
		{"empty revision", "checkoutRevision: null", "checkoutRevision: {origin: repository, revision: ' '}"},
		{"numeric revision", "checkoutRevision: null", "checkoutRevision: {origin: repository, revision: 123}"},
		{"missing context file", "contextFile: context.json\n", ""},
		{"null context file", "contextFile: context.json", "contextFile: null"},
		{"numeric context file", "contextFile: context.json", "contextFile: 7"},
		{"wrong context file", "contextFile: context.json", "contextFile: snapshot.json"},
		{"missing context digest", "contextSHA256: " + testContextDigest + "\n", ""},
		{"null context digest", "contextSHA256: " + testContextDigest, "contextSHA256: null"},
		{"numeric context digest", "contextSHA256: " + testContextDigest, "contextSHA256: 7"},
		{"short context digest", "contextSHA256: " + testContextDigest, "contextSHA256: abcdef"},
		{"uppercase context digest", "contextSHA256: " + testContextDigest, "contextSHA256: " + strings.ToUpper(testContextDigest)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			note := validNote("checkoutRevision: null\n", validNoteBody())
			note = replaceNoteText(t, note, test.oldText, test.newText)
			_, err := parseNote(note, "/cache/note.md", testObservationID, testProjectID, testTaskID)
			if err == nil {
				t.Fatal("expected malformed note error")
			}
			if errors.Is(err, errNoteIdentity) {
				t.Fatalf("malformed field classified as identity mismatch: %v", err)
			}
		})
	}
}

func TestParseNoteClassifiesIdentityMismatches(t *testing.T) {
	tests := []struct {
		name, directoryID, projectID, taskID string
	}{
		{"directory", testPredecessorID, testProjectID, testTaskID},
		{"project", testObservationID, "another-project", testTaskID},
		{"task", testObservationID, testProjectID, "another-task"},
	}
	note := validNote("checkoutRevision: null\n", validNoteBody())
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseNote(note, "/cache/note.md", test.directoryID, test.projectID, test.taskID)
			if !errors.Is(err, errNoteIdentity) {
				t.Fatalf("error = %v; want errNoteIdentity", err)
			}
		})
	}
}

func TestParseNoteRequiresEachSectionExactlyOnce(t *testing.T) {
	for _, section := range requiredNoteSections {
		t.Run("missing "+section, func(t *testing.T) {
			body := strings.Replace(validNoteBody(), "## "+section+"\nNone\n", "", 1)
			_, err := parseNote(validNote("checkoutRevision: null\n", body), "/cache/note.md", testObservationID, testProjectID, testTaskID)
			if err == nil || !strings.Contains(err.Error(), section) {
				t.Fatalf("error = %v", err)
			}
		})
		t.Run("duplicate "+section, func(t *testing.T) {
			body := validNoteBody() + "## " + section + "\nSecond copy.\n"
			_, err := parseNote(validNote("checkoutRevision: null\n", body), "/cache/note.md", testObservationID, testProjectID, testTaskID)
			if err == nil || !strings.Contains(err.Error(), section) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestParseNoteRejectsInvalidUTF8AndNestedDuplicateKeys(t *testing.T) {
	invalidUTF8 := append(validNote("checkoutRevision: null\n", validNoteBody()), 0xff)
	if _, err := parseNote(invalidUTF8, "/cache/note.md", testObservationID, testProjectID, testTaskID); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}

	duplicate := validNote("checkoutRevision: null\ncustom: {nested: {same: first, same: second}}\n", validNoteBody())
	if _, err := parseNote(duplicate, "/cache/note.md", testObservationID, testProjectID, testTaskID); err == nil || !strings.Contains(err.Error(), "already defined") {
		t.Fatalf("nested duplicate error = %v", err)
	}
}

func validNote(extra, body string) []byte {
	fields := map[string]string{
		"projectId":        "projectId: " + testProjectID + "\n",
		"taskId":           "taskId: " + testTaskID + "\n",
		"actor":            "actor: coding-session\n",
		"ticketPath":       "ticketPath: feature/issues/task.md\n",
		"predecessors":     "predecessors: []\n",
		"checkoutRevision": "",
	}
	for _, line := range strings.SplitAfter(extra, "\n") {
		name, _, found := strings.Cut(line, ":")
		if found {
			if _, known := fields[name]; known {
				fields[name] = ""
			}
		}
	}
	return []byte("---\n" +
		"type: RecoveryNote\n" +
		"version: 1\n" +
		"id: " + testObservationID + "\n" +
		fields["projectId"] + fields["taskId"] +
		"observedAt: '2026-09-17T17:00:00Z'\n" +
		fields["actor"] + fields["predecessors"] + fields["ticketPath"] +
		fields["checkoutRevision"] +
		extra +
		"contextFile: context.json\n" +
		"contextSHA256: " + testContextDigest + "\n" +
		"---\n" + body)
}

func validNoteBody() string {
	var body strings.Builder
	for _, section := range requiredNoteSections {
		body.WriteString("## ")
		body.WriteString(section)
		body.WriteString("\nNone\n")
	}
	return body.String()
}

func replaceNoteText(t *testing.T, note []byte, oldText, newText string) []byte {
	t.Helper()
	if strings.Count(string(note), oldText) != 1 {
		t.Fatalf("fixture replacement %q occurs %d times", oldText, strings.Count(string(note), oldText))
	}
	return []byte(strings.Replace(string(note), oldText, newText, 1))
}
