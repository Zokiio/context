package recordread

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Role int

const (
	RecordSource Role = iota
	DocumentSource
)

type permittedRoot struct {
	path string
	root *os.Root
}

type Reader struct {
	project   string
	roots     []permittedRoot
	snapshots map[string]Source
	failures  map[string]Diagnostic
}

func NewReader(project string, allowed []string) (*Reader, error) {
	reader := &Reader{snapshots: map[string]Source{}, failures: map[string]Diagnostic{}}
	paths := append([]string{project}, allowed...)
	for index, path := range paths {
		label := "allowed source"
		if index == 0 {
			label = "project"
		}
		if path == "" {
			reader.Close()
			return nil, fmt.Errorf("%s directory must be nonempty", label)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("resolve %s: %w", label, err)
		}
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("resolve %s: %w", label, err)
		}
		root, err := os.OpenRoot(resolved)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("open %s: %w", label, err)
		}
		reader.roots = append(reader.roots, permittedRoot{path: resolved, root: root})
	}
	reader.project = reader.roots[0].path
	return reader, nil
}

func (r *Reader) Close() {
	for _, root := range r.roots {
		_ = root.root.Close()
	}
}

func (r *Reader) permitted(path string, role Role) (*os.Root, string) {
	roots := r.roots
	if role == RecordSource {
		roots = roots[:1]
	}
	for _, root := range roots {
		relative, err := filepath.Rel(root.path, path)
		if err == nil && filepath.IsLocal(relative) {
			return root.root, relative
		}
	}
	return nil, ""
}

// read keeps ticket scope distinct from document scope for additional document roots.
func (r *Reader) Read(path string, role Role) (Source, *Diagnostic) {
	path = filepath.Clean(path)
	fail := func(code string, err error) (Source, *Diagnostic) {
		return Source{}, &Diagnostic{Code: code, Severity: "error", Message: err.Error(), Path: path}
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Resolve existing ancestors so a missing child under a root alias gets the
		// same scope decision as an existing child. Never read through this fallback.
		attempted := path
		suffix := []string{}
		for {
			canonical, resolveErr := filepath.EvalSymlinks(attempted)
			if resolveErr == nil {
				for i := len(suffix) - 1; i >= 0; i-- {
					canonical = filepath.Join(canonical, suffix[i])
				}
				path = canonical
				break
			}
			parent := filepath.Dir(attempted)
			if parent == attempted {
				break
			}
			suffix = append(suffix, filepath.Base(attempted))
			attempted = parent
		}
		if root, _ := r.permitted(path, role); root == nil {
			return fail("source_outside_scope", errors.New("source is outside the permitted directories"))
		}
		return fail(readErrorCode(err), err)
	}
	path = resolved
	root, relative := r.permitted(path, role)
	if root == nil {
		return fail("source_outside_scope", errors.New("source resolves outside the permitted directories"))
	}
	if diagnostic, exists := r.failures[path]; exists {
		return r.snapshots[path], &diagnostic
	}
	if source, exists := r.snapshots[path]; exists {
		return source, nil
	}
	readFailure := func(code string, err error) (Source, *Diagnostic) {
		source, diagnostic := fail(code, err)
		r.failures[path] = *diagnostic
		return source, diagnostic
	}
	content, err := root.ReadFile(relative)
	if err != nil {
		return readFailure(readErrorCode(err), err)
	}
	source := Source{Path: path, Text: string(content), SHA256: fmt.Sprintf("%x", sha256.Sum256(content)), Reasons: []Reason{}}
	r.snapshots[path] = source
	if !utf8.Valid(content) {
		_, diagnostic := readFailure("invalid_source_encoding", errors.New("source must be UTF-8 to preserve its bytes in JSON"))
		return source, diagnostic
	}
	return source, nil
}

func readErrorCode(err error) string {
	if errors.Is(err, os.ErrNotExist) {
		return "source_missing"
	}
	return "source_unreadable"
}

func (r *Reader) LinkPath(from string, link Relationship) (string, *Diagnostic) {
	u, err := url.Parse(link.Destination)
	if err != nil || u.Scheme != "" || u.Host != "" || u.RawQuery != "" {
		return "", &Diagnostic{Code: "unsupported_source", Severity: "error", Message: "relationship must identify a local file", Path: from, From: from, Link: link.Link}
	}
	if u.Path == "" {
		return from, nil
	}
	if strings.HasPrefix(u.Path, "/") {
		return filepath.Join(r.project, filepath.FromSlash(strings.TrimLeft(u.Path, "/"))), nil
	}
	return filepath.Join(filepath.Dir(from), filepath.FromSlash(u.Path)), nil
}

func (r *Reader) Project() string { return r.project }
