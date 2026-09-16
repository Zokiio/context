// Package discovery locates project and workspace scope outside the record readers.
package discovery

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/recordread"
)

type ConfigProfile string

const (
	SharedConfig   ConfigProfile = "shared"
	PersonalConfig ConfigProfile = "personal"
)

// Origin identifies a declaration in the encountered configuration file, before
// resolving any symlink in that filename.
type Origin struct {
	Path  string
	Field string
}

type Config struct {
	Path       string
	Profile    ConfigProfile
	Raw        []byte
	Metadata   map[string]any
	Body       []byte
	Project    *ProjectBinding
	Workspace  *WorkspaceDeclaration
	Projects   []ProjectRegistration
	Workspaces []WorkspaceRegistration
}

type ProjectBinding struct {
	Origin       Origin
	Records      PathValue
	AllowSources []PathValue
}

type ProjectRegistration struct {
	ProjectBinding
	Key       string
	Alias     string
	Directory PathValue
}

type WorkspaceDeclaration struct {
	Origin  Origin
	ID      string
	Title   string
	Members []Member
}

type WorkspaceRegistration struct {
	WorkspaceDeclaration
	Key       string
	Alias     string
	Directory *PathValue
}

type Member struct {
	Origin       Origin
	Key          string
	Title        string
	Records      PathValue
	Directory    *PathValue
	AllowSources []PathValue
}

// ConfigError gives callers a field-specific failure without discarding the
// authored value. File and YAML failures retain their underlying error.
type ConfigError struct {
	Path    string
	Field   string
	Value   any
	Problem string
	Err     error
}

func (e *ConfigError) Error() string {
	message := fmt.Sprintf("configuration %s field %s (%v): %s", e.Path, e.Field, e.Value, e.Problem)
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message + "; correct this field in the configuration"
}

func (e *ConfigError) Unwrap() error { return e.Err }

func ReadConfig(path string, profile ConfigProfile) (*Config, error) {
	if err := checkRegularFile(path); err != nil {
		return nil, fmt.Errorf("read configuration %s: %w", path, err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration %s: %w", path, err)
	}
	return ParseConfig(path, source, profile)
}

// ParseConfig validates structure but does not require registered targets to be
// available. Metadata, including unknown nested fields, and body bytes are kept
// for setup to preserve when it changes a binding.
func ParseConfig(path string, source []byte, profile ConfigProfile) (*Config, error) {
	absolute, err := absolutePath(path)
	if err != nil {
		return nil, fmt.Errorf("resolve configuration filename %s: %w", path, err)
	}
	p := configParser{path: absolute}
	if profile != SharedConfig && profile != PersonalConfig {
		return nil, p.fail("profile", profile, "must be shared or personal")
	}
	if !utf8.Valid(source) {
		return nil, p.fail("document", nil, "must contain UTF-8 Markdown")
	}
	raw := append([]byte(nil), source...)
	document, err := recordread.ParseDocument(raw)
	if err != nil {
		return nil, &ConfigError{Path: absolute, Field: "frontmatter", Problem: "invalid YAML frontmatter", Err: err}
	}
	if document.Metadata == nil {
		return nil, p.fail("frontmatter", nil, "requires YAML frontmatter")
	}
	fields := document.Metadata
	kind, err := p.string(fields, "", "type", true)
	if err != nil {
		return nil, err
	}
	if kind != "ContextConfig" {
		return nil, p.fail("type", kind, "must be ContextConfig")
	}
	if err := p.version(fields["version"]); err != nil {
		return nil, err
	}
	config := &Config{Path: absolute, Profile: profile, Raw: raw, Metadata: fields, Body: document.Body,
		Projects: []ProjectRegistration{}, Workspaces: []WorkspaceRegistration{}}
	if profile == SharedConfig {
		for _, field := range []string{"projects", "workspaces"} {
			if value, present := fields[field]; present {
				return nil, p.fail(field, value, "personal fields cannot appear in shared configuration")
			}
		}
		if value, present := fields["project"]; present {
			mapping, err := p.mapping(value, "project")
			if err != nil {
				return nil, err
			}
			binding, err := p.project(mapping, "project")
			if err != nil {
				return nil, err
			}
			config.Project = &binding
		}
		if value, present := fields["workspace"]; present {
			mapping, err := p.mapping(value, "workspace")
			if err != nil {
				return nil, err
			}
			workspace, err := p.workspace(mapping, "workspace")
			if err != nil {
				return nil, err
			}
			config.Workspace = &workspace
		}
		if config.Project == nil && config.Workspace == nil {
			return nil, p.fail("project/workspace", nil, "shared configuration requires a project or workspace declaration")
		}
		return config, nil
	}
	for _, field := range []string{"project", "workspace"} {
		if value, present := fields[field]; present {
			return nil, p.fail(field, value, "shared fields cannot appear in personal configuration")
		}
	}
	if err := p.registrations(config); err != nil {
		return nil, err
	}
	return config, nil
}

type configParser struct{ path string }

func (p configParser) fail(field string, value any, problem string) error {
	return &ConfigError{Path: p.path, Field: field, Value: value, Problem: problem}
}

func (p configParser) version(value any) error {
	valid, integer := false, true
	switch value := value.(type) {
	case int:
		valid = value == 1
	case int64:
		valid = value == 1
	case uint64:
		valid = value == 1
	default:
		integer = false
	}
	if !integer {
		return p.fail("version", value, "must be integer 1")
	}
	if !valid {
		return p.fail("version", value, "unsupported version; this reader supports version 1")
	}
	return nil
}

func fieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

func (p configParser) string(fields map[string]any, parent, name string, required bool) (string, error) {
	value, present := fields[name]
	if !present && !required {
		return "", nil
	}
	text, ok := value.(string)
	if !ok || required && strings.TrimSpace(text) == "" {
		problem := "must be a string"
		if required {
			problem = "must be a nonempty string"
		}
		return "", p.fail(fieldPath(parent, name), value, problem)
	}
	return text, nil
}

func (p configParser) mapping(value any, field string) (map[string]any, error) {
	mapping, ok := value.(map[string]any)
	if !ok {
		return nil, p.fail(field, value, "must be a mapping")
	}
	return mapping, nil
}

func (p configParser) array(fields map[string]any, parent, name string, required bool) ([]any, error) {
	value, present := fields[name]
	if !present && !required {
		return []any{}, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, p.fail(fieldPath(parent, name), value, "must be an array")
	}
	return items, nil
}

func (p configParser) pathValue(fields map[string]any, parent, name string) (PathValue, error) {
	value, err := p.string(fields, parent, name, true)
	if err != nil {
		return PathValue{}, err
	}
	return savedPath(p.path, fieldPath(parent, name), value), nil
}

func (p configParser) optionalPath(fields map[string]any, parent, name string) (*PathValue, error) {
	if _, present := fields[name]; !present {
		return nil, nil
	}
	path, err := p.pathValue(fields, parent, name)
	if err != nil {
		return nil, err
	}
	return &path, nil
}

func (p configParser) allowed(fields map[string]any, parent string) ([]PathValue, error) {
	items, err := p.array(fields, parent, "allowSources", false)
	if err != nil {
		return nil, err
	}
	paths := make([]PathValue, 0, len(items))
	for i, item := range items {
		field := fmt.Sprintf("%s.allowSources[%d]", parent, i)
		value, ok := item.(string)
		if !ok || strings.TrimSpace(value) == "" {
			return nil, p.fail(field, item, "must be a nonempty path string")
		}
		paths = append(paths, savedPath(p.path, field, value))
	}
	return paths, nil
}

func (p configParser) project(fields map[string]any, field string) (ProjectBinding, error) {
	binding := ProjectBinding{Origin: Origin{Path: p.path, Field: field}}
	var err error
	if binding.Records, err = p.pathValue(fields, field, "records"); err != nil {
		return binding, err
	}
	binding.AllowSources, err = p.allowed(fields, field)
	return binding, err
}

func (p configParser) workspace(fields map[string]any, field string) (WorkspaceDeclaration, error) {
	workspace := WorkspaceDeclaration{Origin: Origin{Path: p.path, Field: field}, Members: []Member{}}
	var err error
	if workspace.ID, err = p.string(fields, field, "id", true); err != nil {
		return workspace, err
	}
	if workspace.Title, err = p.string(fields, field, "title", true); err != nil {
		return workspace, err
	}
	items, err := p.array(fields, field, "members", true)
	if err != nil {
		return workspace, err
	}
	keys := map[string]string{}
	for i, item := range items {
		memberField := fmt.Sprintf("%s.members[%d]", field, i)
		mapping, err := p.mapping(item, memberField)
		if err != nil {
			return workspace, err
		}
		member := Member{Origin: Origin{Path: p.path, Field: memberField}}
		if member.Key, err = p.string(mapping, memberField, "key", true); err != nil {
			return workspace, err
		}
		if err := p.unique(keys, member.Key, memberField+".key"); err != nil {
			return workspace, err
		}
		if member.Title, err = p.string(mapping, memberField, "title", false); err != nil {
			return workspace, err
		}
		if member.Records, err = p.pathValue(mapping, memberField, "records"); err != nil {
			return workspace, err
		}
		if member.Directory, err = p.optionalPath(mapping, memberField, "directory"); err != nil {
			return workspace, err
		}
		if member.AllowSources, err = p.allowed(mapping, memberField); err != nil {
			return workspace, err
		}
		workspace.Members = append(workspace.Members, member)
	}
	return workspace, nil
}

func (p configParser) unique(seen map[string]string, value, field string) error {
	if previous, duplicate := seen[value]; duplicate {
		return p.fail(field, value, "duplicates "+previous)
	}
	seen[value] = field
	return nil
}

func (p configParser) registrations(config *Config) error {
	projects, err := p.array(config.Metadata, "", "projects", false)
	if err != nil {
		return err
	}
	projectKeys, projectAliases := map[string]string{}, map[string]string{}
	for i, item := range projects {
		field := fmt.Sprintf("projects[%d]", i)
		mapping, err := p.mapping(item, field)
		if err != nil {
			return err
		}
		registration := ProjectRegistration{}
		if registration.Key, registration.Alias, err = p.entryIdentity(mapping, field, projectKeys, projectAliases); err != nil {
			return err
		}
		if registration.ProjectBinding, err = p.project(mapping, field); err != nil {
			return err
		}
		if registration.Directory, err = p.pathValue(mapping, field, "directory"); err != nil {
			return err
		}
		config.Projects = append(config.Projects, registration)
	}
	workspaces, err := p.array(config.Metadata, "", "workspaces", false)
	if err != nil {
		return err
	}
	workspaceKeys, workspaceAliases := map[string]string{}, map[string]string{}
	for i, item := range workspaces {
		field := fmt.Sprintf("workspaces[%d]", i)
		mapping, err := p.mapping(item, field)
		if err != nil {
			return err
		}
		registration := WorkspaceRegistration{}
		if registration.Key, registration.Alias, err = p.entryIdentity(mapping, field, workspaceKeys, workspaceAliases); err != nil {
			return err
		}
		if registration.WorkspaceDeclaration, err = p.workspace(mapping, field); err != nil {
			return err
		}
		if registration.Directory, err = p.optionalPath(mapping, field, "directory"); err != nil {
			return err
		}
		config.Workspaces = append(config.Workspaces, registration)
	}
	return nil
}

func (p configParser) entryIdentity(fields map[string]any, field string, keys, aliases map[string]string) (string, string, error) {
	key, err := p.string(fields, field, "key", true)
	if err != nil {
		return "", "", err
	}
	if err := p.unique(keys, key, field+".key"); err != nil {
		return "", "", err
	}
	alias, err := p.string(fields, field, "alias", false)
	if err != nil {
		return "", "", err
	}
	if _, present := fields["alias"]; present {
		if err := p.unique(aliases, alias, field+".alias"); err != nil {
			return "", "", err
		}
	}
	return key, alias, nil
}
