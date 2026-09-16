package taskcontext

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

type document struct {
	metadata map[string]any
	body     []byte
}

func parseDocument(source []byte) (document, error) {
	first, rest, _ := bytes.Cut(source, []byte("\n"))
	if string(bytes.TrimSuffix(first, []byte("\r"))) != "---" {
		return document{body: source}, nil
	}
	offset := len(source) - len(rest)
	for len(rest) > 0 {
		line, next, hasNewline := bytes.Cut(rest, []byte("\n"))
		if string(bytes.TrimSuffix(line, []byte("\r"))) == "---" {
			metadata := source[len(first)+1 : offset]
			parsed, err := parser.ParseBytes(metadata, 0)
			if err != nil {
				return document{}, fmt.Errorf("parse frontmatter: %w", err)
			}
			if len(parsed.Docs) != 1 || parsed.Docs[0].Body == nil {
				return document{}, errors.New("frontmatter must be a YAML mapping")
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
				return document{}, errors.New("frontmatter must be a YAML mapping")
			}
			var fields map[string]any
			if err := yaml.NodeToValue(node, &fields); err != nil {
				return document{}, fmt.Errorf("parse frontmatter: %w", err)
			}
			return document{metadata: fields, body: next}, nil
		}
		offset += len(line)
		if hasNewline {
			offset++
		}
		rest = next
	}
	return document{}, errors.New("unterminated YAML frontmatter")
}
