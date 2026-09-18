package resumption

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type observationInspection struct {
	absent     bool
	diagnostic *Diagnostic
}

func inspectEmptyObservations(ctx context.Context, working, observations string, maxFiles int) (observationInspection, error) {
	if err := ctx.Err(); err != nil {
		return observationInspection{}, err
	}
	relative, err := filepath.Rel(working, observations)
	if err != nil || !filepath.IsLocal(relative) {
		return incompleteObservationInspection(observations, fmt.Sprintf("recovery observations path %q escapes working directory %q", observations, working)), nil
	}
	root, err := os.OpenRoot(working)
	if err != nil {
		return observationInspection{}, fmt.Errorf("open working directory %q for recovery inspection: %w", working, err)
	}
	defer root.Close()
	current := ""
	for _, component := range splitLocalPath(relative) {
		if err := ctx.Err(); err != nil {
			return observationInspection{}, err
		}
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return observationInspection{absent: true}, nil
		}
		if err != nil {
			path := filepath.Join(working, current)
			return incompleteObservationInspection(path, fmt.Sprintf("inspect recovery path %q: %v", path, err)), nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			path := filepath.Join(working, current)
			return incompleteObservationInspection(path, fmt.Sprintf("recovery path %q must not be a symbolic link", path)), nil
		}
		if !info.IsDir() {
			path := filepath.Join(working, current)
			return incompleteObservationInspection(path, fmt.Sprintf("recovery path %q must be a directory", path)), nil
		}
	}
	if maxFiles < 1 {
		return observationInspection{}, errors.New("cache file limit must be positive")
	}
	directory, err := root.Open(relative)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return observationInspection{}, contextErr
		}
		return incompleteObservationInspection(observations, fmt.Sprintf("open recovery observations %q: %v", observations, err)), nil
	}
	defer directory.Close()
	names, err := directory.Readdirnames(1)
	if err != nil && !errors.Is(err, io.EOF) {
		if contextErr := ctx.Err(); contextErr != nil {
			return observationInspection{}, contextErr
		}
		return incompleteObservationInspection(observations, fmt.Sprintf("inspect recovery observations %q: %v", observations, err)), nil
	}
	if len(names) != 0 {
		return observationInspection{}, fmt.Errorf("recovery observations at %q are not supported by this implementation slice; note reading is added by the next recovery slice", observations)
	}
	return observationInspection{absent: true}, nil
}

func incompleteObservationInspection(path, message string) observationInspection {
	return observationInspection{diagnostic: &Diagnostic{
		Code: "recovery_source_omitted", Severity: "error", Message: message,
		Path: optionalString(path), From: nil, Link: nil, ObservationID: nil,
	}}
}

func splitLocalPath(path string) []string {
	result := []string{}
	for path != "." && path != "" {
		directory, name := filepath.Split(path)
		if name != "" {
			result = append([]string{name}, result...)
		}
		path = filepath.Clean(directory)
		if path == string(filepath.Separator) {
			break
		}
	}
	return result
}
