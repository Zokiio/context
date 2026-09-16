package orientation

import (
	"regexp"
	"strings"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
)

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type snapshotEntry struct {
	kind      string
	link      recordread.Relationship
	digest    string
	reference Reference
	source    *recordread.Source
}

type acceptanceSources struct {
	entries           []snapshotEntry
	reasons           []Finding
	requirementsKnown bool
}

// Snapshot syntax is checked against the Markdown tree. Link labels and prose
// cannot accidentally turn a second link or a fenced digest into a snapshot.
func parseSnapshotSections(body []byte, path string) acceptanceSources {
	result := acceptanceSources{entries: []snapshotEntry{}, reasons: []Finding{}, requirementsKnown: true}
	sections := recordread.Sections(body, path, map[string]string{"Requirements": "requirement", "Evidence": "evidence"})
	tree := parser.New().Parse(body)
	present, counts := map[string]bool{}, map[string]int{}
	invalid := func(name, message string) {
		result.reasons = append(result.reasons, Finding{Code: "invalid_snapshot", Message: message, Path: path})
		if name == "Requirements" {
			result.requirementsKnown = false
		}
	}
	for _, section := range sections.Sections {
		present[section.Name] = true
		kind := "requirement"
		if section.Name == "Evidence" {
			kind = "evidence"
		}
		if !section.Complete {
			invalid(section.Name, section.Name+" contains an unresolved Markdown reference")
		}
		for node := tree.FirstChild(); node != nil; node = node.NextSibling() {
			if node.Pos() < section.ContentStart || node.Pos() >= section.End {
				continue
			}
			if _, definition := node.(*ast.LinkReferenceDefinition); definition {
				continue
			}
			list, ok := node.(*ast.List)
			if !ok {
				if _, paragraph := node.(*ast.Paragraph); paragraph && section.Name == "Requirements" && strings.TrimSpace(recordread.PlainText(node, body)) == "None" && len(section.Links) == 0 && strings.TrimSpace(section.Text) == "None" {
					continue
				}
				invalid(section.Name, section.Name+" must contain snapshot list items")
				continue
			}
			for item := list.FirstChild(); item != nil; item = item.NextSibling() {
				links := []recordread.Relationship{}
				digests := []string{}
				ambiguous := false
				_ = ast.Walk(item, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
					if !entering {
						return ast.WalkContinue, nil
					}
					switch child := child.(type) {
					case *ast.Link:
						links = append(links, recordread.Relationship{Kind: kind, Link: string(child.Destination.Bytes(body)), Destination: child.Destination.Value(body)})
						return ast.WalkSkipChildren, nil
					case *ast.AutoLink:
						links = append(links, recordread.Relationship{Kind: kind, Link: string(child.Destination.Bytes(body)), Destination: child.Destination.Value(body)})
					case *ast.CodeSpan:
						digests = append(digests, child.Value.Value(body))
					case *ast.Image, *ast.CodeBlock, *ast.List:
						ambiguous = true
					}
					return ast.WalkContinue, nil
				})
				if ambiguous || len(links) != 1 || len(digests) != 1 || !sha256Pattern.MatchString(digests[0]) {
					invalid(section.Name, section.Name+" snapshot requires one file link and one inline-code lowercase SHA-256 digest")
					continue
				}
				result.entries = append(result.entries, snapshotEntry{kind: kind, link: links[0], digest: digests[0]})
				counts[section.Name]++
			}
		}
	}
	for _, name := range []string{"Requirements", "Evidence"} {
		if !present[name] {
			result.reasons = append(result.reasons, Finding{Code: "missing_snapshot_section", Message: "Acceptance requires a " + name + " section", Path: path})
			if name == "Requirements" {
				result.requirementsKnown = false
			}
		}
	}
	if present["Evidence"] && counts["Evidence"] == 0 {
		result.reasons = append(result.reasons, Finding{Code: "empty_evidence", Message: "Acceptance requires at least one evidence snapshot", Path: path})
	}
	return result
}

func (e *evaluator) selectAcceptanceSources(subject *record) {
	links := []*selection{}
	for _, s := range e.selections {
		if s.from == subject.source.Path && s.link.Kind == "acceptance" {
			links = append(links, s)
		}
	}
	if len(links) != 1 || links[0].target == nil || links[0].target.kind != "Acceptance" || links[0].source == nil {
		return
	}
	r := links[0].target
	if _, exists := e.acceptanceSources[r.source.Path]; exists {
		return
	}
	captured := parseSnapshotSections(r.doc.Body, r.source.Path)
	for index := range captured.entries {
		entry := &captured.entries[index]
		entry.reference = Reference{From: r.source.Path, Link: entry.link.Link}
		path, diagnostic := e.reader.LinkPath(r.source.Path, entry.link)
		entry.reference.Path = path
		if diagnostic != nil {
			e.diagnose(Diagnostic(*diagnostic))
			continue
		}
		entry.source = e.read(path, recordread.DocumentSource, Reason{Kind: entry.kind, From: r.source.Path, Link: entry.link.Link})
		if entry.source == nil {
			continue
		}
		entry.reference.Path = entry.source.Path
		if doc, err := recordread.ParseDocument([]byte(entry.source.Text)); err == nil {
			switch stringField(doc.Metadata, "type") {
			case "Project", "WorkItem", "Decision", "Acceptance":
				entry.source = e.read(path, recordread.RecordSource, Reason{Kind: entry.kind, From: r.source.Path, Link: entry.link.Link})
			}
		}
		if target := e.records[entry.reference.Path]; target != nil {
			entry.reference.ID, entry.reference.Title = target.id, target.title
		}
	}
	e.acceptanceSources[r.source.Path] = captured
}
