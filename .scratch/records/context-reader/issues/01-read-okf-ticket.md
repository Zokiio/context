---
type: WorkItem
id: bcbad059-f0b1-401b-99e1-3fdc468db8fb
title: "Read an OKF ticket as JSON context"
triage: ready-for-agent
execution: completed
---

# 01: Read an OKF ticket as JSON context

**What to build:** A caller can select a project directory and a self-contained ticket, then receive the ticket's current source text and content digest as one JSON context result. This first slice establishes the working CLI and Go operation, including frontmatter handling and predictable failure output.

## Scope

Exercise self-contained tickets without relationship links in this slice. Linked-document selection, recursive blockers, external context, and configurable limits arrive in the following tickets. Full OKF validation and automatic project-manifest inclusion remain out of scope.

## Acceptance criteria

- [x] The context subcommand accepts required --project and --ticket arguments through urfave/cli v3. Relative project paths resolve against the caller's working directory and relative ticket paths against the project directory. No configuration or environment fallback supplies either argument.
- [x] One Go module contains internal packages with the agreed responsibilities: entry-point wiring, CLI input/output, and context assembly with private document-reading details. CLI framework types do not enter the application operation.
- [x] An existing project directory works outside a Git root and from an unrelated working directory. Root-scoped file access uses os.Root and rejects a starting ticket outside the selected bundle.
- [x] The private reader separates leading YAML frontmatter from the Markdown body and parses metadata with goccy/go-yaml as a mapping. It accepts unknown fields and types without typed-decoding rejection; body content is not inferred from metadata.
- [x] Malformed, unterminated, or non-mapping ticket frontmatter retains the original source text and digest, emits a diagnostic, sets both completeness fields false, and returns exit code 1. Full schema validation is not required.
- [x] The JSON result has schemaVersion 1, complete, traversalComplete, sources, and diagnostics. A source includes its resolved absolute path, unchanged full text, root-selection reason, and SHA-256 digest computed from the same bytes. Document the chosen nested field names and diagnostic codes.
- [x] Uncommitted edits and new files appear exactly as read, with matching source digests. Do not read committed substitutes, rewrite files, or claim an atomic snapshot across sources.
- [x] A complete self-contained ticket returns exit code 0. A missing or unreadable starting ticket returns JSON with both completeness fields false and exit code 1. Missing arguments, an invalid project directory, and failures to run the operation return exit code 2 with an explanation on standard error.
- [x] Explicitly configure CLI writers and error/exit handling so standard output contains only the result JSON. The Go operation returns data and errors without printing or exiting.
- [x] Application tests use real temporary directories and go-cmp, including a second independent project and unchanged-source assertions. A small testscript suite verifies arguments, JSON output, and exit status; build and tests require no installed copy of this product.
- [x] Select compatible released dependency pins and an executable name as implementation choices, and document ordinary build/test commands. Introduce dependencies when this slice uses them; do not add future interface or configuration packages.

## Blocked by

None

## Blocked by decisions

None

## Spec

- [Context reader specification](../../../context-reader/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)

## Comments

### 2026-09-15 implementation assignment

Agent `/root/ticket01` owns the first Go operation, CLI, tests, and build/schema documentation for this ticket. Later selection and limit behavior remain assigned to subsequent tickets. No commit has been created.


### 2026-09-15 completion evidence

Implemented by `/root/ticket01` in the uncommitted working tree based on commit `0c74f92506a67242247a4e76a40785561a561bb3`. This evidence applies to the ticket01 implementation before subsequent tickets modify it.

- Added `cmd/ctx`, `internal/cli`, and `internal/taskcontext` in module `github.com/Zokiio/context`. The CLI injects the application operation; no CLI types enter context assembly.
- Added the self-contained ticket reader with constrained `os.Root` reads, original UTF-8 text and SHA-256 digests, strict mapping frontmatter, and the versioned result and exit contracts. Unknown metadata and tagged or anchored mappings are accepted.
- Application tests cover two independent projects, current file edits, byte preservation with CRLF and Unicode, malformed/nonmapping/unterminated YAML, missing and unreadable sources, symlink containment and project aliases, cancellation, and unchanged records. The testscript suite checks relative invocation from an unrelated directory, one JSON result, required arguments, explicit exit statuses, and no environment fallback. CLI unit tests cover operation and output-writer errors.
- Passed `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build -o /tmp/context-ticket01-ctx ./cmd/ctx` locally with Go 1.27.1. The module declares Go 1.25.0; that minimum was selected from API and dependency requirements but was not separately executed.
- Documented commands, package responsibilities, dependency versions, nested JSON fields, diagnostic codes, and exit statuses in the root readme. Release pins were checked against the Go module proxy.
- Review findings for valid YAML mapping wrappers and absolute paths through a project symlink were fixed and covered by tests. Invalid UTF-8 now produces `invalid_source_encoding` and no source instead of silently replacing bytes in JSON.

Relationship traversal and limits remain scoped to later tickets. No milestone-level real-task acceptance is claimed by this first slice.

## Acceptance

- [Current acceptance](../acceptances/01-read-okf-ticket-migration-20260916.md)
