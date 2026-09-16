package orientation

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
)

type requirementFingerprints struct {
	Version        int
	TicketSHA256   string
	CriteriaSHA256 *string
}

type fingerprintSpan struct{ start, end int }

func setWorkItemFingerprints(work *WorkItem, body []byte) {
	fingerprints := fingerprintBody(body)
	work.FingerprintVersion = &fingerprints.Version
	work.TicketSHA256 = &fingerprints.TicketSHA256
	work.CriteriaSHA256 = fingerprints.CriteriaSHA256
}

// fingerprintBody uses the same captured body as record interpretation. Version 1
// preserves raw bytes; neither Markdown rendering nor metadata serialization is
// part of its input.
func fingerprintBody(body []byte) requirementFingerprints {
	sections := recordread.Sections(body, "", map[string]string{
		"Comments": "comments", "Acceptance": "acceptance", "Acceptance criteria": "criteria",
	})
	retained := []fingerprintSpan{}
	criteriaSpans := []fingerprintSpan{}
	start := 0
	for _, section := range sections.Sections {
		switch section.Name {
		case "Comments", "Acceptance":
			retained = append(retained, fingerprintSpan{start, section.Start})
			start = section.End
		case "Acceptance criteria":
			criteriaSpans = append(criteriaSpans, fingerprintSpan{section.Start, section.End})
		}
	}
	retained = append(retained, fingerprintSpan{start, len(body)})
	var ticket bytes.Buffer
	for _, span := range retained {
		ticket.Write(body[span.start:span.end])
	}
	tree := parser.New().Parse(body)
	definitions := fingerprintDefinitions(body, tree)
	ticketParts := append([][]byte{ticket.Bytes()}, additionalDefinitions(body, tree, definitions, retained)...)
	result := requirementFingerprints{Version: 1, TicketSHA256: fingerprintDigest("ctx.ticket-body.v1", ticketParts)}
	if len(criteriaSpans) > 0 {
		criteriaParts := make([][]byte, 0, len(criteriaSpans))
		for _, span := range criteriaSpans {
			criteriaParts = append(criteriaParts, body[span.start:span.end])
		}
		criteriaParts = append(criteriaParts, additionalDefinitions(body, tree, definitions, criteriaSpans)...)
		digest := fingerprintDigest("ctx.acceptance-criteria.v1", criteriaParts)
		result.CriteriaSHA256 = &digest
	}
	return result
}

// Reference links borrow their destination's source index from the effective
// definition node. Matching that index preserves the parser's label resolution,
// including collapsed links and duplicate definitions.
func fingerprintDefinitions(body []byte, tree ast.Node) map[text.Index]fingerprintSpan {
	definitions := map[text.Index]fingerprintSpan{}
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		definition, ok := node.(*ast.LinkReferenceDefinition)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		end := definition.Destination.Index().Stop
		for _, index := range definition.Title.Indices() {
			end = max(end, index.Stop)
		}
		start := bytes.LastIndexByte(body[:definition.Pos()], '\n') + 1
		if newline := bytes.IndexByte(body[end:], '\n'); newline >= 0 {
			end += newline + 1
		} else {
			end = len(body)
		}
		definitions[definition.Destination.Index()] = fingerprintSpan{start, end}
		return ast.WalkSkipChildren, nil
	})
	return definitions
}

func additionalDefinitions(body []byte, tree ast.Node, definitions map[text.Index]fingerprintSpan, retained []fingerprintSpan) [][]byte {
	parts := [][]byte{}
	seen := map[text.Index]bool{}
	_ = ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || !containsFingerprintPosition(retained, node.Pos()) {
			return ast.WalkContinue, nil
		}
		var reference *ast.ReferenceLink
		var destination text.Index
		switch node := node.(type) {
		case *ast.Link:
			reference = node.Reference
			destination = node.Destination.Index()
		case *ast.Image:
			reference = node.Reference
			destination = node.Destination.Index()
		}
		if reference == nil {
			return ast.WalkContinue, nil
		}
		if seen[destination] {
			return ast.WalkContinue, nil
		}
		seen[destination] = true
		definition, found := definitions[destination]
		if found && !containsFingerprintSpan(retained, definition) {
			parts = append(parts, body[definition.start:definition.end])
		}
		return ast.WalkContinue, nil
	})
	return parts
}

func containsFingerprintPosition(spans []fingerprintSpan, position int) bool {
	for _, span := range spans {
		if span.start <= position && position < span.end {
			return true
		}
	}
	return false
}

func containsFingerprintSpan(spans []fingerprintSpan, candidate fingerprintSpan) bool {
	for _, span := range spans {
		if span.start <= candidate.start && candidate.end <= span.end {
			return true
		}
	}
	return false
}

func fingerprintDigest(tag string, parts [][]byte) string {
	hash := sha256.New()
	hash.Write([]byte(tag))
	hash.Write([]byte{0})
	var count [8]byte
	binary.BigEndian.PutUint64(count[:], uint64(len(parts)))
	hash.Write(count[:])
	for _, part := range parts {
		binary.BigEndian.PutUint64(count[:], uint64(len(part)))
		hash.Write(count[:])
		hash.Write(part)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
