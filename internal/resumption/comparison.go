package resumption

import (
	"os"
	"path/filepath"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
)

func compareSnapshot(current, retained taskcontext.Result, observation *Observation, records string, allowed []string, diagnostics *[]Diagnostic) CandidateComparison {
	result := CandidateComparison{ObservationID: observation.ID, Complete: current.Complete && current.TraversalComplete && retained.Complete && retained.TraversalComplete, Sources: []SourceDifference{}}
	setsComplete := result.Complete
	roots := []string{records}
	for _, root := range allowed {
		absolute, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if physical, err := filepath.EvalSymlinks(absolute); err == nil {
			roots = append(roots, physical)
		}
	}
	currentPaths := map[string]bool{}
	currentRoot := ""
	for _, source := range current.Sources {
		currentPaths[source.Path] = true
		if isRoot(source) {
			currentRoot = source.Path
		}
	}
	previous := make([]ComparedSource, len(retained.Sources))
	identities := make([]string, len(previous))
	used := make([]bool, len(previous))
	status := "valid"
	if !retained.Complete || !retained.TraversalComplete {
		status = "incomplete"
		*diagnostics = append(*diagnostics, Diagnostic{Code: "recovery_source_omitted", Severity: "error", Message: "retained task-context collection is incomplete; unmatched sources remain unknown", Path: &observation.Snapshot.Path, ObservationID: &observation.ID})
	}
	for i, source := range retained.Sources {
		identity, availability := retainedAuthorization(source.Path, roots, currentPaths, isRoot(source) && currentRoot != "")
		identities[i] = identity
		previous[i] = ComparedSource{Path: source.Path, SHA256: &source.SHA256, Availability: availability}
		if availability == "available" {
			previous[i].Text = &source.Text
			result.BaselineAvailable = true
			continue
		}
		result.Complete = false
		code := "recovery_source_omitted"
		if availability == "withheld" {
			code = "recovery_snapshot_outside_scope"
			if status != "unavailable" {
				status = "withheld"
			}
		} else {
			status = "unavailable"
		}
		*diagnostics = append(*diagnostics, Diagnostic{Code: code, Severity: "error", Message: "retained source text is " + availability + " under current source authorization", Path: optionalString(source.Path), ObservationID: &observation.ID})
	}
	for _, source := range current.Sources {
		currentSide := &ComparedSource{Path: source.Path, SHA256: &source.SHA256, Text: &source.Text, Availability: "available"}
		matched := -1
		// Prefer the root's stable identity over an old document occupying its
		// new path. Ordinary sources never receive this relocation match.
		if isRoot(source) {
			for i, old := range retained.Sources {
				if !used[i] && isRoot(old) && previous[i].Availability == "available" {
					matched = i
					break
				}
			}
		}
		if matched < 0 {
			for i := range retained.Sources {
				if !used[i] && identities[i] == source.Path {
					matched = i
					break
				}
			}
		}
		difference := SourceDifference{Status: "unknown", Current: currentSide}
		if matched >= 0 {
			used[matched] = true
			difference.Previous = &previous[matched]
			if previous[matched].Availability == "available" {
				difference.Status = "changed"
				if *previous[matched].SHA256 == source.SHA256 {
					difference.Status = "unchanged"
				}
			}
		} else if setsComplete {
			difference.Status = "added"
		}
		result.Sources = append(result.Sources, difference)
	}
	for i := range previous {
		if used[i] {
			continue
		}
		difference := SourceDifference{Status: "unknown", Previous: &previous[i]}
		if setsComplete && previous[i].Availability == "available" {
			difference.Status = "removed"
		}
		result.Sources = append(result.Sources, difference)
	}
	observation.Snapshot.Status = status
	return result
}

func isRoot(source recordread.Source) bool {
	for _, reason := range source.Reasons {
		if reason.Kind == "root" {
			return true
		}
	}
	return false
}

// A historical path never grants access. Only captured current identity or a
// physical path inside today's roots permits returning its retained text.
func retainedAuthorization(path string, roots []string, current map[string]bool, movedRoot bool) (string, string) {
	if current[path] {
		return path, "available"
	}
	physical, err := filepath.EvalSymlinks(path)
	if err == nil {
		if withinRoots(physical, roots) {
			return physical, "available"
		}
		return physical, "withheld"
	}
	if !os.IsNotExist(err) {
		return "", "unavailable"
	}
	// Only a root move may use a missing filename. Require an existing ancestor
	// within an authorized root, so an old missing checkout cannot authorize it.
	if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", "unavailable"
	}
	ancestor := filepath.Dir(path)
	for {
		resolved, e := filepath.EvalSymlinks(ancestor)
		if e == nil {
			if movedRoot && withinRoots(resolved, roots) {
				suffix, _ := filepath.Rel(ancestor, path)
				return filepath.Join(resolved, suffix), "available"
			}

			return "", "unavailable"
		}
		if !os.IsNotExist(e) {
			return "", "unavailable"
		}
		if info, statErr := os.Lstat(ancestor); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", "unavailable"
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return "", "unavailable"
		}
		ancestor = parent
	}
}
func withinRoots(path string, roots []string) bool {
	for _, root := range roots {
		relative, err := filepath.Rel(root, path)
		if err == nil && filepath.IsLocal(relative) {
			return true
		}
	}
	return false
}
