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
