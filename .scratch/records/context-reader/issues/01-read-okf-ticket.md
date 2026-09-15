---
type: WorkItem
id: bcbad059-f0b1-401b-99e1-3fdc468db8fb
title: "Read an OKF ticket as JSON context"
triage: ready-for-agent
---

# 01: Read an OKF ticket as JSON context

**What to build:** A caller can select a project directory and a self-contained ticket, then receive the ticket's current source text and content digest as one JSON context result. This first slice establishes the working CLI and Go operation, including frontmatter handling and predictable failure output.

## Scope

Exercise self-contained tickets without relationship links in this slice. Linked-document selection, recursive blockers, external context, and configurable limits arrive in the following tickets. Full OKF validation and automatic project-manifest inclusion remain out of scope.

## Acceptance criteria

- [ ] The context subcommand accepts required --project and --ticket arguments through urfave/cli v3. Relative project paths resolve against the caller's working directory and relative ticket paths against the project directory. No configuration or environment fallback supplies either argument.
- [ ] One Go module contains internal packages with the agreed responsibilities: entry-point wiring, CLI input/output, and context assembly with private document-reading details. CLI framework types do not enter the application operation.
- [ ] An existing project directory works outside a Git root and from an unrelated working directory. Root-scoped file access uses os.Root and rejects a starting ticket outside the selected bundle.
- [ ] The private reader separates leading YAML frontmatter from the Markdown body and parses metadata with goccy/go-yaml as a mapping. It accepts unknown fields and types without typed-decoding rejection; body content is not inferred from metadata.
- [ ] Malformed, unterminated, or non-mapping ticket frontmatter retains the original source text and digest, emits a diagnostic, sets both completeness fields false, and returns exit code 1. Full schema validation is not required.
- [ ] The JSON result has schemaVersion 1, complete, traversalComplete, sources, and diagnostics. A source includes its resolved absolute path, unchanged full text, root-selection reason, and SHA-256 digest computed from the same bytes. Document the chosen nested field names and diagnostic codes.
- [ ] Uncommitted edits and new files appear exactly as read, with matching source digests. Do not read committed substitutes, rewrite files, or claim an atomic snapshot across sources.
- [ ] A complete self-contained ticket returns exit code 0. A missing or unreadable starting ticket returns JSON with both completeness fields false and exit code 1. Missing arguments, an invalid project directory, and failures to run the operation return exit code 2 with an explanation on standard error.
- [ ] Explicitly configure CLI writers and error/exit handling so standard output contains only the result JSON. The Go operation returns data and errors without printing or exiting.
- [ ] Application tests use real temporary directories and go-cmp, including a second independent project and unchanged-source assertions. A small testscript suite verifies arguments, JSON output, and exit status; build and tests require no installed copy of this product.
- [ ] Select compatible released dependency pins and an executable name as implementation choices, and document ordinary build/test commands. Introduce dependencies when this slice uses them; do not add future interface or configuration packages.

## Blocked by

None (can start immediately).

## Spec

- [Context reader specification](../../../context-reader/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
