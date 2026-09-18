package resumption

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cacheBudget bounds actual directory and file reads, separately from current sources.
type cacheBudget struct {
	entries int
	bytes   int64
	stopped bool
}

func (b *cacheBudget) names(directory interface{ Readdirnames(int) ([]string, error) }) ([]string, bool, error) {
	n := b.entries
	if n < int(^uint(0)>>1) {
		n++
	}
	names, err := directory.Readdirnames(n)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	complete := len(names) <= b.entries
	if !complete {
		names = names[:b.entries]
		b.stopped = true
	}
	b.entries -= len(names)
	sort.Strings(names)
	return names, complete, nil
}

var errCacheLimit = errors.New("recovery cache limit exceeded")

func (b *cacheBudget) read(root *os.Root, name string) ([]byte, error) {
	if b.stopped || b.entries == 0 {
		b.stopped = true
		return nil, errCacheLimit
	}
	b.entries--
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("fixed recovery file must be a regular file, not an alias")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, actual) {
		return nil, errors.New("recovery file changed during opening")
	}
	data, err := readCacheBytes(file, b.bytes)
	if err != nil {
		if errors.Is(err, errCacheLimit) {
			b.stopped = true
		}
		return nil, err
	}
	b.bytes -= int64(len(data))
	return data, nil
}
func readCacheBytes(reader io.Reader, remaining int64) ([]byte, error) {
	limit := remaining
	if limit < int64(^uint64(0)>>1) {
		limit++
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > remaining {
		return nil, errCacheLimit
	}
	return data, nil
}

// openCacheDirectory pins each real directory as it descends. No subsequent
// recovery read uses an unrestricted absolute pathname.
func openCacheDirectory(working, relative string) (*os.Root, error) {
	if !filepath.IsLocal(relative) {
		return nil, errors.New("recovery path escapes checkout")
	}
	root, err := os.OpenRoot(working)
	if err != nil {
		return nil, err
	}
	current := working
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, e := root.Lstat(part)
		if e != nil {
			root.Close()
			return nil, e
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			message := "recovery path must be a directory"
			if info.Mode()&os.ModeSymlink != 0 {
				message = "recovery path must not be a symbolic link"
			}
			return nil, &os.PathError{Op: "inspect", Path: current, Err: errors.New(message)}
		}
		child, e := root.OpenRoot(part)
		if e != nil {
			root.Close()
			return nil, e
		}
		actual, e := child.Stat(".")
		root.Close()
		if e != nil || !os.SameFile(info, actual) {
			child.Close()
			return nil, errors.New("recovery directory changed during opening")
		}
		root = child
	}
	return root, nil
}

func inspectRecovery(ctx context.Context, result *Result, request Request) error {
	recovery := &result.Recovery
	recovery.GraphStatus = "valid"
	recovery.InventoryComplete = true
	path := filepath.Join(result.Scope.CacheRoot, namespaceKey(*result.Scope.ProjectID), namespaceKey(*result.Scope.TaskID), "observations")
	relative, _ := filepath.Rel(result.Scope.WorkingDirectory, path)
	root, err := openCacheDirectory(result.Scope.WorkingDirectory, relative)
	findings := []Diagnostic{}
	add := func(code, message, path, id string) {
		findings = append(findings, Diagnostic{Code: code, Severity: "error", Message: message, Path: optionalString(path), ObservationID: optionalString(id)})
	}
	defer func() {
		sort.SliceStable(findings, func(i, j int) bool {
			a, b := findings[i], findings[j]
			if *a.Path != *b.Path {
				return *a.Path < *b.Path
			}
			return a.Code < b.Code
		})
		result.Diagnostics = append(result.Diagnostics, findings...)
	}()
	incomplete := func() {
		recovery.InventoryComplete = false
		if recovery.GraphStatus != "invalid" {
			recovery.GraphStatus = "incomplete"
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		recovery.Status = "absent"
		result.Comparison.Complete = result.Context.Complete
		return nil
	}
	if err != nil {
		incomplete()
		failurePath := path
		var pathError *os.PathError
		if errors.As(err, &pathError) && filepath.IsAbs(pathError.Path) {
			failurePath = pathError.Path
		}
		add("recovery_source_omitted", err.Error(), failurePath, "")
		return nil
	}
	defer root.Close()
	directory, err := root.Open(".")
	if err != nil {
		incomplete()
		add("recovery_source_omitted", err.Error(), path, "")
		return nil
	}
	defer directory.Close()
	budget := cacheBudget{entries: request.MaxCacheFiles, bytes: request.MaxCacheBytes}
	names, complete, err := budget.names(directory)
	if err != nil {
		incomplete()
		add("recovery_source_omitted", err.Error(), path, "")
		return nil
	}
	if !complete {
		incomplete()
		add("recovery_limit_exceeded", "observation enumeration exceeded the cache entry limit; observed names are only a subset", path, "")
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		notePath := filepath.Join(path, name, "note.md")
		if budget.stopped {
			add("recovery_source_omitted", "note not read after cache limit", notePath, "")
			continue
		}
		info, e := root.Lstat(name)
		if e != nil {
			incomplete()
			add("recovery_incomplete_observation", e.Error(), notePath, "")
			continue
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			recovery.GraphStatus = "invalid"
			add("recovery_invalid_note", "observation must be a real directory", filepath.Join(path, name), "")
			continue
		}
		observationRoot, e := root.OpenRoot(name)
		if e != nil {
			incomplete()
			add("recovery_incomplete_observation", e.Error(), notePath, "")
			continue
		}
		actual, e := observationRoot.Stat(".")
		if e != nil || !os.SameFile(info, actual) {
			observationRoot.Close()
			incomplete()
			add("recovery_incomplete_observation", "observation changed during opening", notePath, "")
			continue
		}
		data, e := budget.read(observationRoot, "note.md")
		observationRoot.Close()
		if e != nil {
			incomplete()
			code := "recovery_incomplete_observation"
			if errors.Is(e, errCacheLimit) {
				code = "recovery_limit_exceeded"
			}
			add(code, e.Error(), notePath, "")
			continue
		}
		observation, e := parseNote(data, notePath, name, *result.Scope.ProjectID, *result.Scope.TaskID)
		if e != nil {
			recovery.GraphStatus = "invalid"
			code := "recovery_invalid_note"
			if errors.Is(e, errNoteIdentity) {
				code = "recovery_identity_mismatch"
			}
			add(code, e.Error(), notePath, "")
			continue
		}
		recovery.Observations = append(recovery.Observations, observation)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if complete && len(names) > 1 {
		return fmt.Errorf("multiple recovery observations at %q are not supported by this implementation slice; history evaluation is added by the next recovery slice", path)
	}
	if len(recovery.Observations) == 1 && len(recovery.Observations[0].Predecessors) > 0 {
		return errors.New("recovery predecessor history is not supported by this implementation slice; history evaluation is added by the next recovery slice")
	}
	if !recovery.InventoryComplete || recovery.GraphStatus != "valid" {
		return nil
	}
	if len(names) == 0 {
		recovery.Status = "absent"
		result.Comparison.Complete = result.Context.Complete
		return nil
	}
	recovery.Status = "available"
	observation := &recovery.Observations[0]
	recovery.Candidates = append(recovery.Candidates, observation.ID)
	comparison := CandidateComparison{ObservationID: observation.ID, Sources: []SourceDifference{}}
	// Re-open through checked components, rejecting a replaced alias.
	observationRoot, e := openCacheDirectory(result.Scope.WorkingDirectory, filepath.Join(relative, observation.ID))
	if e != nil {
		observation.Snapshot.Status = "unavailable"
		add("recovery_source_omitted", e.Error(), observation.Snapshot.Path, observation.ID)
	} else {
		defer observationRoot.Close()
		data, e := budget.read(observationRoot, "context.json")
		if e != nil {
			observation.Snapshot.Status = "unavailable"
			code := "recovery_source_omitted"
			if errors.Is(e, errCacheLimit) {
				code = "recovery_limit_exceeded"
			}
			add(code, e.Error(), observation.Snapshot.Path, observation.ID)
		} else {
			retained, e := validateSnapshot(data, observation)
			if e != nil {
				observation.Snapshot.Status = "invalid"
				add("recovery_snapshot_invalid", e.Error(), observation.Snapshot.Path, observation.ID)
			} else {
				comparison = compareSnapshot(result.Context, retained, observation, result.Scope.RecordsDirectory, request.AllowedSourceDirs, &findings)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	result.Comparison.Candidates = append(result.Comparison.Candidates, comparison)
	result.Comparison.BaselineAvailable = comparison.BaselineAvailable
	result.Comparison.Complete = comparison.Complete
	return nil
}
