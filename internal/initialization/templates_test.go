package initialization

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGuidance = flag.Bool("update", false, "regenerate this repository's guidance from the embedded templates")

func TestRepositoryGuidance(t *testing.T) {
	documents, err := renderGuidance(Request{
		Skills: ".agents/skills", Docs: "docs/agents", Records: ".scratch/records",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range documents {
		if filepath.Base(doc.path) == "tracker.md" {
			continue // This repository retains its authored tracker conventions.
		}
		path := filepath.Join("..", "..", doc.path)
		if *updateGuidance {
			if err := os.WriteFile(path, doc.text, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		current, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(current, doc.text) {
			t.Errorf("%s differs from its embedded template; run go generate ./internal/initialization", doc.path)
		}
	}
}
