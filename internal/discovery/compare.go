package discovery

import (
	"fmt"
	"strings"
)

func samePathValue(left, right PathValue) bool {
	return SamePath(left.Canonical, right.Canonical)
}

func samePathSet(left, right []PathValue) bool {
	contains := func(paths []PathValue, target PathValue) bool {
		for _, path := range paths {
			if samePathValue(path, target) {
				return true
			}
		}
		return false
	}
	for _, path := range left {
		if !contains(right, path) {
			return false
		}
	}
	for _, path := range right {
		if !contains(left, path) {
			return false
		}
	}
	return true
}

func uniquePaths(paths []PathValue) []string {
	result := []string{}
	for _, path := range paths {
		duplicate := false
		for _, previous := range result {
			if SamePath(previous, path.Canonical) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, path.Canonical)
		}
	}
	return result
}

func sameOptionalPath(left, right *PathValue) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return samePathValue(*left, *right)
}

func sameWorkspace(left, right WorkspaceDeclaration) bool {
	if left.ID != right.ID || left.Title != right.Title || len(left.Members) != len(right.Members) {
		return false
	}
	for i, member := range left.Members {
		other := right.Members[i]
		if member.Key != other.Key || member.Title != other.Title ||
			!samePathValue(member.Records, other.Records) ||
			!sameOptionalPath(member.Directory, other.Directory) ||
			!samePathSet(member.AllowSources, other.AllowSources) {
			return false
		}
	}
	return true
}

func conflict(kind Kind, directory string, left, right candidate) error {
	return fmt.Errorf("conflicting %s declarations at %q: %s conflicts with %s; repair one declaration so their effective values agree", kind, directory, describeCandidate(left), describeCandidate(right))
}

func describeCandidate(item candidate) string {
	origin := fmt.Sprintf("%s field %s", item.origin.Path, item.origin.Field)
	if item.project == nil {
		return fmt.Sprintf("%s (workspace id %q, title %q)", origin, item.workspace.ID, item.workspace.Title)
	}
	roots := []string{}
	for _, root := range item.project.AllowSources {
		roots = append(roots, fmt.Sprintf("%q resolves to %q", root.Authored, root.Canonical))
	}
	return fmt.Sprintf("%s (records %q resolves to %q, allowSources [%s])", origin,
		item.project.Records.Authored, item.project.Records.Canonical, strings.Join(roots, ", "))
}
