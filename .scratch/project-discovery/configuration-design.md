# Read discovery configuration

The configuration reader owns ContextConfig validation and filesystem path interpretation. Existing application readers continue to receive explicit scope. This slice implements [ticket 01](../records/project-discovery/issues/01-read-configuration.md).

## Caller usage

```go
doc, err := discoveryconfig.Read(configFile, discoveryconfig.SharedProfile)
if err != nil {
	return err
}
records := doc.Shared.Project.Records
// A valid declaration can name an unavailable target.
// Discovery decides whether this path affects the selected scope.
if records.Failure != nil {
	return records.Failure
}
useRecords(records.Canonical)
```

Personal callers read `doc.Personal.Projects` and `doc.Personal.Workspaces` in authored order. Future setup can use the original `doc.Source` to preserve unknown YAML and compare the observed file before editing.

## Shape

One `Read(filename, profile)` operation returns typed declarations and captured source text. The filename must be absolute. Home lookup, ancestor search, selection, and writes belong to subsequent tickets.

```go
func Read(filename string, profile Profile) (Document, error) {
	// Read UTF-8 bytes and reuse recordread.ParseDocument.
	// Validate both profile structure and scoped key/alias uniqueness.
	// Resolve paths from the encountered file directory, then observe symlinks.
	return Document{}, errors.New("not implemented")
}

type Document struct {
	File, Source, Body string
	Metadata map[string]any
	Shared *SharedConfig
	Personal *PersonalConfig
}

type TargetPath struct {
	File, Field string
	Authored, Resolved, Canonical string
	Failure *FieldError
}
```

Exactly one profile is populated after a successful read. Project bindings contain records and allowed sources. Project registrations add a key, optional alias, and directory. Workspace declarations contain identity, title, and ordered members. Workspace registrations add a key, optional alias, and optional directory. Members carry their own records and source roots.

`internal/discoveryconfig/config.go` owns the public values and read operation. Private profile decoding and path interpretation remain within that package. `recordread.ParseDocument` owns Markdown framing and YAML decoding. No YAML AST crosses the configuration interface.

The reader preserves the full decoded metadata in one place and the original source as immutable strings. It does not duplicate unknown-field maps across typed values. A `FieldError` identifies the file and field, with an underlying error where available. Relative paths use the encountered filename's directory even when the configuration file is a symlink. Failed canonicalization stays on the affected target, leaving unrelated declarations usable. A canonical path establishes identity, not directory accessibility.

## Synthesis decision

The functional candidate is the base. One operation hides parsing, validation, duplicate detection, and path interpretation. The document-owned alternative required declaration snapshots, copying, path-handle ownership, and separate resolution calls for the same consumer outcome.

An independent cross-judge scored both candidates 2 out of 2 on each of the five criteria: interface depth, source retention and errors, path handling, testability, and ticket scope. The judge recommended the functional candidate for its smaller interface. The implementation follows that recommendation. The configured runner families unavailable in this session were replaced with the available Codex models.

Adopt the alternative's emphasis on complete error provenance and immutable original text. Reuse project and workspace value types within registrations instead of duplicating their fields. Keep all raw metadata at the document level. Omit path-handle tables, public resolution stages, and an error-code taxonomy until a caller needs them.

## Tradeoffs

- Path identities are observations made during the read. Discovery must validate selected directories before invoking a reader. There is no atomic filesystem snapshot.
- One fail-fast field error keeps malformed configurations unusable without introducing partial configuration results.
- Optional string values retain absence separately from authored empty strings. Required identities and paths reject whitespace-only values. Optional aliases and member titles are strings when present; selection can only use an alias that matches the caller's explicit value.
- Original source bytes preserve unknown YAML syntax that decoded metadata may normalize. A later writer must use the source when preserving such syntax.

## Verification

External-package tests cover both profiles, strict types and duplicate identities, exact source retention, declaring-file path bases, target aliases, missing and looping targets, independent record locations, and read-only behavior. Run the complete Go tests, race tests, vet, and build before recording acceptance.

On 2026-09-17, the package tests, complete Go suite, race suite, vet, and CLI build passed. Implementation fit the selected interface without a redesign.
