package taskcontext

import "github.com/Zokiio/context/internal/recordread"

type document struct {
	metadata map[string]any
	body     []byte
}

func parseDocument(source []byte) (document, error) {
	parsed, err := recordread.ParseDocument(source)
	return document{metadata: parsed.Metadata, body: parsed.Body}, err
}
