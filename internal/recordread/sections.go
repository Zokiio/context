package recordread

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/util"
)

type Relationship struct{ Kind, Link, Destination string }

// Section preserves presence, ordering and source spans independently of meaning.
type Section struct {
	Name                     string
	Start, ContentStart, End int
	Content                  string
	Text                     string
	Links                    []Relationship
	Complete                 bool
}

type SectionsResult struct {
	Sections        []Section
	Links           []Relationship
	Diagnostics     []Diagnostic
	InvalidSections map[string]bool
}

var referenceLabel = regexp.MustCompile(`\[((?:\\[\s\S]|[^\[\]\\])+)\]`)

// Sections selects exact named level-two sections and their Markdown links.
// The values in names identify relationship kinds for the caller.
func Sections(body []byte, path string, names map[string]string) SectionsResult {
	result := SectionsResult{Sections: []Section{}, Links: []Relationship{}, Diagnostics: []Diagnostic{}, InvalidSections: map[string]bool{}}
	context := parser.NewContext()
	markdown := parser.New()
	tree := markdown.Parse(body, parser.WithContext(context))
	ordinaryText := make([]int, len(body))
	active := -1
	sectionText := []*strings.Builder{}
	finish := func(end int) {
		if active >= 0 {
			section := &result.Sections[active]
			section.End = end
			section.Content = string(body[section.ContentStart:end])
			section.Text = sectionText[active].String()
		}
	}
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := node.(*ast.Heading); ok && heading.Level <= 2 {
			start := lineStart(body, heading.Pos())
			finish(start)
			active = -1
			name := PlainText(heading, body)
			if _, selected := names[name]; heading.Level == 2 && selected {
				contentStart := nextLine(body, start)
				if heading.HeadingKind == ast.HeadingKindSetext {
					contentStart = nextLine(body, contentStart)
				}
				active = len(result.Sections)
				result.Sections = append(result.Sections, Section{Name: name, Start: start, ContentStart: contentStart, End: len(body), Links: []Relationship{}, Complete: true})
				sectionText = append(sectionText, &strings.Builder{})
			}
			return ast.WalkSkipChildren, nil
		}
		switch node.(type) {
		case *ast.Image, *ast.CodeSpan, *ast.CodeBlock:
			if active >= 0 {
				sectionText[active].WriteString(" [non-prose content] ")
			}
			return ast.WalkSkipChildren, nil
		}
		if active < 0 {
			return ast.WalkContinue, nil
		}
		section := &result.Sections[active]
		var link *Relationship
		switch node := node.(type) {
		case *ast.Paragraph:
			sectionText[active].WriteString(" ")
		case *ast.AutoLink:
			link = &Relationship{Kind: names[section.Name], Link: string(node.Destination.Bytes(body)), Destination: node.Destination.Value(body)}
		case *ast.Link:
			link = &Relationship{Kind: names[section.Name], Link: string(node.Destination.Bytes(body)), Destination: node.Destination.Value(body)}
		case *ast.Text:
			sectionText[active].WriteString(node.Value.Value(body))
			if !node.Value.IsOwned() {
				position := node.Value.Index()
				for i := position.Start; i < position.Stop; i++ {
					ordinaryText[i] = active + 1
				}
			}
		}
		if link != nil {
			section.Links = append(section.Links, *link)
			result.Links = append(result.Links, *link)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	finish(len(body))
	// Missing full or collapsed reference definitions normally become plain text.
	// Seed only those forms on a second parse, retaining the original text selection.
	for _, match := range referenceLabel.FindAllSubmatchIndex(body, -1) {
		full := match[0] > 0 && body[match[0]-1] == ']'
		collapsed := bytes.HasPrefix(body[match[1]:], []byte("[]"))
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
			raw := string(link.Destination.Bytes(body))
			position := link.Pos()
			if strings.HasPrefix(raw, "\x00") && link.Reference != nil && link.Reference.ReferenceLinkKind != ast.ReferenceLinkKindShortcut && position >= 0 && position < len(ordinaryText) && ordinaryText[position] > 0 {
				section := &result.Sections[ordinaryText[position]-1]
				section.Complete = false
				result.InvalidSections[section.Name] = true
				label := strings.TrimPrefix(raw, "\x00")
				result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "unresolved_reference", Severity: "error", Message: "unresolved Markdown reference: " + label, Path: path, From: path, Link: "[" + label + "]"})
			}
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return result
}

func PlainText(node ast.Node, source []byte) string {
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

func lineStart(source []byte, position int) int {
	if position < 0 {
		return 0
	}
	if position > len(source) {
		position = len(source)
	}
	return bytes.LastIndexByte(source[:position], '\n') + 1
}

func nextLine(source []byte, position int) int {
	if next := bytes.IndexByte(source[position:], '\n'); next >= 0 {
		return position + next + 1
	}
	return len(source)
}
