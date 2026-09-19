package resumption

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/taskcontext"
)

func graphID(index int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", index) }
func graphFixture(t *testing.T) (Request, taskcontext.Result) {
	t.Helper()
	request, directory, retained := checkpointFixture(t)
	if err := os.RemoveAll(directory); err != nil {
		t.Fatal(err)
	}
	return request, retained
}
func graphCheckpoint(t *testing.T, request Request, retained taskcontext.Result, id int, predecessors ...int) string {
	t.Helper()
	directory := filepath.Join(observationsPath(request.WorkingDirectory, "project", "task"), graphID(id))
	publishCheckpoint(t, directory, retained)
	path := filepath.Join(directory, "note.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(predecessors))
	for _, predecessor := range predecessors {
		ids = append(ids, graphID(predecessor))
	}
	encoded, _ := json.Marshal(ids)
	note := strings.Replace(string(data), checkpointID, graphID(id), 1)
	note = strings.Replace(note, `"predecessors":[]`, `"predecessors":`+string(encoded), 1)
	writeResumeFile(t, path, note)
	return directory
}
func TestRecoveryGraphHistories(t *testing.T) {
	cases := []struct {
		name       string
		edges      [][]int
		candidates []int
	}{
		{"linear", [][]int{{}, {1}, {2}}, []int{3}},
		{"multiple roots", [][]int{{}, {}}, []int{1, 2}},
		{"competing successors", [][]int{{}, {1}, {1}}, []int{2, 3}},
		{"reconciled successors", [][]int{{}, {1}, {1}, {2, 3}}, []int{4}},
		{"partial reconciliation", [][]int{{}, {1}, {1}, {2}}, []int{3, 4}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request, retained := graphFixture(t)
			// Reverse publication and timestamps to make both irrelevant to selection.
			for i := len(test.edges) - 1; i >= 0; i-- {
				directory := graphCheckpoint(t, request, retained, i+1, test.edges[i]...)
				path := filepath.Join(directory, "note.md")
				data, _ := os.ReadFile(path)
				writeResumeFile(t, path, strings.Replace(string(data), "2026-09-19T12:00:00Z", fmt.Sprintf("2026-09-%02dT12:00:00Z", 20-i), 1))
			}
			before := snapshotTree(t, filepath.Dir(request.RecordsDirectory))
			result := resumeCheckpoint(t, request)
			want := []string{}
			for _, id := range test.candidates {
				want = append(want, graphID(id))
			}
			if !result.Complete || result.Recovery.GraphStatus != "valid" || !reflect.DeepEqual(result.Recovery.Candidates, want) || len(result.Comparison.Candidates) != len(want) {
				t.Fatalf("graph result: %+v", result)
			}
			wantStatus := "available"
			if len(want) > 1 {
				wantStatus = "conflicting"
			}
			if result.Recovery.Status != wantStatus {
				t.Fatal(result.Recovery)
			}
			for index, observation := range result.Recovery.Observations {
				if observation.ID != graphID(index+1) {
					t.Fatalf("observation order: %+v", result.Recovery.Observations)
				}
				selected := false
				for _, id := range want {
					selected = selected || observation.ID == id
				}
				expected := "not_loaded"
				if selected {
					expected = "valid"
				}
				if observation.Snapshot.Status != expected {
					t.Fatal(observation.Snapshot)
				}
				if !selected && (observation.Snapshot.ObservedSHA256 != nil || observation.Snapshot.SourceCount != nil) {
					t.Fatal(observation.Snapshot)
				}
			}
			for index, candidate := range result.Comparison.Candidates {
				if candidate.ObservationID != want[index] || !candidate.Complete || !candidate.BaselineAvailable {
					t.Fatal(candidate)
				}
			}
			if !reflect.DeepEqual(before, snapshotTree(t, filepath.Dir(request.RecordsDirectory))) {
				t.Fatal("resume mutated graph")
			}
		})
	}
}
func TestRecoveryGraphInvalidityAndPartialInspection(t *testing.T) {
	for _, mode := range []string{"dangling", "cycle", "self", "duplicate predecessor", "duplicate identity", "malformed", "unfinished", "invalid and unfinished", "cycle and unfinished"} {
		t.Run(mode, func(t *testing.T) {
			request, retained := graphFixture(t)
			first := graphCheckpoint(t, request, retained, 1)
			second := graphCheckpoint(t, request, retained, 2, 1)
			wantGraph, wantCode := "invalid", "recovery_invalid_note"
			switch mode {
			case "dangling":
				graphCheckpoint(t, request, retained, 2, 9)
				wantCode = "recovery_predecessor_missing"
			case "cycle", "cycle and unfinished":
				graphCheckpoint(t, request, retained, 1, 2)
				wantCode = "recovery_cycle"
			case "self":
				graphCheckpoint(t, request, retained, 2, 2)
			case "duplicate predecessor":
				graphCheckpoint(t, request, retained, 2, 1, 1)
			case "duplicate identity":
				path := filepath.Join(second, "note.md")
				data, _ := os.ReadFile(path)
				writeResumeFile(t, path, strings.Replace(string(data), `"id":"`+graphID(2)+`"`, `"id":"`+graphID(1)+`"`, 1))
				wantCode = "recovery_identity_mismatch"
			case "malformed", "invalid and unfinished":
				writeResumeFile(t, filepath.Join(first, "note.md"), "not a recovery note")
			case "unfinished":
				os.Remove(filepath.Join(second, "note.md"))
				wantGraph = "incomplete"
				wantCode = "recovery_incomplete_observation"
			}
			if strings.Contains(mode, "and unfinished") {
				os.Mkdir(filepath.Join(filepath.Dir(first), graphID(3)), 0700)
			}
			result := resumeCheckpoint(t, request)
			if result.Complete || result.Recovery.GraphStatus != wantGraph || result.Recovery.Status != "unknown" || len(result.Recovery.Candidates) != 0 || len(result.Comparison.Candidates) != 0 {
				t.Fatalf("invalid graph: %+v", result)
			}
			found := false
			for _, diagnostic := range result.Diagnostics {
				found = found || diagnostic.Code == wantCode
			}
			if !found {
				t.Fatalf("missing %s: %+v", wantCode, result.Diagnostics)
			}
			for _, observation := range result.Recovery.Observations {
				if observation.Snapshot.Status != "not_loaded" {
					t.Fatal(observation)
				}
			}
			if mode == "dangling" && (len(result.Recovery.Observations) != 2 || result.Recovery.Observations[1].Predecessors[0] != graphID(9)) {
				t.Fatal("dangling metadata was lost")
			}
			if strings.Contains(mode, "unfinished") && result.Recovery.InventoryComplete {
				t.Fatal("unfinished inventory marked complete")
			}
		})
	}
}
func TestRecoveryLoadsOnlyLeavesAndAggregatesSnapshots(t *testing.T) {
	request, retained := graphFixture(t)
	root := graphCheckpoint(t, request, retained, 1)
	left := graphCheckpoint(t, request, retained, 2, 1)
	graphCheckpoint(t, request, retained, 3, 1)
	os.Remove(filepath.Join(root, "context.json"))
	result := resumeCheckpoint(t, request)
	if !result.Complete || result.Recovery.Observations[0].Snapshot.Status != "not_loaded" {
		t.Fatal(result)
	}
	writeResumeFile(t, filepath.Join(left, "context.json"), "corrupt snapshot")
	result = resumeCheckpoint(t, request)
	if result.Complete || !result.Comparison.BaselineAvailable || result.Recovery.Status != "conflicting" || result.Recovery.GraphStatus != "valid" || len(result.Comparison.Candidates) != 2 {
		t.Fatal(result)
	}
	if len(result.Comparison.Candidates[0].Sources) != 0 || result.Comparison.Candidates[0].Complete || !result.Comparison.Candidates[1].Complete {
		t.Fatal(result.Comparison)
	}
}
func TestRecoveryBudgetsStopAfterFirstBreach(t *testing.T) {
	request, retained := graphFixture(t)
	directories := []string{}
	for i := 1; i <= 3; i++ {
		directories = append(directories, graphCheckpoint(t, request, retained, i))
	}
	// All metadata fits. Only one candidate snapshot fits the byte budget.
	var noteBytes int64
	for _, directory := range directories {
		info, _ := os.Stat(filepath.Join(directory, "note.md"))
		noteBytes += info.Size()
	}
	snapshot, _ := os.ReadFile(filepath.Join(directories[0], "context.json"))
	request.MaxCacheBytes = noteBytes + int64(len(snapshot))
	result := resumeCheckpoint(t, request)
	if result.Complete || result.Recovery.Status != "conflicting" || !result.Recovery.InventoryComplete || len(result.Comparison.Candidates) != 3 {
		t.Fatal(result)
	}
	if result.Recovery.Observations[0].Snapshot.Status != "valid" || result.Recovery.Observations[1].Snapshot.Status != "unavailable" || result.Recovery.Observations[2].Snapshot.Status != "unavailable" {
		t.Fatal(result.Recovery)
	}
	foundLimit, foundPending := false, false
	for _, d := range result.Diagnostics {
		if d.ObservationID != nil && *d.ObservationID == graphID(2) && d.Code == "recovery_limit_exceeded" {
			foundLimit = true
		}
		if d.ObservationID != nil && *d.ObservationID == graphID(3) && d.Code == "recovery_source_omitted" {
			foundPending = true
		}
	}
	if !foundLimit || !foundPending {
		t.Fatal(result.Diagnostics)
	}
	request.MaxCacheBytes = 0
	request.MaxCacheFiles = 4 // three directories and one note
	result = resumeCheckpoint(t, request)
	if result.Recovery.InventoryComplete || len(result.Recovery.Observations) != 1 || len(result.Recovery.Candidates) != 0 || result.Recovery.Observations[0].Snapshot.Status != "not_loaded" {
		t.Fatal(result)
	}
	// An unread existing predecessor cannot be diagnosed as a proven dangling edge.
	graphCheckpoint(t, request, retained, 1, 3)
	result = resumeCheckpoint(t, request)
	if result.Recovery.GraphStatus != "incomplete" {
		t.Fatal(result.Recovery)
	}
}
func TestRecoveryConcurrentPublicationAndExplicitQuarantine(t *testing.T) {
	request, retained := graphFixture(t)
	root := graphCheckpoint(t, request, retained, 1)
	type pending struct {
		directory      string
		note, snapshot []byte
	}
	writers := []pending{}
	for _, id := range []int{2, 3} {
		directory := graphCheckpoint(t, request, retained, id, 1)
		note, _ := os.ReadFile(filepath.Join(directory, "note.md"))
		snapshot, _ := os.ReadFile(filepath.Join(directory, "context.json"))
		os.RemoveAll(directory)
		writers = append(writers, pending{directory, note, snapshot})
	}
	results := make(chan error, len(writers))
	start := make(chan struct{})
	for _, writer := range writers {
		go func() {
			<-start
			if err := os.Mkdir(writer.directory, 0700); err != nil {
				results <- err
				return
			}
			if err := os.WriteFile(filepath.Join(writer.directory, "context.json"), writer.snapshot, 0600); err != nil {
				results <- err
				return
			}
			temporary := filepath.Join(writer.directory, "note.tmp")
			if err := os.WriteFile(temporary, writer.note, 0600); err != nil {
				results <- err
				return
			}
			results <- os.Rename(temporary, filepath.Join(writer.directory, "note.md"))
		}()
	}
	close(start)
	for range writers {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	result := resumeCheckpoint(t, request)
	if !result.Complete || !reflect.DeepEqual(result.Recovery.Candidates, []string{graphID(2), graphID(3)}) {
		t.Fatal(result)
	}
	unfinished := filepath.Join(filepath.Dir(root), graphID(4))
	os.Mkdir(unfinished, 0700)
	result = resumeCheckpoint(t, request)
	if result.Recovery.GraphStatus != "incomplete" || len(result.Recovery.Candidates) != 0 {
		t.Fatal(result)
	}
	quarantine := filepath.Join(filepath.Dir(filepath.Dir(root)), "quarantine", graphID(5))
	os.MkdirAll(filepath.Dir(quarantine), 0700)
	if err := os.Rename(unfinished, quarantine); err != nil {
		t.Fatal(err)
	}
	// Quarantine and arbitrary inner entries do not consume the active budget.
	for i := 0; i < 30; i++ {
		writeResumeFile(t, filepath.Join(quarantine, fmt.Sprint(i)), "unread")
		writeResumeFile(t, filepath.Join(root, fmt.Sprint(i)), "unread")
	}
	request.MaxCacheFiles = 8 // three directories, three notes, two snapshots
	result = resumeCheckpoint(t, request)
	if !result.Complete || result.Recovery.Status != "conflicting" {
		t.Fatal(result)
	}
	request.MaxCacheFiles = 0
	graphCheckpoint(t, request, retained, 6, 2, 3)
	result = resumeCheckpoint(t, request)
	if !result.Complete || !reflect.DeepEqual(result.Recovery.Candidates, []string{graphID(6)}) {
		t.Fatal(result)
	}
}
func TestRecoveryDiagnosticIdentityAndOrder(t *testing.T) {
	path, id := "/note.md", graphID(1)
	first := Diagnostic{Code: "recovery_cycle", Severity: "error", Message: "first", Path: &path, ObservationID: &id}
	second := first
	second.Message = "different words"
	other := first
	other.ObservationID = optionalString(graphID(2))
	got := appendUniqueDiagnostics([]Diagnostic{first}, []Diagnostic{second, other})
	if len(got) != 2 || got[0].Message != "first" {
		t.Fatal(got)
	}
	request, retained := graphFixture(t)
	graphCheckpoint(t, request, retained, 1, 9)
	graphCheckpoint(t, request, retained, 2, 8)
	result := resumeCheckpoint(t, request)
	for index, d := range result.Diagnostics {
		if index > 0 {
			previous := result.Diagnostics[index-1]
			if *previous.Path > *d.Path || (*previous.Path == *d.Path && previous.Code > d.Code) {
				t.Fatal(result.Diagnostics)
			}
		}
		if d.ObservationID == nil || d.Link == nil || d.From != nil {
			t.Fatal(d)
		}
	}
}
