package recordread_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/goccy/go-yaml"
)

func TestParseDocumentRejectsResolvedDuplicateKeys(t *testing.T) {
	for _, test := range []struct {
		name, metadata string
	}{
		{"nested alias after literal", "name: &key records\nproject: {records: first, *key : second}"},
		{"nested alias before literal", "name: &key records\nproject: {*key : first, records: second}"},
		{"sequence member", "name: &key records\nmembers: [{records: first, *key : second}]"},
		{"anchor within mapping", "project: {name: &key records, records: first, *key : second}"},
		{"root alias", "name: &key records\nrecords: first\n*key : second"},
		{"numeric keys", "custom: {1: first, 0x1: second}"},
		{"boolean keys", "custom: {true: first, TRUE: second}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte("---\n" + test.metadata + "\n---\nBody.\n")
			before := bytes.Clone(source)
			_, err := recordread.ParseDocument(source)
			var duplicate *yaml.DuplicateKeyError
			if !errors.As(err, &duplicate) {
				t.Fatalf("error = %v; want duplicate key diagnostic", err)
			}
			if duplicate.GetToken() == nil || !strings.Contains(err.Error(), "parse frontmatter:") {
				t.Fatalf("duplicate error lost parser attribution: %v", err)
			}
			if !bytes.Equal(source, before) {
				t.Fatal("validation modified source bytes")
			}
		})
	}
}

func TestParseDocumentKeepsUniqueAliasKeysAndMergeOverrides(t *testing.T) {
	source := []byte("---\n" + `name: &key records
defaults: &defaults {records: inherited, custom: retained}
project: {<<: *defaults, *key : selected}
members: [{*key : member}]
unknown: {17: preserved}
` + "---\n\nKeep **Markdown**.\n")
	before := bytes.Clone(source)
	document, err := recordread.ParseDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	project := document.Metadata["project"].(map[string]any)
	if project["records"] != "selected" || project["custom"] != "retained" {
		t.Fatalf("merge override changed: %#v", project)
	}
	member := document.Metadata["members"].([]any)[0].(map[string]any)
	if member["records"] != "member" || document.Metadata["unknown"].(map[string]any)["17"] != "preserved" {
		t.Fatalf("alias key or unknown metadata changed: %#v", document.Metadata)
	}
	if !bytes.Equal(source, before) || string(document.Body) != "\nKeep **Markdown**.\n" {
		t.Fatal("source or Markdown body changed")
	}
}

func TestParseDocumentResolvesAliasKeysInDeclarationOrder(t *testing.T) {
	source := []byte("---\n" + `first: &key before
project: {*key : first, after: second}
second: &key after
other: {*key : third, before: fourth}
` + "---\n")
	document, err := recordread.ParseDocument(source)
	if err != nil {
		t.Fatal(err)
	}
	project := document.Metadata["project"].(map[string]any)
	other := document.Metadata["other"].(map[string]any)
	if project["before"] != "first" || project["after"] != "second" || other["after"] != "third" || other["before"] != "fourth" {
		t.Fatalf("anchor redefinition changed earlier keys: %#v", document.Metadata)
	}
}
