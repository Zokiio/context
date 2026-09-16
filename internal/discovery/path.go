package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// PathValue keeps the authored value and encountered location for diagnostics.
// Canonical can identify a missing target. Err records failures that prevent
// establishing that identity; callers validate availability after selection.
type PathValue struct {
	Origin    Origin
	Authored  string
	Path      string
	Canonical string
	Err       error
}

func savedPath(configPath, field, authored string) PathValue {
	path := authored
	if !filepath.IsAbs(path) {
		directory, _ := filepath.Split(configPath)
		path = directory + path
	}
	canonical, err := CanonicalPath(path)
	return PathValue{Origin: Origin{Path: configPath, Field: field}, Authored: authored,
		Path: path, Canonical: canonical, Err: err}
}

// CanonicalPath resolves existing symlinks, including dangling links, without
// requiring the final location to exist. It does not expand ~ or environment
// variables. On an identity-resolution failure, the returned path retains the
// resolved prefix and the unresolved remainder for diagnostics.
func CanonicalPath(path string) (string, error) {
	absolute, err := absolutePath(path)
	if err != nil {
		return path, err
	}
	root, pending := pathParts(absolute)
	canonical, links := root, 0
	for len(pending) != 0 {
		part := pending[0]
		pending = pending[1:]
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			canonical = filepath.Dir(canonical)
			continue
		}
		next := filepath.Join(canonical, part)
		info, err := os.Lstat(next)
		if err != nil {
			target := filepath.Join(append([]string{next}, pending...)...)
			if os.IsNotExist(err) {
				return target, nil
			}
			return target, err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			if len(pending) != 0 && !info.IsDir() {
				return filepath.Join(append([]string{next}, pending...)...), &os.PathError{Op: "resolve", Path: next, Err: syscall.ENOTDIR}
			}
			canonical = next
			continue
		}
		links++
		if links > 255 {
			return filepath.Join(append([]string{next}, pending...)...), fmt.Errorf("resolve %s: too many symbolic links", next)
		}
		target, err := os.Readlink(next)
		if err != nil {
			return filepath.Join(append([]string{next}, pending...)...), err
		}
		if filepath.IsAbs(target) {
			var parts []string
			canonical, parts = pathParts(target)
			pending = append(parts, pending...)
		} else {
			pending = append(strings.Split(target, string(filepath.Separator)), pending...)
		}
	}
	return canonical, nil
}

func pathParts(path string) (string, []string) {
	root := filepath.VolumeName(path) + string(filepath.Separator)
	return root, strings.Split(strings.TrimPrefix(path, root), string(filepath.Separator))
}

// SamePath compares canonical identities, accounting for filesystems where
// different spellings identify the same existing location.
func SamePath(left, right string) bool {
	if left == right {
		return true
	}
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

// Preserve .. until the filesystem walk has followed preceding symlinks.
func absolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd + string(filepath.Separator) + path, nil
}
