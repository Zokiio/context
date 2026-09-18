package resumption

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Zokiio/context/internal/recordread"
	"github.com/Zokiio/context/internal/taskcontext"
)

func validateSnapshot(data []byte, observation *Observation) (taskcontext.Result, error) {
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	observation.Snapshot.ObservedSHA256 = &digest
	fail := func(message string) (taskcontext.Result, error) { return taskcontext.Result{}, errors.New(message) }
	if digest != observation.Snapshot.RecordedSHA256 {
		return fail("retained context byte digest does not match note")
	}
	if !utf8.Valid(data) {
		return fail("retained context must be UTF-8")
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return fail("retained context must be one JSON object")
	}
	if err := jsonFields(object, map[string]string{"schemaVersion": "number", "complete": "boolean", "traversalComplete": "boolean", "sources": "array", "diagnostics": "array"}); err != nil {
		return fail(err.Error())
	}
	if string(object["schemaVersion"]) != "1" {
		return fail("retained context schemaVersion must be integer 1")
	}
	var sources []map[string]json.RawMessage
	if err := json.Unmarshal(object["sources"], &sources); err != nil {
		return fail("retained sources must be objects")
	}
	for _, source := range sources {
		if err := jsonFields(source, map[string]string{"path": "string", "text": "string", "sha256": "string", "reasons": "array"}); err != nil {
			return fail("retained source: " + err.Error())
		}
		var reasons []map[string]json.RawMessage
		if err := json.Unmarshal(source["reasons"], &reasons); err != nil {
			return fail("retained reasons must be objects")
		}
		for _, reason := range reasons {
			if err := jsonFields(reason, map[string]string{"kind": "string"}); err != nil {
				return fail("retained reason: " + err.Error())
			}
			if err := optionalJSONStrings(reason, "from", "link"); err != nil {
				return fail(err.Error())
			}
			var kind string
			_ = json.Unmarshal(reason["kind"], &kind)
			if kind == "root" {
				if _, ok := reason["from"]; ok {
					return fail("root reason must not contain from")
				}
				if _, ok := reason["link"]; ok {
					return fail("root reason must not contain link")
				}
			}
		}
	}
	var diagnostics []map[string]json.RawMessage
	if err := json.Unmarshal(object["diagnostics"], &diagnostics); err != nil {
		return fail("retained diagnostics must be objects")
	}
	for _, diagnostic := range diagnostics {
		if err := jsonFields(diagnostic, map[string]string{"code": "string", "severity": "string", "message": "string", "path": "string"}); err != nil {
			return fail("retained diagnostic: " + err.Error())
		}
		if err := optionalJSONStrings(diagnostic, "from", "link"); err != nil {
			return fail(err.Error())
		}
	}
	var retained taskcontext.Result
	if err := json.Unmarshal(data, &retained); err != nil {
		return fail(err.Error())
	}
	seen := map[string]bool{}
	roots := 0
	for _, source := range retained.Sources {
		if !filepath.IsAbs(source.Path) || filepath.Clean(source.Path) != source.Path || seen[source.Path] {
			return fail("retained source paths must be distinct canonical absolute paths")
		}
		seen[source.Path] = true
		if source.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(source.Text))) {
			return fail("retained source text digest does not match")
		}
		if len(source.Reasons) == 0 {
			return fail("retained source must have an inclusion reason")
		}
		for _, reason := range source.Reasons {
			switch reason.Kind {
			case "root":
				roots++
				if reason.From != "" || reason.Link != "" {
					return fail("root reason must not have a referring source or link")
				}
				doc, err := recordread.ParseDocument([]byte(source.Text))
				if err != nil {
					return fail("retained root is not a valid WorkItem")
				}
				typ, _ := doc.Metadata["type"].(string)
				id, _ := doc.Metadata["id"].(string)
				if typ != "WorkItem" || strings.TrimSpace(id) != observation.TaskID {
					return fail("retained root WorkItem identity does not match the selected task")
				}
			case "spec", "context", "blocked_by":
			default:
				return fail("retained source has an unknown reason kind")
			}
		}
	}
	if roots != 1 {
		return fail("retained context must have exactly one root reason")
	}
	for _, diagnostic := range retained.Diagnostics {
		if diagnostic.Severity != "error" && diagnostic.Severity != "warning" {
			return fail("retained diagnostic has an invalid severity")
		}
	}
	count := len(retained.Sources)
	observation.Snapshot.SourceCount = &count
	return retained, nil
}

func jsonFields(object map[string]json.RawMessage, fields map[string]string) error {
	for key, kind := range fields {
		raw, ok := object[key]
		raw = bytes.TrimSpace(raw)
		if !ok || len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
			return fmt.Errorf("%s is required and must not be null", key)
		}
		valid := false
		switch kind {
		case "string":
			var value string
			valid = json.Unmarshal(raw, &value) == nil
		case "boolean":
			var value bool
			valid = json.Unmarshal(raw, &value) == nil
		case "array":
			valid = raw[0] == '['
		case "number":
			var value json.Number
			valid = raw[0] != '"' && json.Unmarshal(raw, &value) == nil
		}
		if !valid {
			return fmt.Errorf("%s must be %s", key, kind)
		}
	}
	return nil
}
func optionalJSONStrings(object map[string]json.RawMessage, keys ...string) error {
	for _, key := range keys {
		if _, ok := object[key]; ok {
			if err := jsonFields(object, map[string]string{key: "string"}); err != nil {
				return err
			}
		}
	}
	return nil
}
