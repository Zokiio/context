package taskcontext

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/util"
)

type relationship struct {
	kind        string
	link        string
	destination string
}

// Missing definitions normally become plain text. The normal AST owns selection.
// A second parse seeds missing definitions only to diagnose explicit references.
// Diagnostics must start in ordinary text from the normal AST, so added definitions
// cannot turn nested labels inside existing links or images into relationships.
var referenceLabel = regexp.MustCompile(`\[((?:\\[\s\S]|[^\[\]\\])+)\]`)

func extractRelationships(body []byte, path string) ([]relationship, []Diagnostic) {
	context := parser.NewContext()
	markdown := parser.New()
	tree := markdown.Parse(body, parser.WithContext(context))
	links := []relationship{}
	diagnostics := []Diagnostic{}
	ordinaryText := make([]bool, len(body))
	kind := ""
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := node.(*ast.Heading); ok && heading.Level <= 2 {
			kind = ""
			if heading.Level == 2 {
				switch plainText(heading, body) {
				case "Spec":
					kind = "spec"
				case "Blocked by":
					kind = "blocked_by"
				case "Context":
					kind = "context"
				}
			}
			return ast.WalkSkipChildren, nil
		}
		switch node.(type) {
		case *ast.Image, *ast.CodeSpan, *ast.CodeBlock:
			return ast.WalkSkipChildren, nil
		}
		if kind == "" {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.AutoLink:
			links = append(links, relationship{kind: kind, link: string(node.Destination.Bytes(body)), destination: node.Destination.Value(body)})
			return ast.WalkSkipChildren, nil
		case *ast.Link:
			links = append(links, relationship{kind: kind, link: string(node.Destination.Bytes(body)), destination: node.Destination.Value(body)})
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if !node.Value.IsOwned() {
				position := node.Value.Index()
				for i := position.Start; i < position.Stop; i++ {
					ordinaryText[i] = true
				}
			}
		}
		return ast.WalkContinue, nil
	})
	for _, match := range referenceLabel.FindAllSubmatchIndex(body, -1) {
		// Seed only explicit full or collapsed reference labels. Seeding an ordinary
		// nested [word] would create a shortcut link and suppress its enclosing label.
		full := match[0] > 0 && body[match[0]-1] == ']'
		collapsed := strings.HasPrefix(string(body[match[1]:]), "[]")
		if !full && !collapsed {
			continue
		}
		label := util.ToLinkReference(body[match[2]:match[3]])
		if _, exists := context.LinkDefinition(label); !exists {
			context.AddLinkDefinition(parser.NewLinkDefinition([]byte(label), []byte("\x00"+label), nil))
		}
	}

	diagnosticTree := markdown.Parse(body, parser.WithContext(context))
	_ = ast.Walk(diagnosticTree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.(type) {
		case *ast.Image, *ast.CodeSpan, *ast.CodeBlock:
			return ast.WalkSkipChildren, nil
		}
		if link, ok := node.(*ast.Link); ok {
			rawDestination := string(link.Destination.Bytes(body))
			position := link.Pos()
			if strings.HasPrefix(rawDestination, "\x00") && link.Reference != nil && link.Reference.ReferenceLinkKind != ast.ReferenceLinkKindShortcut && position >= 0 && position < len(ordinaryText) && ordinaryText[position] {
				label := strings.TrimPrefix(rawDestination, "\x00")
				diagnostics = append(diagnostics, Diagnostic{Code: "unresolved_reference", Severity: "error", Message: "unresolved Markdown reference: " + label, Path: path, From: path, Link: "[" + label + "]"})
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return links, diagnostics
}

func plainText(node ast.Node, source []byte) string {
	var value strings.Builder
	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if text, ok := node.(*ast.Text); ok {
				value.WriteString(text.Value.Value(source))
			}
		}
		return ast.WalkContinue, nil
	})
	return value.String()
}
