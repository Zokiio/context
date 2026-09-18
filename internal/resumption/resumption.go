package resumption

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zokiio/context/internal/discovery"
	"github.com/Zokiio/context/internal/orientation"
	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
)

// Resume refreshes current facts and compares one retained root checkpoint.
func Resume(ctx context.Context, request Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if request.RecordsDirectory == "" || request.WorkingDirectory == "" || request.TicketPath == "" {
		return Result{}, errors.New("records directory, working directory, and ticket are required")
	}
	if request.MaxFiles < 0 || request.MaxBytes < 0 || request.MaxCacheFiles < 0 || request.MaxCacheBytes < 0 {
		return Result{}, errors.New("source and cache limits must be positive")
	}
	if request.MaxCacheFiles == 0 {
		request.MaxCacheFiles = DefaultMaxCacheFiles
	}
	if request.MaxCacheBytes == 0 {
		request.MaxCacheBytes = DefaultMaxCacheBytes
	}
	records, err := availableDirectory(request.RecordsDirectory, "records")
	if err != nil {
		return Result{}, err
	}
	working, err := availableDirectory(request.WorkingDirectory, "working")
	if err != nil {
		return Result{}, err
	}
	capture, err := recordread.NewCapture(records, request.AllowedSourceDirs, recordread.Limits{MaxFiles: request.MaxFiles, MaxBytes: request.MaxBytes})
	if err != nil {
		return Result{}, err
	}
	defer capture.Close()

	// Identity comes from the Project manifest, so capture it before the task
	// traversal. The existing readers still own parsing and diagnostics.
	manifest, diagnostic := capture.Read(filepath.Join(records, "project.md"), recordread.RecordSource)
	if diagnostic == nil || manifest.Path != "" {
		_ = capture.Admit(manifest)
	}
	currentContext, err := taskcontext.AssembleWithCapture(ctx, request.TicketPath, capture)
	if err != nil {
		return Result{}, err
	}
	currentOrientation, err := orientation.OrientWithCapture(ctx, capture)
	if err != nil {
		return Result{}, err
	}

	cacheRoot := filepath.Join(working, ".context-cache", "resume-v1")
	result := Result{
		SchemaVersion: 1,
		Kind:          "task-resumption",
		Scope: Scope{
			RecordsDirectory: records,
			WorkingDirectory: working,
			CacheRoot:        cacheRoot,
		},
		Orientation: currentOrientation,
		Context:     currentContext,
		Recovery: Recovery{
			Status: "unknown", InventoryComplete: false, GraphStatus: "not_evaluated",
			Observations: []Observation{}, Candidates: []string{},
		},
		Comparison:  Comparison{BaselineAvailable: false, Complete: false, Candidates: []CandidateComparison{}},
		Diagnostics: normalizeDiagnostics(currentContext.Diagnostics, currentOrientation.Diagnostics),
	}
	result.Scope.ProjectID = projectIdentity(currentOrientation)
	result.Scope.TaskID = taskIdentity(currentContext, currentOrientation)
	if result.Scope.ProjectID != nil && result.Scope.TaskID != nil {
		if err := inspectRecovery(ctx, &result, request); err != nil {
			return Result{}, err
		}
	}
	result.Complete = currentOrientation.Complete && currentContext.Complete && result.Comparison.Complete && result.Recovery.InventoryComplete && result.Recovery.GraphStatus == "valid"
	return result, nil
}

func availableDirectory(path, label string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s directory: %w", label, err)
	}
	canonical, err := discovery.CanonicalPath(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve %s directory %q: %w", label, path, err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("inspect %s directory %q: %w", label, canonical, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s directory %q is not a directory", label, canonical)
	}
	root, err := os.OpenRoot(canonical)
	if err != nil {
		return "", fmt.Errorf("open %s directory %q: %w", label, canonical, err)
	}
	defer root.Close()
	directory, err := root.Open(".")
	if err != nil {
		return "", fmt.Errorf("open %s directory %q: %w", label, canonical, err)
	}
	directory.Close()
	return canonical, nil
}

func projectIdentity(result orientation.Result) *string {
	if !result.InventoryComplete || result.Project == nil {
		return nil
	}
	return effectiveIdentity(result.Project.ID)
}

func taskIdentity(current taskcontext.Result, project orientation.Result) *string {
	if !project.InventoryComplete {
		return nil
	}
	root := ""
	for _, source := range current.Sources {
		for _, reason := range source.Reasons {
			if reason.Kind == "root" {
				if root != "" && root != source.Path {
					return nil
				}
				root = source.Path
			}
		}
	}
	if root == "" {
		return nil
	}
	var identity *string
	for _, work := range project.WorkItems {
		if work.Source != root {
			continue
		}
		if identity != nil || work.IdentityAmbiguous || work.ID == nil {
			return nil
		}
		identity = effectiveIdentity(*work.ID)
	}
	return identity
}

func effectiveIdentity(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func namespaceKey(identity string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
}

func normalizeDiagnostics(contextDiagnostics []taskcontext.Diagnostic, orientationDiagnostics []orientation.Diagnostic) []Diagnostic {
	result := []Diagnostic{}
	seen := map[string]bool{}
	appendDiagnostic := func(code, severity, message, path, from, link string) {
		key := strings.Join([]string{code, severity, path, from, link, ""}, "\x00")
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, Diagnostic{Code: code, Severity: severity, Message: message, Path: optionalString(path), From: optionalString(from), Link: optionalString(link), ObservationID: nil})
	}
	for _, diagnostic := range contextDiagnostics {
		appendDiagnostic(diagnostic.Code, diagnostic.Severity, diagnostic.Message, diagnostic.Path, diagnostic.From, diagnostic.Link)
	}
	for _, diagnostic := range orientationDiagnostics {
		appendDiagnostic(diagnostic.Code, diagnostic.Severity, diagnostic.Message, diagnostic.Path, diagnostic.From, diagnostic.Link)
	}
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
