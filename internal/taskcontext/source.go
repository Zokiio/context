package taskcontext

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

type sourceRole int

const (
	ticketSource sourceRole = iota
	documentSource
)

type permittedRoot struct {
	path string
	root *os.Root
}

type sourceReader struct {
	project string
	roots   []permittedRoot
}

func newSourceReader(request Request) (*sourceReader, error) {
	reader := &sourceReader{}
	paths := append([]string{request.ProjectDir}, request.AllowedSourceDirs...)
	for index, path := range paths {
		label := "allowed source"
		if index == 0 {
			label = "project"
		}
		if path == "" {
			reader.close()
			return nil, fmt.Errorf("%s directory must be nonempty", label)
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			reader.close()
			return nil, fmt.Errorf("resolve %s: %w", label, err)
		}
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			reader.close()
			return nil, fmt.Errorf("resolve %s: %w", label, err)
		}
		root, err := os.OpenRoot(resolved)
		if err != nil {
			reader.close()
			return nil, fmt.Errorf("open %s: %w", label, err)
		}
		reader.roots = append(reader.roots, permittedRoot{path: resolved, root: root})
	}
	reader.project = reader.roots[0].path
	return reader, nil
}

func (r *sourceReader) close() {
	for _, root := range r.roots {
		_ = root.root.Close()
	}
}

func (r *sourceReader) permitted(path string, role sourceRole) (*os.Root, string) {
	roots := r.roots
	if role == ticketSource {
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
func (r *sourceReader) read(path string, role sourceRole) (Source, *Diagnostic) {
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
	content, err := root.ReadFile(relative)
	if err != nil {
		return fail(readErrorCode(err), err)
	}
	if !utf8.Valid(content) {
		return fail("invalid_source_encoding", errors.New("source must be UTF-8 to preserve its bytes in JSON"))
	}
	return Source{Path: path, Text: string(content), SHA256: fmt.Sprintf("%x", sha256.Sum256(content)), Reasons: []Reason{}}, nil
}

func readErrorCode(err error) string {
	if errors.Is(err, os.ErrNotExist) {
		return "source_missing"
	}
	return "source_unreadable"
}

func (r *sourceReader) linkPath(from string, link relationship) (string, *Diagnostic) {
	u, err := url.Parse(link.destination)
	if err != nil || u.Scheme != "" || u.Host != "" || u.RawQuery != "" {
		return "", &Diagnostic{Code: "unsupported_source", Severity: "error", Message: "relationship must identify a local file", Path: from, From: from, Link: link.link}
	}
	if u.Path == "" {
		return from, nil
	}
	if strings.HasPrefix(u.Path, "/") {
		return filepath.Join(r.project, filepath.FromSlash(strings.TrimLeft(u.Path, "/"))), nil
	}
	return filepath.Join(filepath.Dir(from), filepath.FromSlash(u.Path)), nil
}
