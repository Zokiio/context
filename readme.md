# Project and context management

A Go CLI for inspecting local project records and assembling task context for coding agents.

Agent sessions lose context. Developers then repeat requirements, reconstruct decisions, and gather the same documents for each new session. This project aims to preserve that information and select the sources an agent needs for a work item.

## Current status

The CLI reads a ticket, its recursive blockers, and each ticket's explicitly linked Spec and Context documents. Callers authorize external document directories and bound collection by file count and source bytes. The product name remains open; the development executable is `ctx`.

`ctx orient` shows one project's authored goals, commitments, work inventory, and eligible work with reasons. It evaluates prerequisites, cycles, blocking decisions, and recorded acceptance from captured local sources. Missing required information remains explicit, and requirement fingerprints support authored acceptance records.

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

## Orient a session

Inspect one project bundle before choosing a ticket:

```sh
/tmp/ctx orient --project /path/to/bundle
/tmp/ctx orient --project /path/to/bundle --json
```

`orient` requires `--project` and accepts no ticket or positional argument. It reads `project.md` first, then discovers Markdown records in the bundle. A project need not be a Git repository. Discovery includes untracked and ignored files and does not traverse directory symlink aliases.

`--project`, `--max-files`, `--max-bytes`, and `--json` each accept at most one occurrence. Repeating one returns exit status `2`, including when its values agree. `--allow-source` remains repeatable.

The default text report shows project identity, authored goals, current commitments, work in progress, backlog, decisions, and gaps. Each work item keeps execution, triage, commitment membership, readiness, and eligibility separate. Source paths, whole-file digests, and relationship reasons identify the records behind the report. No generated summary replaces authored goals.

Use a selected work item's source path as `ctx context --ticket` to obtain its complete requirements. Reuse the project and allowed-source arguments. The [authoring profiles](docs/agents/issue-tracker.md) define the manifest sections and record fields.

Authorize external goal, specification, context, and evidence documents with repeatable `--allow-source` arguments. Each argument is literal, including commas. Project and allowed-source paths resolve against the caller's working directory. Record targets must remain inside the bundle, even when external document roots are authorized.

Collection defaults to 100 distinct files and 1,048,576 source bytes across the manifest, inventory, and linked documents. `--max-files` and `--max-bytes` accept positive overrides. Each resolved source is read once; all parsing and digests use that captured content. Collection stops at the first limit breach and identifies known pending sources. It does not claim to enumerate undiscovered records.

The command leaves records and Git state unchanged. Digests identify observed file contents, not an atomic snapshot of the entire filesystem.

### Orientation JSON

`--json` writes one schema-version-1 object containing the same consumed facts as the text report, plus preserved authored metadata. Lists are always arrays. Unavailable identity, execution, membership, and fingerprint values are `null`. Unavailable project identity is `project: null`; evaluation states use the explicit value `unknown`.

| Field | Meaning |
| --- | --- |
| `schemaVersion` | `1` |
| `complete` | Required information was available and evaluable within the limits |
| `inventoryComplete` | All candidate Markdown records were discovered and parsed |
| `project` | ID, title, source path, known-section flags, and authored metadata |
| `goals` | Authored goal text, its source, and linked source references |
| `currentCommitments` | Direct ticket references in authored commitment order |
| `workItems` | Discovered work-item facts and explained evaluation results |
| `shortlist` | Eligible references in authored commitment order |
| `inProgress` | References to work recorded as in progress |
| `backlog` | References to unfinished work outside current commitments |
| `decisions` | Decision identity, authoritative state and resolution, check status and reasons, affected work, source, references, and metadata |
| `sources` | Resolved paths, whole-file SHA-256 digests, and distinct inclusion reasons |
| `diagnostics` | Stable codes, severity, explanations, and affected source or relationship |

Each work item includes `id`, `title`, `source`, `identityAmbiguous`, `triage`, `execution`, `lifecycle`, `committed`, `specifications`, `dependencies`, `checks`, `readiness`, `eligible`, `exclusionReasons`, `acceptance`, `fingerprintVersion`, `ticketSHA256`, `criteriaSHA256`, and authored `metadata`. A check contains `name`, `status`, and `reasons`. Check status is `pass`, `fail`, or `unknown`; aggregate readiness is `ready`, `blocked`, or `unknown`. `eligible` is a separate boolean.

`dependencies` contains each reachable direct or transitive authored edge once, in depth-first authored order. Each edge has `from` and `to` references, `status`, `cycle`, and `reasons`. The target reference retains the referring source and authored link. `cycle: true` identifies edges inside a cyclic component. The separate `dependencies` check retains the aggregate result and explanations.

Work items and decisions expose `identityAmbiguous: true` when typed records share their ID. Reports retain each record and source path without selecting one as authoritative. Missing IDs remain `null`. An omitted work-item `status` gives the effective lifecycle `stable`, while its authored metadata remains unchanged.

Unknown metadata fields are retained in JSON. YAML values that JSON cannot represent use tagged objects. Non-finite floats use `{"yamlType":"float","value":".nan"}`, with `value` set to `.nan`, `.inf`, or `-.inf`. Mappings with non-string keys use entries such as `{"yamlType":"mapping","entries":[{"key":1,"value":"one"}]}`. Ordinary JSON-compatible metadata keeps its normal representation.

A reference contains resolved `path`, nullable `id` and `title`, and optional referring `from` and authored `link`. Source reasons contain `kind` and optional `from` and `link`. Findings and diagnostics retain their source path and relationship when known. Source digests are separate from requirement fingerprints. Orientation does not return each source's full body.

Each selected decision exposes `state`, `resolution`, `checkStatus`, `reasons`, and `affectedWork`. Check status is `fail` for an open decision, `pass` for a resolved decision with an authored answer, or `unknown` when required information is unavailable or invalid. `affectedWork` lists each ticket that explicitly names the decision as a blocker. An open project-level decision does not block unrelated work.

The decision record's state overrides an outdated Open decisions index link. Resolved decisions keep their authored inbound `references` and answer without appearing in the text report's open list. Missing or invalid decisions leave affected checks unknown and the report partial. Known open decisions can produce a complete report with exit status `0`.

When a decision answer imposes implementation requirements, also link it through the ticket's Spec or Context section. `ctx context` retains its existing selection rules and does not automatically follow Blocked by decisions.

Readiness checks acceptance criteria, selected context, work dependencies, and blocking decisions. A prerequisite passes only when its work is completed, it has nonempty criteria, its acceptance is valid and fresh, its blocking decisions are resolved, and all of its own prerequisites pass. Known failures make readiness blocked; otherwise, unknown information takes precedence over ready. Reasons retain unknown conditions even when another condition is a known blocker.

Cycles block their members and work that depends on them. A fully observed cycle remains a complete evaluation. `ctx context` can still return complete source context for that cycle. Unrelated work remains inspectable, and a partial report can retain independently established eligible work. Incomplete inventory suppresses the shortlist because undiscovered identities may affect the result.

The shortlist preserves authored commitment order. Other work is ordered by stable ID and resolved path. The report lists eligible choices with reasons without ranking a preferred task. An empty project with all required declarations can produce a complete report.

Exit status `0` means complete evaluation, including an empty shortlist or known blocked work. Status `1` means a partial report, including a missing manifest, malformed record, unknown required fact, or collection limit. Status `2` means invalid arguments, unusable configured roots, cancellation, or an application or output failure. Reports and data diagnostics go to stdout. Invocation and operation errors go to stderr. An output failure can leave partial bytes on stdout.

### Requirement fingerprints

Each parsed work item exposes `fingerprintVersion: 1` and `ticketSHA256`. The ticket fingerprint covers its exact Markdown body after frontmatter removal, excluding every level-two Comments and Acceptance section. Nested content in those excluded sections is also excluded. Other requirements, including introductory text, Scope, criteria, and dependency links, remain covered.

`criteriaSHA256` covers each complete Acceptance criteria section in document order, including its heading and nested content. A missing section gives `null`. An explicitly empty section still has a fingerprint; its availability does not establish readiness.

Both fingerprints include effective reference definitions used by retained content when those definitions are outside the retained bytes. Definitions are appended once in first-use order. Changing a used definition in Comments changes the relevant fingerprint. Ordinary comments, current acceptance links, and frontmatter-only edits leave requirement fingerprints unchanged.

The separate `sources[].sha256` identifies the whole source file, so it changes after a comment or metadata edit. Fingerprints use the captured source bytes, preserve line endings and Unicode, and perform no filesystem rereads. They do not certify acceptance or the current code checkout.

The [acceptance authoring reference](docs/agents/acceptance.md#reproduce-version-1-fingerprints) defines the exact domain tags and byte encoding. [Conformance fixtures](internal/orientation/testdata/fingerprints/) retain manually selected input parts and literal expected digests, independently checked with Python and OpenSSL.

### Recorded acceptance

A ticket's single current Acceptance link selects its decision. Orientation validates the record's subject, attribution, fingerprints, and retained Requirements and Evidence snapshots. Missing or malformed information leaves acceptance unknown. Changed requirements or evidence make acceptance stale and block a prerequisite, while remaining a complete evaluation when every required fact is known.

Each work item's `acceptance` is `null` when unfinished work has no current decision. Otherwise, the summary contains `status`, `checkStatus`, `record`, `actor`, `decidedAt`, `testedRevision`, `fingerprintVersion`, `ticketSHA256`, `criteriaSHA256`, `humanApprovals`, `requirements`, `evidence`, `reasons`, and authored `metadata`. Status is `valid`, `stale`, or `unknown`; the corresponding check is `pass`, `fail`, or `unknown`. Recorded fingerprints in this summary remain separate from the work item's current fingerprints.

Each requirement or evidence snapshot contains its `reference`, recorded `sha256`, observed `currentSHA256`, `status`, and `reasons`. Unavailable digests are `null`. Actor entries contain `kind` and `identity`; tested revisions contain `origin` and `revision`. Each human approval has its own `actor`, `decidedAt`, and `testedRevision`.

Requirement snapshots cover the ticket's selected Spec and Context documents and linked decisions. Additional snapshots remain explicit accepted requirements. Evidence must include at least one local UTF-8 source. Allowed-source roots, collection limits, and source digests apply to both kinds of snapshot. The [acceptance authoring guide](docs/agents/acceptance.md) describes the record format and reassessment procedure.

The acceptance actor can be a human or a workflow. Optional human approvals retain their own actor, decision time, and tested revision. The report attributes these assertions without authenticating the named actors. A workflow decision does not imply human approval.

Tested revisions retain their original origin and revision value. A later Git HEAD does not invalidate unchanged historical acceptance. Orientation checks the current requirements and retained evidence against the recorded decision; it does not rerun verification or certify the current application checkout.

## Read task context

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

## Context JSON contract

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

- `cmd/ctx` wires both application operations into the CLI and exits with its status.
- `internal/cli` owns urfave/cli v3 flags, text and JSON rendering, and exit statuses. It accepts an explicit `Operations` value.
- `internal/taskcontext` exposes `Assemble(context.Context, Request) (Result, error)` and owns task-context selection.
- `internal/orientation` exposes `Orient(context.Context, Request) (Result, error)` and owns discovery and project evaluation.
- `internal/discoveryconfig` reads shared and personal ContextConfig documents, validates declarations, and retains path identities and unavailable-target diagnostics for later scope selection. Reader commands do not use it yet.
- `internal/recordread` shares Markdown and YAML parsing and authorized source reads between the application operations.

The application operation does not depend on CLI types, print output, or exit. Dependencies are pinned in `go.mod`: urfave/cli v3.12.0, Goldmark v2.1.0, goccy/go-yaml v1.19.2, go-cmp v0.7.0, and go-internal v1.16.0. Application tests use real temporary directories; testscript covers the CLI contract.

For direct Go callers, zero-valued `Request.MaxFiles` and `Request.MaxBytes` select the defaults. Negative limits return an operation error. CLI callers must use positive values when supplying either flag.
