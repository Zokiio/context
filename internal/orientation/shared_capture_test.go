package orientation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
	"github.com/google/go-cmp/cmp"
)

func TestSharedCapturePreservesBytesAcrossReaderPhases(t *testing.T) {
	project, requirement, files := sharedCaptureProject(t)
	var totalBytes int64
	for _, content := range files {
		totalBytes += int64(len(content))
	}
	capture := newManifestFirstCapture(t, project, []string{filepath.Dir(project)}, recordread.Limits{MaxFiles: 4, MaxBytes: totalBytes})
	defer capture.Close()

	task, err := taskcontext.AssembleWithCapture(context.Background(), "root.md", capture)
	if err != nil || !task.Complete || len(task.Sources) != 2 {
		t.Fatalf("task context: %+v, %v", task, err)
	}
	if err := os.WriteFile(filepath.Join(project, "root.md"), []byte("changed and no longer a work item\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requirement, []byte("changed requirement\n"), 0600); err != nil {
		t.Fatal(err)
	}

	projectView, err := orientation.OrientWithCapture(context.Background(), capture)
	if err != nil || !projectView.Complete || !projectView.InventoryComplete || len(projectView.WorkItems) != 1 || projectView.WorkItems[0].ID == nil || *projectView.WorkItems[0].ID != "task" {
		t.Fatalf("orientation did not reuse task bytes: %+v, %v", projectView, err)
	}
	rootDigest := task.Sources[0].SHA256
	requirementDigest := task.Sources[1].SHA256
	foundRoot, foundRequirement := false, false
	for _, source := range projectView.Sources {
		switch filepath.Base(source.Path) {
		case "root.md":
			foundRoot = true
			if source.SHA256 != rootDigest {
				t.Fatalf("root changed between phases: %+v", source)
			}
		case "requirements.md":
			foundRequirement = true
			if source.SHA256 != requirementDigest {
				t.Fatalf("requirement changed between phases: %+v", source)
			}
		}
	}
	if !foundRoot || !foundRequirement {
		t.Fatalf("shared sources missing from orientation: %+v", projectView.Sources)
	}
	if usage := capture.Usage(); usage.Files != 4 || usage.Bytes != totalBytes {
		t.Fatalf("shared sources counted twice: %+v", usage)
	}
}

func TestSharedLimitKeepsCompleteTaskContextAndPartialOrientation(t *testing.T) {
	project, _, _ := sharedCaptureProject(t)
	capture := newManifestFirstCapture(t, project, []string{filepath.Dir(project)}, recordread.Limits{MaxFiles: 3})
	defer capture.Close()

	task, err := taskcontext.AssembleWithCapture(context.Background(), "root.md", capture)
	if err != nil || !task.Complete || !task.TraversalComplete || len(task.Sources) != 2 {
		t.Fatalf("task context: %+v, %v", task, err)
	}
	projectView, err := orientation.OrientWithCapture(context.Background(), capture)
	if err != nil || projectView.Complete || projectView.InventoryComplete || len(projectView.WorkItems) != 1 || !hasCode(projectView, "source_limit_exceeded") {
		t.Fatalf("partial orientation: %+v, %v", projectView, err)
	}
	if usage := capture.Usage(); usage.Files != 3 {
		t.Fatalf("limit accounting: %+v", usage)
	}
}

func TestSharedLimitStopsNewSourcesAcrossReaderPhases(t *testing.T) {
	project, _, files := sharedCaptureProject(t)
	maxBytes := int64(len(files["project.md"]) + len(files["root.md"]) + 5)
	capture := newManifestFirstCapture(t, project, []string{filepath.Dir(project)}, recordread.Limits{MaxFiles: 10, MaxBytes: maxBytes})
	defer capture.Close()

	task, err := taskcontext.AssembleWithCapture(context.Background(), "root.md", capture)
	if err != nil || task.Complete || task.TraversalComplete || !capture.Exhausted() || len(task.Sources) != 1 || !taskHasCode(task, "source_limit_exceeded") {
		t.Fatalf("task context breach: %+v, %v", task, err)
	}
	usageBefore := capture.Usage()
	projectView, err := orientation.OrientWithCapture(context.Background(), capture)
	if err != nil || projectView.Complete || projectView.InventoryComplete || len(projectView.WorkItems) != 1 || !hasCode(projectView, "source_omitted") {
		t.Fatalf("partial orientation after earlier breach: %+v, %v", projectView, err)
	}
	if diff := cmp.Diff(usageBefore, capture.Usage()); diff != "" {
		t.Fatalf("a smaller source was admitted after the first breach (-before +after):\n%s", diff)
	}
}

func TestSharedCaptureMatchesStandaloneReadersAndCancellationWritesNothing(t *testing.T) {
	project, _, _ := sharedCaptureProject(t)
	allowed := []string{filepath.Dir(project)}
	standaloneTask, err := taskcontext.Assemble(context.Background(), taskcontext.Request{ProjectDir: project, TicketPath: "root.md", AllowedSourceDirs: allowed})
	if err != nil {
		t.Fatal(err)
	}
	standaloneProject, err := orientation.Orient(context.Background(), orientation.Request{ProjectDir: project, AllowedSourceDirs: allowed})
	if err != nil {
		t.Fatal(err)
	}
	capture := newManifestFirstCapture(t, project, allowed, recordread.Limits{})
	defer capture.Close()
	sharedTask, err := taskcontext.AssembleWithCapture(context.Background(), "root.md", capture)
	if err != nil {
		t.Fatal(err)
	}
	sharedProject, err := orientation.OrientWithCapture(context.Background(), capture)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(standaloneTask, sharedTask); diff != "" {
		t.Fatalf("task context changed (-standalone +shared):\n%s", diff)
	}
	if diff := cmp.Diff(standaloneProject, sharedProject); diff != "" {
		t.Fatalf("orientation changed (-standalone +shared):\n%s", diff)
	}

	before := directorySnapshot(t, filepath.Dir(project))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := taskcontext.AssembleWithCapture(ctx, "root.md", capture); err != context.Canceled {
		t.Fatalf("task cancellation: %v", err)
	}
	if _, err := orientation.OrientWithCapture(ctx, capture); err != context.Canceled {
		t.Fatalf("orientation cancellation: %v", err)
	}
	if diff := cmp.Diff(before, directorySnapshot(t, filepath.Dir(project))); diff != "" {
		t.Fatalf("read operations changed files (-before +after):\n%s", diff)
	}
}

func newManifestFirstCapture(t *testing.T, project string, allowed []string, limits recordread.Limits) *recordread.Capture {
	t.Helper()
	capture, err := recordread.NewCapture(project, allowed, limits)
	if err != nil {
		t.Fatal(err)
	}
	manifest, diagnostic := capture.Read(filepath.Join(project, "project.md"), recordread.RecordSource)
	if diagnostic != nil {
		capture.Close()
		t.Fatal(diagnostic)
	}
	if limit := capture.Admit(manifest); limit != nil {
		capture.Close()
		t.Fatal(limit)
	}
	return capture
}

func sharedCaptureProject(t *testing.T) (string, string, map[string]string) {
	t.Helper()
	parent := t.TempDir()
	project := filepath.Join(parent, "bundle")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"project.md":      "---\ntype: Project\nid: project\ntitle: Project\n---\n## Goals\n[Requirements](../requirements.md)\n## Current commitments\n[Task](root.md)\n## Open decisions\nNone\n",
		"root.md":         workItem("task", "unstarted", "## Context\n[Requirements](../requirements.md)\n"),
		"extra.md":        "Unmanaged project note.\n",
		"requirements.md": "original requirement\n",
	}
	for name, content := range files {
		path := filepath.Join(project, name)
		if name == "requirements.md" {
			path = filepath.Join(parent, name)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return project, filepath.Join(parent, "requirements.md"), files
}

func directorySnapshot(t *testing.T, root string) map[string]string {
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

func taskHasCode(result taskcontext.Result, code string) bool {
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
