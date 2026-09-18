package resumption

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/taskcontext"
)

func TestResumeWithoutObservationStorage(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	before := snapshotTree(t, filepath.Dir(records))
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Complete || result.SchemaVersion != 1 || result.Kind != "task-resumption" {
		t.Fatalf("result: %+v", result)
	}
	if result.Scope.ProjectID == nil || *result.Scope.ProjectID != "project" || result.Scope.TaskID == nil || *result.Scope.TaskID != "task" {
		t.Fatalf("scope identity: %+v", result.Scope)
	}
	if result.Scope.RecordsDirectory != records || result.Scope.WorkingDirectory != working || result.Scope.CacheRoot != filepath.Join(working, ".context-cache", "resume-v1") {
		t.Fatalf("scope paths: %+v", result.Scope)
	}
	if result.Recovery.Status != "absent" || !result.Recovery.InventoryComplete || result.Recovery.GraphStatus != "valid" || len(result.Recovery.Observations) != 0 || len(result.Recovery.Candidates) != 0 {
		t.Fatalf("recovery: %+v", result.Recovery)
	}
	if result.Comparison.BaselineAvailable || !result.Comparison.Complete || len(result.Comparison.Candidates) != 0 {
		t.Fatalf("comparison: %+v", result.Comparison)
	}
	if result.Context.Sources == nil || result.Orientation.Sources == nil || result.Diagnostics == nil {
		t.Fatal("required arrays must be present")
	}
	if got := snapshotTree(t, filepath.Dir(records)); !reflect.DeepEqual(before, got) {
		t.Fatalf("resume wrote files: before=%v after=%v", before, got)
	}
}

func TestResumeTreatsEmptyObservationDirectoryAsAbsent(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	observations := observationsPath(working, "project", "task")
	if err := os.MkdirAll(observations, 0700); err != nil {
		t.Fatal(err)
	}
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxCacheFiles: 1, MaxCacheBytes: 1})
	if err != nil || !result.Complete || result.Recovery.Status != "absent" {
		t.Fatalf("empty store: %+v, %v", result, err)
	}
}

func TestResumeRejectsNonemptyObservationStoreWithoutScanningIt(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	observations := observationsPath(working, "project", "task")
	for index := 0; index < 1000; index++ {
		path := filepath.Join(observations, namespaceName(index))
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxCacheFiles: 1, MaxCacheBytes: 1})
	if err == nil || !contains(err.Error(), "not supported by this implementation slice") {
		t.Fatalf("nonempty store: %v", err)
	}
}

func TestResumeSkipsCacheWhenIdentityIsUnavailable(t *testing.T) {
	records, working := resumeFixture(t, false, "")
	path := filepath.Join(working, ".context-cache", "resume-v1", "must-not-be-inspected")
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "entry"), []byte("ignored"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.Scope.ProjectID == nil || result.Scope.TaskID != nil || result.Recovery.Status != "unknown" || result.Recovery.InventoryComplete || result.Recovery.GraphStatus != "not_evaluated" || result.Comparison.Complete {
		t.Fatalf("unknown identity: %+v", result)
	}
}

func TestResumeKeepsOrientationWhenTicketEscapesRecordScope(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	outside := filepath.Join(filepath.Dir(records), "outside.md")
	writeResumeFile(t, outside, workItemDocument("outside", true))
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "../outside.md"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || !result.Orientation.Complete || result.Context.Complete || result.Scope.TaskID != nil || result.Recovery.Status != "unknown" {
		t.Fatalf("escaped ticket result: %+v", result)
	}
	if !hasContextDiagnostic(result.Context.Diagnostics, "source_outside_scope", outside) {
		t.Fatalf("escaped ticket diagnostics: %+v", result.Context.Diagnostics)
	}
}

func TestResumeUsesOnlySelectedIdentityNamespace(t *testing.T) {
	records, working := resumeFixture(t, true, "selected")
	writeResumeFile(t, filepath.Join(records, "other.md"), workItemDocument("other", true))
	other := observationsPath(working, "project", "other")
	if err := os.MkdirAll(filepath.Join(other, "entry"), 0700); err != nil {
		t.Fatal(err)
	}
	otherProject := observationsPath(working, "other-project", "selected")
	if err := os.MkdirAll(filepath.Join(otherProject, "entry"), 0700); err != nil {
		t.Fatal(err)
	}
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil || !result.Complete || result.Recovery.Status != "absent" || result.Scope.TaskID == nil || *result.Scope.TaskID != "selected" {
		t.Fatalf("namespace selection: %+v, %v", result, err)
	}
}

func TestResumeReportsCachePathAliasesAsPartial(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(working, ".context-cache")); err != nil {
		t.Fatal(err)
	}
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil || result.Complete || result.Recovery.InventoryComplete || result.Recovery.Status != "unknown" || result.Recovery.GraphStatus != "incomplete" || len(result.Diagnostics) != 1 {
		t.Fatalf("cache alias: %+v, %v", result, err)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "recovery_source_omitted" || diagnostic.Path == nil || *diagnostic.Path != filepath.Join(working, ".context-cache") || !contains(diagnostic.Message, "must not be a symbolic link") {
		t.Fatalf("cache alias diagnostic: %+v", diagnostic)
	}
}

func TestResumeReportsMalformedCacheStructureAsPartial(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	cache := filepath.Join(working, ".context-cache")
	if err := os.WriteFile(cache, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil || result.Complete || !result.Context.Complete || !result.Orientation.Complete || result.Recovery.InventoryComplete || result.Recovery.GraphStatus != "incomplete" || result.Comparison.Complete {
		t.Fatalf("malformed cache: %+v, %v", result, err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "recovery_source_omitted" || !contains(result.Diagnostics[0].Message, "must be a directory") {
		t.Fatalf("malformed cache diagnostic: %+v", result.Diagnostics)
	}
}

func TestResumeSharedCurrentSourceLimit(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	complete, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxFiles: 2})
	if err != nil || !complete.Complete {
		t.Fatalf("manifest and task should fit shared exact limit: %+v, %v", complete, err)
	}
	partial, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxFiles: 1})
	if err != nil || partial.Complete || partial.Context.Complete || partial.Scope.TaskID != nil || partial.Recovery.Status != "unknown" {
		t.Fatalf("shared limit partial result: %+v, %v", partial, err)
	}
}

func TestResumeAttributesManifestFirstByteLimit(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	manifestInfo, err := os.Stat(filepath.Join(records, "project.md"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxBytes: manifestInfo.Size() - 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || result.Context.Complete || result.Orientation.Complete || result.Context.TraversalComplete {
		t.Fatalf("manifest breach reported complete: %+v", result)
	}
	manifest, task := filepath.Join(records, "project.md"), filepath.Join(records, "task.md")
	if !hasResumeDiagnostic(result.Orientation.Diagnostics, "source_limit_exceeded", manifest) || !hasContextDiagnostic(result.Context.Diagnostics, "source_omitted", task) {
		t.Fatalf("manifest breach attribution: context=%+v orientation=%+v", result.Context.Diagnostics, result.Orientation.Diagnostics)
	}
}

func TestResumeIncompleteInventoryCannotEstablishIdentityUniqueness(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	writeResumeFile(t, filepath.Join(records, "unread.md"), "---\ntype: Other\nid: other\ntitle: Other\n---\n")
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxFiles: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.Orientation.InventoryComplete || result.Scope.ProjectID != nil || result.Scope.TaskID != nil || result.Recovery.Status != "unknown" {
		t.Fatalf("incomplete inventory established identity: %+v", result)
	}
}

func TestResumeDuplicateProjectIdentityDisablesCacheLookup(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	writeResumeFile(t, filepath.Join(records, "duplicate.md"), workItemDocument("project", true))
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Scope.ProjectID != nil || result.Scope.TaskID == nil || result.Recovery.Status != "unknown" || result.Complete {
		t.Fatalf("duplicate project identity: %+v", result)
	}
}

func TestResumeJSONHasRequiredNullsAndArrays(t *testing.T) {
	records, working := resumeFixture(t, false, "")
	result, err := Resume(context.Background(), Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schemaVersion", "kind", "complete", "scope", "orientation", "context", "recovery", "comparison", "diagnostics"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing top-level key %q: %s", key, encoded)
		}
	}
	scope := raw["scope"].(map[string]any)
	if scope["taskId"] != nil {
		t.Fatalf("unknown task identity must be null: %s", encoded)
	}
	recovery := raw["recovery"].(map[string]any)
	comparison := raw["comparison"].(map[string]any)
	for name, value := range map[string]any{"observations": recovery["observations"], "candidates": recovery["candidates"], "comparison candidates": comparison["candidates"], "diagnostics": raw["diagnostics"]} {
		if value == nil {
			t.Fatalf("%s must be an array: %s", name, encoded)
		}
	}
}

func TestResumeCancellationAndInvalidRequests(t *testing.T) {
	records, working := resumeFixture(t, true, "task")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Resume(ctx, Request{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	for _, request := range []Request{
		{},
		{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxFiles: -1},
		{RecordsDirectory: records, WorkingDirectory: working, TicketPath: "task.md", MaxCacheFiles: -1},
	} {
		if _, err := Resume(context.Background(), request); err == nil {
			t.Fatalf("invalid request accepted: %+v", request)
		}
	}
}

func TestNormalizeDiagnosticsKeepsReaderOrderAndStableIdentity(t *testing.T) {
	contextDiagnostics := []taskcontext.Diagnostic{
		{Code: "source_missing", Severity: "error", Message: "context wording", Path: "/missing", From: "/task", Link: "missing.md"},
		{Code: "dependency_cycle", Severity: "warning", Message: "cycle", Path: "/task"},
	}
	orientationDiagnostics := []orientation.Diagnostic{
		{Code: "source_missing", Severity: "error", Message: "different wording", Path: "/missing", From: "/task", Link: "missing.md"},
		{Code: "invalid_profile", Severity: "error", Message: "profile", Path: "/other"},
	}
	got := normalizeDiagnostics(contextDiagnostics, orientationDiagnostics)
	if len(got) != 3 || got[0].Message != "context wording" || got[1].Code != "dependency_cycle" || got[2].Code != "invalid_profile" {
		t.Fatalf("normalized diagnostics: %+v", got)
	}
	if got[0].Path == nil || *got[0].Path != "/missing" || got[1].From != nil || got[2].ObservationID != nil {
		t.Fatalf("diagnostic attribution: %+v", got)
	}
}

func resumeFixture(t *testing.T, identity bool, taskID string) (string, string) {
	t.Helper()
	root := t.TempDir()
	records, working := filepath.Join(root, "records"), filepath.Join(root, "checkout")
	if err := os.MkdirAll(records, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(working, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\nCurrent direction.\n## Current commitments\n[Task](task.md)\n## Open decisions\nNone\n"
	writeResumeFile(t, filepath.Join(records, "project.md"), manifest)
	writeResumeFile(t, filepath.Join(records, "task.md"), workItemDocument(taskID, identity))
	return canonical(t, records), canonical(t, working)
}

func workItemDocument(id string, identity bool) string {
	identityFields := ""
	if identity {
		identityFields = "id: " + id + "\n"
	}
	return "---\ntype: WorkItem\n" + identityFields + "title: Task\ntriage: ready-for-agent\nexecution: unstarted\n---\n## Acceptance criteria\n- Resume it.\n## Blocked by\nNone\n## Blocked by decisions\nNone\n"
}

func observationsPath(working, projectID, taskID string) string {
	return filepath.Join(working, ".context-cache", "resume-v1", namespaceKey(projectID), namespaceKey(taskID), "observations")
}

func namespaceName(index int) string {
	return filepath.Base(filepath.Join("entry", string(rune('a'+index%26)))) + "-" + string(rune('a'+index/26%26))
}

func writeResumeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func canonical(t *testing.T, path string) string {
	t.Helper()
	result, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[relative] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func contains(text, fragment string) bool {
	for index := 0; index+len(fragment) <= len(text); index++ {
		if text[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

func hasResumeDiagnostic(diagnostics []orientation.Diagnostic, code, path string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Path == path {
			return true
		}
	}
	return false
}

func hasContextDiagnostic(diagnostics []taskcontext.Diagnostic, code, path string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Path == path {
			return true
		}
	}
	return false
}
