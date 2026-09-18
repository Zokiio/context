package resumption

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/taskcontext"
)

const checkpointID = "281d6632-b93d-43ce-a5b7-796721964024"

func checkpointFixture(t *testing.T) (Request, string, taskcontext.Result) {
	t.Helper()
	records, working := resumeFixture(t, true, "task")
	request := Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"}
	prior, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: records, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(observationsPath(working, "project", "task"), checkpointID)
	publishCheckpoint(t, directory, prior)
	return request, directory, prior
}
func publishCheckpoint(t *testing.T, directory string, prior taskcontext.Result) {
	t.Helper()
	data, err := json.Marshal(prior)
	if err != nil {
		t.Fatal(err)
	}
	publishCheckpointBytes(t, directory, data)
}
func publishCheckpointBytes(t *testing.T, directory string, data []byte) {
	t.Helper()
	metadata := map[string]any{"type": "RecoveryNote", "version": 1, "id": checkpointID, "projectId": "project", "taskId": "task", "actor": "fixture", "observedAt": "2026-09-19T12:00:00Z", "predecessors": []string{}, "ticketPath": "task.md", "checkoutRevision": nil, "contextFile": "context.json", "contextSHA256": fmt.Sprintf("%x", sha256.Sum256(data))}
	header, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	body := "## Approach\nRead the ticket.\n## Completed\nCaptured context.\n## Remaining\nReview.\n## Checks\nReported local check, revision fixture.\n## Questions\nNone\n## Failed approaches\nNone\n## Next step\nCompare.\n"
	writeResumeFile(t, filepath.Join(directory, "note.md"), "---\n"+string(header)+"\n---\n"+body)
	writeResumeFile(t, filepath.Join(directory, "context.json"), string(data))
}
func resumeCheckpoint(t *testing.T, request Request) Result {
	t.Helper()
	result, err := Resume(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestRetainedCheckpointComparisonAndReadOnly(t *testing.T) {
	request, _, _ := checkpointFixture(t)
	before := snapshotTree(t, filepath.Dir(request.RecordsDirectory))
	result := resumeCheckpoint(t, request)
	if !result.Complete || result.Recovery.Status != "available" || result.Recovery.GraphStatus != "valid" || result.Recovery.Observations[0].Snapshot.Status != "valid" || result.Comparison.Candidates[0].Sources[0].Status != "unchanged" {
		t.Fatalf("checkpoint: %+v", result)
	}
	if !reflect.DeepEqual(before, snapshotTree(t, filepath.Dir(request.RecordsDirectory))) {
		t.Fatal("resume changed files")
	}
	writeResumeFile(t, filepath.Join(request.RecordsDirectory, "task.md"), strings.Replace(workItemDocument("task", true), "title: Task", "title: New metadata", 1))
	result = resumeCheckpoint(t, request)
	difference := result.Comparison.Candidates[0].Sources[0]
	if !result.Complete || difference.Status != "changed" || *difference.Previous.Text == *difference.Current.Text {
		t.Fatalf("metadata comparison: %+v", difference)
	}
}
func TestRetainedCheckpointMovedRoot(t *testing.T) {
	request, _, _ := checkpointFixture(t)
	old := filepath.Join(request.RecordsDirectory, "task.md")
	newPath := filepath.Join(request.RecordsDirectory, "renamed.md")
	if err := os.Rename(old, newPath); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(request.RecordsDirectory, "project.md")
	data, _ := os.ReadFile(manifest)
	writeResumeFile(t, manifest, strings.ReplaceAll(string(data), "task.md", "renamed.md"))
	request.TicketPath = "renamed.md"
	result := resumeCheckpoint(t, request)
	sources := result.Comparison.Candidates[0].Sources
	if !result.Complete || len(sources) != 1 || sources[0].Status != "unchanged" || sources[0].Previous.Path != old || sources[0].Current.Path != newPath {
		t.Fatalf("root move: %+v", result)
	}
}
func TestRetainedSnapshotSchemaFailuresKeepGraph(t *testing.T) {
	cases := map[string]func(map[string]any){
		"missing version":  func(m map[string]any) { delete(m, "schemaVersion") },
		"zero version":     func(m map[string]any) { m["schemaVersion"] = 0 },
		"null sources":     func(m map[string]any) { m["sources"] = nil },
		"null diagnostics": func(m map[string]any) { m["diagnostics"] = nil },
		"missing complete": func(m map[string]any) { delete(m, "complete") },
		"wrong traversal":  func(m map[string]any) { m["traversalComplete"] = "true" },
		"null text":        func(m map[string]any) { m["sources"].([]any)[0].(map[string]any)["text"] = nil },
		"bad digest":       func(m map[string]any) { m["sources"].([]any)[0].(map[string]any)["sha256"] = strings.Repeat("0", 64) },
		"no reasons":       func(m map[string]any) { m["sources"].([]any)[0].(map[string]any)["reasons"] = []any{} },
		"null reason kind": func(m map[string]any) {
			m["sources"].([]any)[0].(map[string]any)["reasons"] = []any{map[string]any{"kind": nil}}
		},
		"duplicate root": func(m map[string]any) { s := m["sources"].([]any); m["sources"] = append(s, s[0]) },
		"diagnostic type": func(m map[string]any) {
			m["diagnostics"] = []any{map[string]any{"code": "a", "severity": 1, "message": "b", "path": "/x"}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			request, directory, prior := checkpointFixture(t)
			data, _ := json.Marshal(prior)
			var raw map[string]any
			json.Unmarshal(data, &raw)
			mutate(raw)
			data, _ = json.Marshal(raw)
			publishCheckpointBytes(t, directory, data)
			result := resumeCheckpoint(t, request)
			snapshot := result.Recovery.Observations[0].Snapshot
			if result.Complete || result.Recovery.Status != "available" || result.Recovery.GraphStatus != "valid" || snapshot.Status != "invalid" || snapshot.SourceCount != nil || snapshot.ObservedSHA256 == nil || result.Comparison.BaselineAvailable || len(result.Comparison.Candidates) != 1 || len(result.Comparison.Candidates[0].Sources) != 0 {
				t.Fatalf("invalid schema leaked baseline: %+v", result)
			}
		})
	}
}
func TestRetainedSnapshotMissingDigestAndTrailingJSON(t *testing.T) {
	for _, mode := range []string{"missing", "digest", "trailing"} {
		t.Run(mode, func(t *testing.T) {
			request, directory, prior := checkpointFixture(t)
			path := filepath.Join(directory, "context.json")
			if mode == "missing" {
				os.Remove(path)
			} else if mode == "digest" {
				writeResumeFile(t, path, "changed bytes")
			} else {
				data, _ := json.Marshal(prior)
				publishCheckpointBytes(t, directory, append(data, []byte(" {}")...))
			}
			result := resumeCheckpoint(t, request)
			snapshot := result.Recovery.Observations[0].Snapshot
			want := "invalid"
			if mode == "missing" {
				want = "unavailable"
			}
			if result.Complete || result.Recovery.GraphStatus != "valid" || snapshot.Status != want || len(result.Comparison.Candidates[0].Sources) != 0 {
				t.Fatalf("snapshot failure: %+v", result)
			}
		})
	}
}
func TestRetainedPartialSnapshotKeepsUsableSources(t *testing.T) {
	request, directory, prior := checkpointFixture(t)
	prior.Complete = false
	prior.TraversalComplete = false
	publishCheckpoint(t, directory, prior)
	writeResumeFile(t, filepath.Join(request.RecordsDirectory, "extra.md"), "new document\n")
	writeResumeFile(t, filepath.Join(request.RecordsDirectory, "task.md"), workItemDocument("task", true)+"## Context\n[Extra](extra.md)\n")
	result := resumeCheckpoint(t, request)
	if result.Complete || !result.Comparison.BaselineAvailable || result.Recovery.Observations[0].Snapshot.Status != "incomplete" || result.Comparison.Candidates[0].Sources[1].Status != "unknown" {
		t.Fatalf("partial snapshot: %+v", result)
	}
}
func TestRetainedAuthorizationAndNonRootRename(t *testing.T) {
	for _, mode := range []string{"withheld", "missing", "renamed", "removed"} {
		t.Run(mode, func(t *testing.T) {
			request, directory, prior := checkpointFixture(t)
			docs := canonical(t, t.TempDir())
			oldPath := filepath.Join(docs, "old.md")
			text := "private retained document\n"
			writeResumeFile(t, oldPath, text)
			prior.Sources = append(prior.Sources, taskcontext.Source{Path: oldPath, Text: text, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(text))), Reasons: []taskcontext.Reason{{Kind: "context", From: prior.Sources[0].Path, Link: oldPath}}})
			publishCheckpoint(t, directory, prior)
			if mode != "withheld" {
				request.AllowedSourceDirs = []string{docs}
			}
			if mode == "missing" {
				os.Remove(oldPath)
			}
			if mode == "renamed" {
				newPath := filepath.Join(docs, "new.md")
				os.Rename(oldPath, newPath)
				relative, _ := filepath.Rel(request.RecordsDirectory, newPath)
				writeResumeFile(t, prior.Sources[0].Path, workItemDocument("task", true)+"## Context\n[New]("+relative+")\n")
			}
			result := resumeCheckpoint(t, request)
			sources := result.Comparison.Candidates[0].Sources
			last := sources[len(sources)-1]
			if mode == "removed" {
				if !result.Complete || last.Status != "removed" {
					t.Fatalf("removed: %+v", result)
				}
				return
			}
			if result.Complete || last.Status != "unknown" || last.Previous.Text != nil {
				t.Fatalf("unavailable previous: %+v", result)
			}
			if mode == "withheld" && last.Previous.Availability != "withheld" {
				t.Fatal(last)
			}
			if mode == "renamed" && (len(sources) != 3 || sources[1].Status != "added") {
				t.Fatalf("non-root rename inferred: %+v", sources)
			}
		})
	}
}
func TestCheckpointBudgetExactFitAndLookahead(t *testing.T) {
	request, directory, _ := checkpointFixture(t)
	note, _ := os.ReadFile(filepath.Join(directory, "note.md"))
	snapshot, _ := os.ReadFile(filepath.Join(directory, "context.json"))
	request.MaxCacheFiles = 3
	request.MaxCacheBytes = int64(len(note) + len(snapshot))
	if result := resumeCheckpoint(t, request); !result.Complete {
		t.Fatalf("exact fit: %+v", result)
	}
	request.MaxCacheBytes--
	result := resumeCheckpoint(t, request)
	if result.Complete || result.Recovery.GraphStatus != "valid" || result.Recovery.Observations[0].Snapshot.ObservedSHA256 != nil {
		t.Fatalf("byte breach: %+v", result)
	}
	request.MaxCacheBytes = 0
	request.MaxCacheFiles = 2
	result = resumeCheckpoint(t, request)
	if result.Complete || result.Recovery.Observations[0].Snapshot.Status != "unavailable" {
		t.Fatalf("entry breach: %+v", result)
	}
	reader := &countingCacheReader{remaining: 1000}
	_, err := readCacheBytes(reader, 7)
	if err != errCacheLimit || reader.read != 8 {
		t.Fatalf("read %d bytes, err %v", reader.read, err)
	}
	directoryReader := &countingDirectory{names: []string{"z", "x", "a", "b"}}
	budget := cacheBudget{entries: 2}
	names, complete, err := budget.names(directoryReader)
	if err != nil || complete || directoryReader.request != 3 || !reflect.DeepEqual(names, []string{"x", "z"}) {
		t.Fatalf("bounded observed subset: %v %v %+v", names, err, directoryReader)
	}
}

type countingCacheReader struct{ remaining, read int }

func (r *countingCacheReader) Read(p []byte) (int, error) {
	n := min(len(p), r.remaining)
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	r.remaining -= n
	r.read += n
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

type countingDirectory struct {
	names   []string
	request int
}

func (d *countingDirectory) Readdirnames(n int) ([]string, error) {
	d.request = n
	return d.names[:min(n, len(d.names))], nil
}
func TestCheckpointRejectsAliasesAndUnfinishedPublication(t *testing.T) {
	for _, mode := range []string{"directory inside", "directory outside", "note", "snapshot", "unfinished"} {
		t.Run(mode, func(t *testing.T) {
			request, directory, _ := checkpointFixture(t)
			switch mode {
			case "directory inside", "directory outside":
				target := filepath.Join(filepath.Dir(filepath.Dir(directory)), "quarantine", "retained")
				if mode == "directory outside" {
					target = filepath.Join(t.TempDir(), "retained")
				}
				os.MkdirAll(filepath.Dir(target), 0700)
				if err := os.Rename(directory, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, directory); err != nil {
					t.Fatal(err)
				}
			case "note", "snapshot":
				name := "note.md"
				if mode == "snapshot" {
					name = "context.json"
				}
				old := filepath.Join(directory, name)
				target := filepath.Join(t.TempDir(), name)
				os.Rename(old, target)
				os.Symlink(target, old)
			case "unfinished":
				os.Remove(filepath.Join(directory, "note.md"))
			}
			result := resumeCheckpoint(t, request)
			if result.Complete || result.Comparison.BaselineAvailable {
				t.Fatalf("alias/unfinished accepted: %+v", result)
			}
			if mode == "snapshot" {
				if result.Recovery.Status != "available" {
					t.Fatal(result.Recovery)
				}
			} else if len(result.Recovery.Candidates) != 0 {
				t.Fatal(result.Recovery)
			}
		})
	}
}
func TestCheckpointJSONNullShape(t *testing.T) {
	request, _, _ := checkpointFixture(t)
	result := resumeCheckpoint(t, request)
	data, _ := json.Marshal(result)
	if bytes.Contains(data, []byte(`"predecessors":null`)) || !bytes.Contains(data, []byte(`"checkoutRevision":null`)) || !bytes.Contains(data, []byte(`"observedSHA256":"`)) {
		t.Fatalf("shape: %s", data)
	}
}

func TestCheckpointHistoryIsExplicitlyUnsupported(t *testing.T) {
	for _, mode := range []string{"two roots", "predecessor"} {
		t.Run(mode, func(t *testing.T) {
			request, directory, _ := checkpointFixture(t)
			if mode == "two roots" {
				if err := os.Mkdir(filepath.Join(filepath.Dir(directory), "381d6632-b93d-43ce-a5b7-796721964024"), 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				path := filepath.Join(directory, "note.md")
				data, _ := os.ReadFile(path)
				writeResumeFile(t, path, strings.Replace(string(data), `"predecessors":[]`, `"predecessors":["381d6632-b93d-43ce-a5b7-796721964024"]`, 1))
			}
			if _, err := Resume(context.Background(), request); err == nil || !strings.Contains(err.Error(), "not supported by this implementation slice") {
				t.Fatalf("history: %v", err)
			}
		})
	}
}
func TestCheckpointExtremeLimitsDoNotOverflow(t *testing.T) {
	reader := &countingCacheReader{remaining: 3}
	data, err := readCacheBytes(reader, int64(^uint64(0)>>1))
	if err != nil || len(data) != 3 {
		t.Fatalf("large byte limit: %q %v", data, err)
	}
	directory := &countingDirectory{names: []string{"x"}}
	budget := cacheBudget{entries: int(^uint(0) >> 1)}
	_, complete, err := budget.names(directory)
	if err != nil || !complete || directory.request <= 0 {
		t.Fatalf("large entry limit: %+v %v", directory, err)
	}
}
func TestRetainedSnapshotWrongRootIdentity(t *testing.T) {
	request, directory, prior := checkpointFixture(t)
	prior.Sources[0].Text = workItemDocument("different", true)
	prior.Sources[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(prior.Sources[0].Text)))
	publishCheckpoint(t, directory, prior)
	result := resumeCheckpoint(t, request)
	if result.Recovery.Observations[0].Snapshot.Status != "invalid" || len(result.Comparison.Candidates[0].Sources) != 0 {
		t.Fatalf("wrong identity accepted: %+v", result)
	}
}

func TestRetainedRootAuthorizationDoesNotGuessAnOldCheckout(t *testing.T) {
	root := canonical(t, t.TempDir())
	oldCheckout := filepath.Join(filepath.Dir(root), "missing-old-checkout", "task.md")
	if _, availability := retainedAuthorization(oldCheckout, []string{root}, nil, true); availability != "unavailable" {
		t.Fatalf("missing old checkout: %s", availability)
	}
	alias := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{alias, filepath.Join(alias, "task.md")} {
		if _, availability := retainedAuthorization(path, []string{root}, nil, true); availability != "unavailable" {
			t.Fatalf("dangling alias authorized: %s %s", path, availability)
		}
	}
}

func TestRetainedRootUsesCurrentEffectiveType(t *testing.T) {
	request, directory, _ := checkpointFixture(t)
	writeResumeFile(t, filepath.Join(request.RecordsDirectory, "task.md"), strings.Replace(workItemDocument("task", true), "type: WorkItem", `type: " WorkItem "`, 1))
	retained, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: request.RecordsDirectory, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	publishCheckpoint(t, directory, retained)
	result := resumeCheckpoint(t, request)
	if !result.Complete || result.Recovery.Observations[0].Snapshot.Status != "valid" {
		t.Fatalf("effective root type: %+v", result)
	}
}
