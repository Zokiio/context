package discoveryconfig

import (
	"fmt"
	"strings"
)

// decoder keeps the first structural error. Invalid declarations are never
// returned by Read, and path resolution stops once a structural error is known.
type decoder struct {
	file string
	err  *FieldError
}

func (d *decoder) fail(field, message string) {
	if d.err == nil {
		d.err = &FieldError{File: d.file, Field: field, Message: message}
	}
}

func fieldPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func (d *decoder) stringValue(fields map[string]any, key, prefix string, required bool) string {
	field := fieldPath(prefix, key)
	value, present := fields[key]
	if !present {
		if required {
			d.fail(field, "required field is missing")
		}
		return ""
	}
	text, ok := value.(string)
	if !ok {
		d.fail(field, "must be a string")
		return ""
	}
	if required && strings.TrimSpace(text) == "" {
		d.fail(field, "must be a nonempty string")
	}
	return text
}

func (d *decoder) optionalString(fields map[string]any, key, prefix string) *string {
	if _, present := fields[key]; !present {
		return nil
	}
	value := d.stringValue(fields, key, prefix, false)
	return &value
}

func (d *decoder) mapping(value any, field string) map[string]any {
	fields, ok := value.(map[string]any)
	if !ok {
		d.fail(field, "must be a mapping")
	}
	return fields
}

func (d *decoder) sequence(fields map[string]any, key, prefix string, required bool) []any {
	field := fieldPath(prefix, key)
	value, present := fields[key]
	if !present {
		if required {
			d.fail(field, "required field is missing")
		}
		return []any{}
	}
	items, ok := value.([]any)
	if !ok {
		d.fail(field, "must be an array")
		return []any{}
	}
	return items
}

func (d *decoder) decode(doc *Document, profile Profile) {
	fields := doc.Metadata
	if d.stringValue(fields, "type", "", true) != "ContextConfig" {
		d.fail("type", "must be ContextConfig")
	}
	// YAML integer scalars retain their numeric type. A float or quoted "1"
	// must not acquire the meaning of the version-1 integer.
	versionOK := false
	switch version := fields["version"].(type) {
	case uint64:
		versionOK = version == 1
	case int64:
		versionOK = version == 1
	}
	if !versionOK {
		d.fail("version", "must be the supported integer version 1")
	}
	if profile == SharedProfile {
		for _, key := range []string{"projects", "workspaces"} {
			if _, present := fields[key]; present {
				d.fail(key, "personal fields are not allowed in shared configuration")
			}
		}
		shared := &SharedConfig{}
		if value, present := fields["project"]; present {
			binding := d.project(d.mapping(value, "project"), "project")
			shared.Project = &binding
		}
		if value, present := fields["workspace"]; present {
			workspace := d.workspace(d.mapping(value, "workspace"), "workspace")
			shared.Workspace = &workspace
		}
		if shared.Project == nil && shared.Workspace == nil {
			d.fail("$", "shared configuration requires project or workspace")
		}
		doc.Shared = shared
		return
	}
	for _, key := range []string{"project", "workspace"} {
		if _, present := fields[key]; present {
			d.fail(key, "shared fields are not allowed in personal configuration")
		}
	}
	personal := &PersonalConfig{Projects: []ProjectRegistration{}, Workspaces: []WorkspaceRegistration{}}
	keys, aliases := map[string]string{}, map[string]string{}
	for i, value := range d.sequence(fields, "projects", "", false) {
		prefix := fmt.Sprintf("projects[%d]", i)
		entry := d.mapping(value, prefix)
		project := ProjectRegistration{
			Key:       d.stringValue(entry, "key", prefix, true),
			Alias:     d.optionalString(entry, "alias", prefix),
			Directory: d.path(entry, "directory", prefix),
		}
		project.ProjectBinding = d.project(entry, prefix)
		d.unique(keys, project.Key, prefix+".key", "key")
		if project.Alias != nil {
			d.unique(aliases, *project.Alias, prefix+".alias", "alias")
		}
		personal.Projects = append(personal.Projects, project)
	}
	keys, aliases = map[string]string{}, map[string]string{}
	for i, value := range d.sequence(fields, "workspaces", "", false) {
		prefix := fmt.Sprintf("workspaces[%d]", i)
		entry := d.mapping(value, prefix)
		workspace := WorkspaceRegistration{
			Key:       d.stringValue(entry, "key", prefix, true),
			Alias:     d.optionalString(entry, "alias", prefix),
			Directory: d.optionalPath(entry, "directory", prefix),
		}
		workspace.WorkspaceDeclaration = d.workspace(entry, prefix)
		d.unique(keys, workspace.Key, prefix+".key", "key")
		if workspace.Alias != nil {
			d.unique(aliases, *workspace.Alias, prefix+".alias", "alias")
		}
		personal.Workspaces = append(personal.Workspaces, workspace)
	}
	doc.Personal = personal
}

func (d *decoder) project(fields map[string]any, prefix string) ProjectBinding {
	project := ProjectBinding{Records: d.path(fields, "records", prefix), AllowSources: []TargetPath{}}
	for i, value := range d.sequence(fields, "allowSources", prefix, false) {
		field := fmt.Sprintf("%s.allowSources[%d]", prefix, i)
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			d.fail(field, "must be a nonempty path string")
		}
		project.AllowSources = append(project.AllowSources, d.target(text, field))
	}
	return project
}

func (d *decoder) workspace(fields map[string]any, prefix string) WorkspaceDeclaration {
	workspace := WorkspaceDeclaration{
		ID:      d.stringValue(fields, "id", prefix, true),
		Title:   d.stringValue(fields, "title", prefix, true),
		Members: []WorkspaceMember{},
	}
	keys := map[string]string{}
	for i, value := range d.sequence(fields, "members", prefix, true) {
		field := fmt.Sprintf("%s.members[%d]", prefix, i)
		entry := d.mapping(value, field)
		member := WorkspaceMember{
			Key:       d.stringValue(entry, "key", field, true),
			Title:     d.optionalString(entry, "title", field),
			Directory: d.optionalPath(entry, "directory", field),
		}
		member.ProjectBinding = d.project(entry, field)
		d.unique(keys, member.Key, field+".key", "member key")
		workspace.Members = append(workspace.Members, member)
	}
	return workspace
}

func (d *decoder) unique(seen map[string]string, value, field, label string) {
	if first, exists := seen[value]; exists {
		d.fail(field, fmt.Sprintf("duplicate %s %q; first declared at %s", label, value, first))
	} else {
		seen[value] = field
	}
}
