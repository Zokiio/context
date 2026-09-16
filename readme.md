# Project and context management

A Go CLI for keeping project records in local files tracked by Git and assembling task context for coding agents.

Agent sessions lose context. Developers then repeat requirements, reconstruct decisions, and gather the same documents for each new session. This project aims to preserve that information and select the sources an agent needs for a work item.

## Current status

The CLI reads a ticket, its recursive blockers, and each ticket's explicitly linked Spec and Context documents. Callers authorize external document directories and bound collection by file count and source bytes. The product name remains open; the development executable is `ctx`.

The first-reader milestone has passed local tests and a [skill-driven real-task acceptance trial](.scratch/context-reader/acceptance-trial.md).

The implementation uses a reusable Go core with a CLI as its first interface. The [product vision](docs/vision.md) describes the scope and adoption stages.

## First milestone

The first milestone is a read-only context reader. Given a project directory and a ticket path, the reader assembles the ticket, its blockers recursively, and each selected ticket's linked specifications and context documents.

The specified result contains whole source files, their paths, and the reasons for inclusion in a versioned JSON object. Sources reflect current files on disk, including uncommitted edits. If selected sources are unavailable or exceed configured limits, the reader returns the available context marked incomplete, with diagnostics and a nonzero exit status.

The [context reader specification](.scratch/context-reader/spec.md) defines selection rules, limits, and acceptance criteria.

## Project records

A project has its own identity, independent of where its records live. A project can span repositories, and a workspace groups projects without copying their authoritative records.

The recorded design uses one Open Knowledge Format (OKF) bundle per project. Markdown links in named sections express relationships between work records. The architectural decisions explain these choices:

- [One OKF bundle per project](docs/adr/0001-one-okf-bundle-per-project.md)
- [Work relationships in Markdown sections](docs/adr/0002-work-relationships-in-markdown-sections.md)

The [domain glossary](CONTEXT.md) defines project, workspace, work item, task context, and related terms.

## Repository conventions

Specs and tickets for developing this product live under `.scratch/` and are tracked in Git. These bootstrap records follow the [local issue tracker conventions](docs/agents/issue-tracker.md).

[Repository guidance](AGENTS.md) lists the instructions for agents working in this repository.

## Build and test

Install Go 1.25 or newer. Build and tests need no installed copy of `ctx`.

```sh
go build -o /tmp/ctx ./cmd/ctx
go test ./...
go vet ./...
```

Read a self-contained ticket from an explicit bundle directory:

```sh
/tmp/ctx context --project /path/to/bundle --ticket issues/example.md
```

`--project` is the OKF bundle root, such as this repository's `.scratch/records/`, and need not be a Git root. Relative project paths resolve against your working directory. Relative ticket paths resolve against the project directory. Both flags are required and have no environment or configuration fallback. Absolute ticket paths must resolve inside the selected bundle. The reader neither requires nor includes `project.md` automatically.

The reader includes the ticket, then its Spec documents, then its Context documents. It follows blockers breadth-first, including each blocker's Spec and Context documents before advancing to the next blocker. It recognizes level-two `Spec`, `Blocked by`, and `Context` headings, including repeated sections and nested subsections. A level-one or level-two heading ends a section.

Inline links and resolved reference links select documents. Undefined full or collapsed references produce diagnostics. Undefined shortcut text such as `[note]` remains prose. Images, code, and ordinary prose links do not select sources. Included documents do not expand selection through their own links.

Relative document links resolve from the referring ticket's directory. Leading-slash links resolve from the bundle root. Fragments select whole files without heading checks. Source identity uses resolved absolute paths, including symlink resolution. Repeated links and aliases add distinct inclusion reasons to one source.

Authorize external Spec and Context directories with repeatable `--allow-source` arguments. Each directory must exist; an invalid directory returns exit status `2`. Relative directories resolve from your working directory. Each argument is literal, including commas in directory names. Starting tickets and blockers must stay inside the bundle even when external directories are authorized.

For this repository, run from the repository root:

```sh
/tmp/ctx context --project .scratch/records \
	--ticket context-reader/issues/05-bound-context-collection.md \
	--allow-source .
```

This authorizes the repository directory because the tickets link to the root `CONTEXT.md`, the external spec, and `docs/`. Only authored relationship links select files within that scope.

The local [task-context skill](.agents/skills/task-context/SKILL.md) runs this reader and delivers the full selected source text to a coding agent before ticket implementation or acceptance verification.

Shared blockers appear once. Dependency cycles produce warnings and preserve success when every selected source is available. If a document is later selected as a blocker, the reader discovers its ticket relationships without adding a duplicate source.

Collection defaults to 100 files and 1,048,576 source bytes. Override these with `--max-files` and `--max-bytes`, each a positive integer. Zero, negative, and malformed flag values return exit status `2`. Byte counts use original source bytes before JSON encoding and count each source once.

At the first limit breach, the reader stops adding files and returns the whole files already included. Both completeness fields become `false`. Diagnostics identify the first excluded source and known pending sources. They do not enumerate descendants of tickets that were never explored. If the starting ticket exceeds the byte limit, the result has no sources.

## JSON contract

`context` writes one JSON object to standard output. Help and invocation or execution errors go to standard error.

| Field | Type | Meaning |
| --- | --- | --- |
| `schemaVersion` | integer | `1` |
| `complete` | boolean | Whether all selected sources were included |
| `traversalComplete` | boolean | Whether ticket discovery finished |
| `sources` | array | Included sources, or `[]` |
| `diagnostics` | array | Problems, or `[]` |
| `sources[].path` | string | Resolved absolute path, with symlink aliases resolved |
| `sources[].text` | string | Full original UTF-8 source text, including frontmatter |
| `sources[].sha256` | string | Lowercase hexadecimal SHA-256 of the same source bytes |
| `sources[].reasons` | array | Distinct reasons with kind `root`, `spec`, `context`, or `blocked_by`; the requested ticket has `[{"kind":"root"}]` |
| `sources[].reasons[].from` | string, optional | Referring ticket path for a relationship |
| `sources[].reasons[].link` | string, optional | Authored link destination, including fragments and escapes |
| `diagnostics[].from` | string, optional | Referring ticket path for a relationship problem |
| `diagnostics[].link` | string, optional | Authored destination, or the unresolved reference label in brackets |
| `diagnostics[].code` | string | Stable diagnostic identifier listed below |
| `diagnostics[].severity` | string | `error` or `warning` |
| `diagnostics[].message` | string | Human explanation; wording is not a stable contract |
| `diagnostics[].path` | string | Relevant absolute source path, resolved when available |

| Diagnostic code | Meaning |
| --- | --- |
| `source_missing` | The selected file does not exist |
| `source_unreadable` | The selected source cannot be read, including a directory selected as a file |
| `source_outside_scope` | A ticket resolves outside the bundle, or a document is outside all permitted roots |
| `unsupported_source` | A relationship identifies a remote URL or unsupported target |
| `unresolved_reference` | A full or collapsed reference link has no definition |
| `invalid_frontmatter` | Leading YAML is malformed, unterminated, empty, or not a mapping |
| `invalid_source_encoding` | The source is not UTF-8 and cannot be represented unchanged in JSON |
| `dependency_cycle` | A blocker relationship closes a dependency cycle; warning only |
| `source_limit_exceeded` | Including this source would exceed a collection limit |
| `source_omitted` | A known pending source was not processed after a limit breach |

Source and relationship errors set `complete` to `false`. A missing linked document or unresolved Spec or Context reference does not prevent ticket relationship discovery, so `traversalComplete` stays `true`. An unresolved blocker reference sets both fields to `false` because the blocker and its descendants cannot be discovered. An unavailable ticket or invalid ticket frontmatter sets both fields to `false`. Invalid frontmatter retains the full source and digest. Invalid UTF-8 omits the source to avoid returning changed bytes. Unknown metadata keys and concept types are accepted. Successful parsing does not certify OKF schema validity or work readiness.

Exit status is `0` for complete JSON, `1` for incomplete JSON, and `2` for invalid invocation or failure to run or render the operation. A missing ticket is status `1`; an invalid project directory is status `2`.

The reader uses current files on disk and does not write records. Digests identify each file's bytes; they do not claim an atomic snapshot across files.

## Go package responsibilities

- `cmd/ctx` wires the application operation into the CLI and exits with its status.
- `internal/cli` owns urfave/cli v3 flags, JSON rendering, and exit statuses.
- `internal/taskcontext` exposes `Assemble(context.Context, Request) (Result, error)` and owns private document parsing and filesystem access through `os.Root`.

The application operation does not depend on CLI types, print output, or exit. Dependencies are pinned in `go.mod`: urfave/cli v3.12.0, Goldmark v2.1.0, goccy/go-yaml v1.19.2, go-cmp v0.7.0, and go-internal v1.16.0. Application tests use real temporary directories; testscript covers the CLI contract.

For direct Go callers, zero-valued `Request.MaxFiles` and `Request.MaxBytes` select the defaults. Negative limits return an operation error. CLI callers must use positive values when supplying either flag.
