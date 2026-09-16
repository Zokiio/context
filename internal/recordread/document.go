package recordread

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

type Document struct {
	Metadata map[string]any
	Body     []byte
}

func ParseDocument(source []byte) (Document, error) {
	first, rest, _ := bytes.Cut(source, []byte("\n"))
	if string(bytes.TrimSuffix(first, []byte("\r"))) != "---" {
		return Document{Body: source}, nil
	}
	offset := len(source) - len(rest)
	for len(rest) > 0 {
		line, next, hasNewline := bytes.Cut(rest, []byte("\n"))
		if string(bytes.TrimSuffix(line, []byte("\r"))) == "---" {
			metadata := source[len(first)+1 : offset]
			parsed, err := parser.ParseBytes(metadata, 0)
			if err != nil {
				return Document{}, fmt.Errorf("parse frontmatter: %w", err)
			}
			if len(parsed.Docs) != 1 || parsed.Docs[0].Body == nil {
				return Document{}, errors.New("frontmatter must be a YAML mapping")
			}
			node := parsed.Docs[0].Body
			for {
				switch wrapped := node.(type) {
				case *ast.AnchorNode:
					node = wrapped.Value
				case *ast.TagNode:
					node = wrapped.Value
				default:
					goto unwrapped
				}
			}
		unwrapped:
			switch node.(type) {
			case *ast.MappingNode, *ast.MappingValueNode:
			default:
				return Document{}, errors.New("frontmatter must be a YAML mapping")
			}
			if err := validateMappingKeys(node); err != nil {
				return Document{}, fmt.Errorf("parse frontmatter: %w", err)
			}
			var fields map[string]any
			if err := yaml.NodeToValue(node, &fields); err != nil {
				return Document{}, fmt.Errorf("parse frontmatter: %w", err)
			}
			return Document{Metadata: fields, Body: next}, nil
		}
		offset += len(line)
		if hasNewline {
			offset++
		}
		rest = next
	}
	return Document{}, errors.New("unterminated YAML frontmatter")
}

// Nested maps decoded into any do not receive go-yaml's duplicate-key check.
// Check their authored keys before decoding can overwrite an aliased key. Merge
// entries remain separate, since explicit keys may override inherited values.
func validateMappingKeys(node ast.Node) error {
	validator := mappingKeyValidator{decoder: yaml.NewDecoder(bytes.NewReader(nil))}
	return validator.walk(node)
}

type mappingKeyValidator struct {
	decoder *yaml.Decoder
}

func (v *mappingKeyValidator) walk(node ast.Node) error {
	switch node := node.(type) {
	case *ast.MappingNode:
		return v.mapping(node.Values)
	case *ast.MappingValueNode:
		return v.mapping([]*ast.MappingValueNode{node})
	case *ast.SequenceNode:
		for _, value := range node.Values {
			if err := v.walk(value); err != nil {
				return err
			}
		}
	case *ast.AnchorNode:
		if err := v.walk(node.Value); err != nil {
			return err
		}
		// Retain earlier anchors for later keys, including anchor redefinitions.
		// The full document decode below still owns invalid-alias diagnostics.
		var value any
		_ = v.decoder.DecodeFromNode(node, &value)
	case *ast.TagNode:
		return v.walk(node.Value)
	case *ast.MappingKeyNode:
		return v.walk(node.Value)
	}
	return nil
}

func (v *mappingKeyValidator) mapping(entries []*ast.MappingValueNode) error {
	seen := map[string]ast.MapKeyNode{}
	for _, entry := range entries {
		if err := v.walk(entry.Key); err != nil {
			return err
		}
		var value any
		if !entry.Key.IsMergeKey() && v.decoder.DecodeFromNode(entry.Key, &value) == nil {
			// Match the string keys used by the decoded metadata maps.
			key := fmt.Sprint(value)
			if value == nil {
				key = "null"
			}
			if first, duplicate := seen[key]; duplicate {
				position := first.GetToken().Position
				return &yaml.DuplicateKeyError{
					Message: fmt.Sprintf("duplicate key %q; first declared at [%d:%d]", key, position.Line, position.Column),
					Token:   entry.Key.GetToken(),
				}
			}
			seen[key] = entry.Key
		}
		if err := v.walk(entry.Value); err != nil {
			return err
		}
	}
	return nil
}
