// Package discoveryconfig reads declared project and workspace configuration.
// It does not discover, select, or write a scope.
package discoveryconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/recordread"
)

type Profile uint8

const (
	SharedProfile Profile = iota + 1
	PersonalProfile
)

// Document retains the original text for future edits. Metadata includes unknown
// fields at every nesting level; Source also preserves YAML syntax and spelling.
// Exactly one of Shared and Personal is populated after a successful Read.
type Document struct {
	File     string
	Source   string
	Body     string
	Metadata map[string]any
	Shared   *SharedConfig
	Personal *PersonalConfig
}

type SharedConfig struct {
	Project   *ProjectBinding
	Workspace *WorkspaceDeclaration
}

type PersonalConfig struct {
	Projects   []ProjectRegistration
	Workspaces []WorkspaceRegistration
}

type ProjectBinding struct {
	Records      TargetPath
	AllowSources []TargetPath
}

type ProjectRegistration struct {
	ProjectBinding
	Key       string
	Alias     *string
	Directory TargetPath
}

type WorkspaceDeclaration struct {
	ID      string
	Title   string
	Members []WorkspaceMember
}

type WorkspaceRegistration struct {
	WorkspaceDeclaration
	Key       string
	Alias     *string
	Directory *TargetPath
}

type WorkspaceMember struct {
	ProjectBinding
	Key       string
	Title     *string
	Directory *TargetPath
}

// TargetPath distinguishes an authored value, its absolute spelling, and its
// observed filesystem identity. Canonical is empty iff Failure is non-nil.
// Successful canonicalization does not establish directory access or readiness.
type TargetPath struct {
	File      string
	Field     string
	Authored  string
	Resolved  string
	Canonical string
	Failure   *FieldError
}

// FieldError identifies the declaring file and nested configuration field.
// Value and Resolved carry filesystem provenance when a target cannot resolve.
type FieldError struct {
	File     string
	Field    string
	Value    string
	Resolved string
	Message  string
	Err      error
}

func (e *FieldError) Error() string {
	message := fmt.Sprintf("%s: %s: %s", e.File, e.Field, e.Message)
	if e.Value != "" {
		message += fmt.Sprintf(" (value %q, resolved %q)", e.Value, e.Resolved)
	}
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}

func (e *FieldError) Unwrap() error { return e.Err }

// Read requires an absolute encountered filename and an explicit profile, so it
// has no ambient cwd or home dependency. It validates structure and observes path
// identities without requiring every registered target to be available. Target
// failures remain on their fields; malformed configuration returns an error.
func Read(filename string, profile Profile) (Document, error) {
	fail := func(message string, err error) (Document, error) {
		return Document{}, &FieldError{File: filename, Field: "$", Message: message, Err: err}
	}
	if !filepath.IsAbs(filename) {
		return fail("configuration filename must be absolute", nil)
	}
	if profile != SharedProfile && profile != PersonalProfile {
		return fail("configuration profile must be shared or personal", nil)
	}
	source, err := os.ReadFile(filename)
	if err != nil {
		return fail("read configuration", err)
	}
	if !utf8.Valid(source) {
		return fail("configuration must be UTF-8", nil)
	}
	parsed, err := recordread.ParseDocument(source)
	if err != nil {
		return fail("parse configuration frontmatter", err)
	}
	if parsed.Metadata == nil {
		return fail("configuration requires YAML frontmatter", nil)
	}
	d := decoder{file: filename}
	doc := Document{File: filename, Source: string(source), Body: string(parsed.Body), Metadata: parsed.Metadata}
	d.decode(&doc, profile)
	if d.err != nil {
		return Document{}, d.err
	}
	return doc, nil
}
