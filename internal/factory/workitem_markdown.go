package factory

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

const descriptionStart = "<!-- waymark:description -->"
const descriptionEnd = "<!-- /waymark:description -->"

func descriptionBlock(text string) string {
	if text == "" {
		return "\n"
	}
	return "\n" + descriptionStart + "\n" + text + "\n" + descriptionEnd + "\n\n"
}
func descriptionRange(body []byte) (int, int, bool) {
	begin := bytes.Index(body, []byte(descriptionStart))
	if begin < 0 {
		return 0, 0, false
	}
	begin += len(descriptionStart)
	end := bytes.Index(body[begin:], []byte(descriptionEnd))
	if end < 0 {
		return 0, 0, false
	}
	return begin, begin + end, true
}
func validateDescription(text string) error {
	if strings.Contains(text, descriptionStart) || strings.Contains(text, descriptionEnd) {
		return fault(CodeInvalid, "description contains a reserved record delimiter", nil)
	}
	sections := recordread.Sections([]byte(text), "", map[string]string{"Acceptance criteria": "reserved", "Blocked by": "reserved", "Blocked by decisions": "reserved", "Spec": "reserved", "Context": "reserved", "Acceptance": "reserved", "Comments": "reserved"})
	if len(sections.Sections) > 0 {
		return fault(CodeInvalid, "use the dedicated field for the reserved record section: "+sections.Sections[0].Name, nil)
	}
	return nil
}
func sectionPaths(raw []byte, path, name string) []string {
	result := []string{}
	document, err := recordread.ParseDocument(raw)
	if err != nil {
		return result
	}
	for _, link := range recordread.Sections(document.Body, path, map[string]string{name: "document"}).Links {
		result = append(result, link.Destination)
	}
	return result
}
func documentLinkLines(paths []string) []string {
	if len(paths) == 0 {
		return []string{"None"}
	}
	result := []string{}
	for _, path := range paths {
		// Angle destinations preserve spaces; validation rejects line breaks and < >.
		result = append(result, "- ["+escapeLinkText(path)+"](<"+path+">)")
	}
	return result
}
func rewriteDocumentLinks(raw []byte, path, name string, paths []string) ([]byte, error) {
	if slices.Equal(sectionPaths(raw, path, name), paths) {
		return raw, nil
	}
	document, err := recordread.ParseDocument(raw)
	if err != nil {
		return nil, err
	}
	sections := recordread.Sections(document.Body, path, map[string]string{name: "document"}).Sections
	if len(sections) == 0 {
		return append(append([]byte{}, bytes.TrimRight(raw, "\n")...), "\n\n## "+name+"\n\n"+strings.Join(documentLinkLines(paths), "\n")+"\n"...), nil
	}
	return replaceSection(raw, path, name, documentLinkLines(paths))
}
func validateWorkLinks(project Project, path string, draft WorkItemDraft) error {
	reader, err := recordread.NewReader(project.BundlePath, project.AllowedSourceRoots)
	if err != nil {
		return fault(CodeInvalid, "open project source roots", err)
	}
	defer reader.Close()
	for _, links := range [][]string{draft.Spec, draft.Context} {
		for _, link := range links {
			parsed, err := url.Parse(link)
			if strings.TrimSpace(link) != link || link == "" || strings.ContainsAny(link, "\r\n<>\x00") || err != nil || parsed.IsAbs() || parsed.Host != "" {
				return fault(CodeInvalid, "spec and context links must be local Markdown paths", err)
			}
			target, diagnostic := reader.LinkPath(path, recordread.Relationship{Destination: link, Link: link})
			if diagnostic != nil {
				return fault(CodeInvalid, "invalid source link "+link+": "+diagnostic.Message, nil)
			}
			if !strings.EqualFold(filepath.Ext(target), ".md") {
				return fault(CodeInvalid, "spec and context links must name Markdown files", nil)
			}
			if _, diagnostic = reader.Read(target, recordread.DocumentSource); diagnostic != nil {
				return fault(CodeInvalid, "source link "+link+" cannot be read: "+diagnostic.Message, nil)
			}
		}
	}
	return nil
}
func setRecordBool(raw []byte, key string, value bool) ([]byte, error) {
	doc, err := recordread.ParseDocument(raw)
	if err != nil || !bytes.HasSuffix(raw, doc.Body) || len(raw) == len(doc.Body) {
		return nil, fault(CodeInvalid, "cannot update work item metadata", err)
	}
	offset := len(raw) - len(doc.Body)
	lines := strings.SplitAfter(string(raw[:offset]), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key+":") {
			lines[i] = fmt.Sprintf("%s: %t\n", key, value)
			return append([]byte(strings.Join(lines, "")), doc.Body...), nil
		}
	}
	// Insert before the closing frontmatter delimiter without reserializing any
	// unknown metadata, comments, anchors, or field ordering.
	for i := len(lines) - 1; i > 0; i-- {
		if strings.TrimSpace(lines[i]) == "---" {
			lines[i] = fmt.Sprintf("%s: %t\n", key, value) + lines[i]
			return append([]byte(strings.Join(lines, "")), doc.Body...), nil
		}
	}
	return nil, fault(CodeInvalid, "missing closing record frontmatter", nil)
}

func commitmentChange(manifest, title, path string, queued bool, project Project) (workFileChange, error) {
	raw, err := os.ReadFile(manifest)
	if err != nil {
		return workFileChange{}, fault(CodeInternal, "read project commitments", err)
	}
	doc, err := recordread.ParseDocument(raw)
	if err != nil {
		return workFileChange{}, fault(CodeInvalid, "parse project commitments", err)
	}
	sections := recordread.Sections(doc.Body, manifest, map[string]string{"Current commitments": "commitment"}).Sections
	if len(sections) > 1 {
		return workFileChange{}, fault(CodeInvalid, "project must have one Current commitments section before managed queue changes", nil)
	}
	var links []recordread.Relationship
	if len(sections) == 1 {
		links = sections[0].Links
		if !sections[0].Complete {
			return workFileChange{}, fault(CodeInvalid, "project commitment references are incomplete", nil)
		}
	}
	reader, err := recordread.NewReader(project.BundlePath, project.AllowedSourceRoots)
	if err != nil {
		return workFileChange{}, err
	}
	defer reader.Close()
	lines := []string{}
	for _, link := range links {
		target, diagnostic := reader.LinkPath(manifest, link)
		if diagnostic != nil {
			return workFileChange{}, fault(CodeInvalid, "project commitment cannot be resolved", nil)
		}
		if filepath.Clean(target) == filepath.Clean(path) {
			continue
		}
		// Preserve the existing target and readable title where possible.
		label := link.Destination
		if content, diagnostic := reader.Read(target, recordread.RecordSource); diagnostic == nil {
			if d, e := recordread.ParseDocument([]byte(content.Text)); e == nil {
				if t, ok := d.Metadata["title"].(string); ok {
					label = t
				}
			}
		}
		lines = append(lines, "- ["+escapeLinkText(label)+"](<"+link.Destination+">)")
	}
	if queued {
		lines = append(lines, "- ["+escapeLinkText(title)+"]("+relativeLink(manifest, path)+")")
	}
	if len(lines) == 0 {
		lines = []string{"None"}
	}
	var updated []byte
	if len(sections) == 0 {
		updated = append(append([]byte{}, bytes.TrimRight(raw, "\n")...), "\n\n## Current commitments\n\n"+strings.Join(lines, "\n")+"\n"...)
	} else {
		updated, err = replaceSection(raw, manifest, "Current commitments", lines)
	}
	if err != nil {
		return workFileChange{}, err
	}
	return workFileChange{Path: manifest, Before: raw, After: updated}, nil
}

// markdownTargets reads actual Markdown links, excluding code examples and raw
// text. It is used before deleting a draft, including links in unknown sections.
func markdownTargets(body []byte) []string {
	targets := []string{}
	tree := parser.New().Parse(body)
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if link, ok := node.(*ast.Link); ok {
			targets = append(targets, link.Destination.Value(body))
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return targets
}
