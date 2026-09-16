package taskcontext

import "github.com/Zokiio/context/internal/recordread"

type relationship struct{ kind, link, destination string }
type relationshipResult struct {
	links             []relationship
	diagnostics       []Diagnostic
	traversalComplete bool
}

func extractRelationships(body []byte, path string) relationshipResult {
	sections := recordread.Sections(body, path, map[string]string{"Spec": "spec", "Blocked by": "blocked_by", "Context": "context"})
	links := make([]relationship, 0, len(sections.Links))
	for _, link := range sections.Links {
		links = append(links, relationship{kind: link.Kind, link: link.Link, destination: link.Destination})
	}
	return relationshipResult{links: links, diagnostics: sections.Diagnostics, traversalComplete: !sections.InvalidSections["Blocked by"]}
}
