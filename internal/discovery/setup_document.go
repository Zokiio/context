package discovery

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// setupDocument edits only setup-owned fields. Unknown values stay YAML nodes,
// so tags and scalar types survive instead of passing through a generic encoder.
// Aliases become independent values before editing; changing one registration
// must not also change another declaration that referenced its original value.
func setupDocument(config *Config, selected map[string]any) (string, error) {
	if len(config.Raw) == 0 {
		metadata, err := yaml.Marshal(config.Metadata)
		if err != nil {
			return "", err
		}
		return "---\n" + string(metadata) + "---\n" + string(config.Body), nil
	}
	header := config.Raw[:len(config.Raw)-len(config.Body)]
	start := bytes.IndexByte(header, '\n') + 1
	end := bytes.LastIndex(header, []byte("\n---")) + 1
	file, err := parser.ParseBytes(header[start:end], parser.ParseComments)
	if err != nil {
		return "", err
	}
	root, err := detachSetupAliases(file.Docs[0].Body, map[string]*string{})
	if err != nil {
		return "", err
	}
	file.Docs[0].Body = root
	managed := map[string]any{"records": selected["records"], "allowSources": selected["allowSources"]}
	if config.Profile == SharedConfig {
		project := setupYAMLField(root, "project")
		if project == nil {
			if err := mergeSetupFields(&file.Docs[0].Body, map[string]any{"project": managed}); err != nil {
				return "", err
			}
		} else if err := mergeSetupFields(&project.Value, managed); err != nil {
			return "", err
		}
	} else {
		for _, key := range []string{"key", "directory", "alias"} {
			if value, present := selected[key]; present {
				managed[key] = value
			}
		}
		projects := setupYAMLField(root, "projects")
		if projects == nil {
			if err := mergeSetupFields(&file.Docs[0].Body, map[string]any{"projects": []any{managed}}); err != nil {
				return "", err
			}
		} else {
			sequence, ok := setupUntagged(projects.Value).(*ast.SequenceNode)
			if !ok {
				return "", fmt.Errorf("projects must remain a YAML sequence")
			}
			index := len(config.Projects)
			for i, project := range config.Projects {
				if project.Key == selected["key"] {
					index = i
					break
				}
			}
			if index < len(sequence.Values) {
				if err := mergeSetupFields(&sequence.Values[index], managed); err != nil {
					return "", err
				}
			} else {
				addition, err := yaml.ValueToNode([]any{managed}, yaml.Flow(true))
				if err != nil {
					return "", err
				}
				sequence.Merge(addition.(*ast.SequenceNode))
			}
		}
	}
	return "---\n" + file.String() + "---\n" + string(config.Body), nil
}

func setupUntagged(node ast.Node) ast.Node {
	for {
		tag, ok := node.(*ast.TagNode)
		if !ok {
			return node
		}
		node = tag.Value
	}
}

func mergeSetupFields(slot *ast.Node, fields map[string]any) error {
	if tag, ok := (*slot).(*ast.TagNode); ok {
		return mergeSetupFields(&tag.Value, fields)
	}
	if entry, ok := (*slot).(*ast.MappingValueNode); ok {
		*slot = ast.Mapping(entry.Key.GetToken(), entry.IsFlowStyle, entry)
	}
	mapping, ok := (*slot).(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("selected binding must remain a YAML mapping")
	}
	updates, err := yaml.ValueToNode(fields, yaml.Flow(true))
	if err != nil {
		return err
	}
	// The decoder applies merged fields in document order. Append the managed
	// fields after those merges so the selected values remain effective.
	mapping.Values = slices.DeleteFunc(mapping.Values, func(entry *ast.MappingValueNode) bool {
		if entry.Key.IsMergeKey() {
			return false
		}
		var key string
		if yaml.NodeToValue(entry.Key, &key) != nil {
			return false
		}
		_, managed := fields[key]
		return managed
	})
	mapping.Merge(updates.(*ast.MappingNode))
	return nil
}

// Follow merge fields in the same document order as the configuration decoder.
func setupYAMLField(node ast.Node, key string) *ast.MappingValueNode {
	switch node := node.(type) {
	case *ast.TagNode:
		return setupYAMLField(node.Value, key)
	case *ast.SequenceNode:
		for i := len(node.Values) - 1; i >= 0; i-- {
			if field := setupYAMLField(node.Values[i], key); field != nil {
				return field
			}
		}
	case ast.MapNode:
		var found *ast.MappingValueNode
		entries := node.MapRange()
		for entries.Next() {
			entry := entries.KeyValue()
			if entry.Key.IsMergeKey() {
				if field := setupYAMLField(entry.Value, key); field != nil {
					found = field
				}
				continue
			}
			var value any
			if yaml.NodeToValue(entry.Key, &value) == nil && value == key {
				found = entry
			}
		}
		return found
	}
	return nil
}

func detachSetupAliases(node ast.Node, anchors map[string]*string) (ast.Node, error) {
	switch value := node.(type) {
	case *ast.AnchorNode:
		name := value.Name.GetToken().Value
		anchors[name] = nil
		detached, err := detachSetupAliases(value.Value, anchors)
		if err != nil {
			return nil, err
		}
		source := detached.String()
		anchors[name] = &source
		return detached, nil
	case *ast.AliasNode:
		name := value.Value.GetToken().Value
		source, present := anchors[name]
		if !present || source == nil {
			return nil, fmt.Errorf("cannot preserve recursive or unresolved YAML alias %q during setup", name)
		}
		// A mapping wrapper also represents an implicit null, whose String is
		// empty, and retains scalar tags and multiline values while copying.
		copy, err := parser.ParseBytes([]byte("value:\n  "+strings.ReplaceAll(*source, "\n", "\n  ")), parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("copy YAML alias %q: %w", name, err)
		}
		return setupYAMLField(copy.Docs[0].Body, "value").Value, nil
	case *ast.MappingNode:
		for _, entry := range value.Values {
			if _, err := detachSetupAliases(entry, anchors); err != nil {
				return nil, err
			}
		}
	case *ast.MappingValueNode:
		key, err := detachSetupAliases(value.Key, anchors)
		if err != nil {
			return nil, err
		}
		mapKey, ok := key.(ast.MapKeyNode)
		if !ok {
			return nil, fmt.Errorf("cannot preserve a compound YAML alias key during setup")
		}
		if key != value.Key {
			key.AddColumn(value.Key.GetToken().Position.Column - key.GetToken().Position.Column)
		}
		value.Key = mapKey
		detached, err := detachSetupAliases(value.Value, anchors)
		if err != nil {
			return nil, err
		}
		if detached != value.Value {
			if err := value.Replace(detached); err != nil {
				return nil, err
			}
		}
	case *ast.SequenceNode:
		for i, entry := range value.Values {
			detached, err := detachSetupAliases(entry, anchors)
			if err != nil {
				return nil, err
			}
			if detached != entry {
				if err := value.Replace(i, detached); err != nil {
					return nil, err
				}
			}
		}
	case *ast.TagNode:
		detached, err := detachSetupAliases(value.Value, anchors)
		if err != nil {
			return nil, err
		}
		value.Value = detached
	case *ast.MappingKeyNode:
		detached, err := detachSetupAliases(value.Value, anchors)
		if err != nil {
			return nil, err
		}
		value.Value = detached
	}
	return node, nil
}

// Compare decoded metadata, where YAML's NaN represents the same value even
// though Go equality deliberately considers it unequal to itself.
func sameSetupMetadata(left, right any) bool {
	switch left := left.(type) {
	case map[string]any:
		right, ok := right.(map[string]any)
		if !ok || len(left) != len(right) {
			return false
		}
		for key, value := range left {
			other, present := right[key]
			if !present || !sameSetupMetadata(value, other) {
				return false
			}
		}
		return true
	case []any:
		right, ok := right.([]any)
		return ok && slices.EqualFunc(left, right, sameSetupMetadata)
	case float64:
		right, ok := right.(float64)
		return ok && (left == right || math.IsNaN(left) && math.IsNaN(right))
	default:
		return reflect.DeepEqual(left, right)
	}
}
