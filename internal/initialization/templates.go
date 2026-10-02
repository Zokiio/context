// Package initialization prepares a project for the existing ctx readers.
package initialization

import (
	"bytes"
	"embed"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templates embed.FS

//go:generate go test -run ^TestRepositoryGuidance$ -update

type document struct {
	path string
	text []byte
}

type guidance struct {
	Development                                                bool
	Command, Scope, Records, Local, Tracker                    string
	AcceptanceLink, TrackerLink, TaskContextLink, SnapshotLink string
}

func renderGuidance(request Request, development bool) ([]document, error) {
	values := guidance{
		Development: development, Command: shellQuote(request.Executable),
		Scope:   "--project " + shellQuote(request.Directory),
		Records: request.Records, Local: request.Local, Tracker: request.Tracker,
	}
	if development {
		values.Command, values.Scope = "<temporary-ctx>", "--project <absolute-repository-directory>"
	}
	files := []struct{ source, target string }{
		{"task-context.md", filepath.Join(request.Skills, "task-context", "SKILL.md")},
		{"recovery-notes.md", filepath.Join(request.Skills, "recovery-notes", "SKILL.md")},
		{"recovery-profile.md", filepath.Join(request.Skills, "recovery-notes", "PROFILE.md")},
		{"acceptance.md", filepath.Join(request.Docs, "acceptance.md")},
		{"tracker.md", filepath.Join(request.Docs, "tracker.md")},
	}
	var documents []document
	for _, file := range files {
		link := func(target string) string {
			rel, _ := filepath.Rel(filepath.Dir(file.target), target)
			return (&url.URL{Path: filepath.ToSlash(rel)}).String()
		}
		values.AcceptanceLink = link(filepath.Join(request.Docs, "acceptance.md"))
		values.TaskContextLink = link(filepath.Join(request.Skills, "task-context", "SKILL.md"))
		values.TrackerLink = link(filepath.Join(request.Docs, "tracker.md"))
		values.SnapshotLink = values.TrackerLink
		if development {
			values.TrackerLink = link("docs/agents/issue-tracker.md")
			values.SnapshotLink = link("docs/tracker-snapshots.md")
		}
		tmpl, err := template.ParseFS(templates, "templates/"+file.source)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := tmpl.Execute(&out, values); err != nil {
			return nil, fmt.Errorf("render %s: %w", file.source, err)
		}
		documents = append(documents, document{file.target, out.Bytes()})
	}
	return documents, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
