# Assemble context for an explicit work item

Status: ready-for-agent

## Problem Statement

Starting a coding agent on a known work item requires the user to gather requirements, dependencies, and relevant project documents by hand. Missing context can cause repeated investigation or work that contradicts an established decision. Passing an entire repository makes the relevant material harder to identify.

The first user is the project's author. A representative task is delivering backend configuration to a device within a larger Wi-Fi feature. The backend ticket needs its own requirements, blocker tickets, and explicitly linked project documents. This feature addresses collecting those sources, rather than the entire grooming-to-merge workflow.

## Solution

Provide a read-only CLI backed by a reusable Go context-assembly operation. Given an explicit project directory and ticket path, assemble the ticket, its recursively selected blockers, and each selected ticket's Spec and Context sources.

Return whole source files, their paths, and reasons for inclusion as one versioned JSON result. OKF remains the on-disk record format; JSON packages the reader's result without rewriting those records. Read current on-disk content, including uncommitted changes and new files. If selected content is unavailable or exceeds configured limits, return available context marked incomplete, explain omissions, and exit unsuccessfully.

The user can pass this result to an existing coding agent. No server, external tracker, or LLM connection is required. Skills continue to own edits during this milestone.

## User Stories

1. As a developer, I want to select a project explicitly, so that the reader does not silently use a different project.
2. As a developer, I want to select a specific ticket, so that context corresponds to the work I intend to do.
3. As a developer, I want project records outside a Git root to work, so that record location does not depend on repository layout.
4. As a developer, I want to invoke the reader from outside the selected project, so that invocation location does not change which project it reads.
5. As a coding agent, I want the starting ticket's source text, so that I can read its requirements and acceptance criteria.
6. As a developer, I want a self-contained ticket to work without a separate specification, so that small work items do not need duplicate documents.
7. As a coding agent, I want direct blocker tickets included, so that dependencies on other work are visible.
8. As a coding agent, I want blockers followed recursively, so that indirect dependencies are visible too.
9. As a coding agent, I want the starting ticket's linked specifications included, so that I can understand its intended behavior.
10. As a coding agent, I want selected blocker tickets' linked specifications included, so that their requirements are available too.
11. As a coding agent, I want every selected ticket's Context links included, so that relevant project documents accompany the work.
12. As a developer, I want ordinary prose links to remain references, so that selection does not expand through unrelated documents.
13. As a developer, I want to allow specific external source directories, so that project context can live outside the OKF bundle.
14. As a developer, I want linked plain Markdown context to be readable, so that existing documentation need not be converted first.
15. As a developer, I want external context access to respect the allowed directories, so that it stays within the scope I supplied.
16. As a coding agent, I want each selected file included once, so that repeated references do not consume duplicate context.
17. As a coding agent, I want each source's file path, so that I can locate its original document.
18. As a coding agent, I want a reason for each source's inclusion, so that I can understand its relationship to the task.
19. As a developer, I want whole source text rather than generated summaries, so that selection preserves the author's wording.
20. As a developer, I want uncommitted edits reflected in the result, so that an agent sees current documents.
21. As a developer, I want newly created linked files included, so that a commit is not required before using context.
22. As a coding agent, I want a completeness indicator, so that I know when selected information is missing.
23. As a developer, I want available context returned when a selected file is missing, so that one broken relationship does not hide the rest.
24. As a developer, I want unreadable sources identified, so that I can repair access or the source itself.
25. As a developer, I want a broken Spec link to make the result incomplete, so that missing requirements are not silently accepted.
26. As a developer, I want dependency cycles reported without endless traversal, so that I can inspect context and repair the cycle.
27. As a caller, I want a cycle warning to preserve success when all selected sources are present, so that completeness remains distinct from relationship quality.
28. As a developer, I want a configurable file-count limit, so that I can bound context size.
29. As a developer, I want a configurable total-source-byte limit, so that I can control the amount of returned source content.
30. As a coding agent, I want limits to preserve whole files and identify omissions, so that I do not receive unexplained fragments of requirements.
31. As an automated caller, I want structured output and clear exit behavior, so that I can distinguish complete results, incomplete results, and invocation failures.
32. As an automated caller, I want invalid or ambiguous input to fail without prompting, so that unattended calls do not hang or guess scope.
33. As a developer, I want reading context to leave records unchanged, so that inspection does not modify project knowledge.
34. As a developer, I want a second fixture project to work, so that the reader is not specific to its own repository.
35. As a developer, I want completeness to describe selected-source coverage, so that it does not falsely claim validity, readiness, or sufficient understanding.
36. As a future interface author, I want the same Go application operation to be callable independently of the CLI, so that selection rules are not duplicated.
37. As an automated caller, I want a versioned JSON result, so that I can interpret the response without treating it as an OKF storage format.
38. As a developer, I want repeated references to retain their distinct inclusion reasons, so that deduplication does not hide relationships.
39. As a caller, I want unfinished traversal distinguished from missing content, so that I know when the reader could not discover all relationships.
40. As a developer, I want file fragments to preserve whole-file context, so that heading links do not silently narrow the returned source text.
41. As a caller, I want a content digest for each source, so that I can identify the exact bytes read, including uncommitted edits.

## Implementation Decisions

### Application boundary

- A Go application operation owns selection, recursive blocker traversal, deduplication, completeness, limits, and diagnostics.
- The CLI resolves explicit project scope, invokes that operation, renders its result, and maps the outcome to an exit status.
- Start with one Go module and internal packages shared by the product's interfaces. A public Go library contract is outside this milestone.
- A command entry point wires dependencies. A CLI package owns arguments, rendering, and exit codes. A context-assembly package owns the reader operation, keeping document parsing and filesystem details private initially.
- Extract shared record-reading code when another operation needs it. Do not create empty packages for future integrations.
- Use Goldmark v2 for Markdown and goccy/go-yaml for YAML frontmatter. Keep frontmatter delimiter handling in the private document reader. Strict syntax parsing must allow unknown metadata fields; do not enable unknown-field rejection through typed decoding.
- Use the Go standard library for filesystem access and JSON, including `os.Root` for reads constrained to an allowed directory tree. The resolver interprets document links and selects the permitted root before the reader opens a source.
- Use urfave/cli v3 for command and flag declarations. Keep framework types inside the CLI package and construct commands with explicit dependencies. Defer cli-altsrc until configuration becomes a defined feature.
- No process-global current project, required server, external tracker dependency, or LLM dependency is introduced.

### Frontmatter and body parsing

- The private document reader separates YAML frontmatter from the Markdown body before relationship extraction. Use YAML parsing as well as Markdown parsing; text inside frontmatter must never become a body relationship.
- Preserve the original full source text independently of parsed metadata. Do not regenerate returned text by serializing YAML or Markdown.
- Match the upstream reference parser's strict handling of malformed or unterminated frontmatter. Frontmatter must parse as a YAML mapping. Tolerate unknown metadata keys and concept types, and continue accepting plain Markdown context documents.
- If a ticket's frontmatter cannot be parsed, retain its source text in the result, report the error, and mark relationship discovery and context incomplete. Do not guess relationships from that ticket. Continue processing other known sources.
- Parsing is separate from full OKF validation. Successfully extracting metadata and body content does not certify the document's schema or the project bundle.

### Records and relationships

- Respect one OKF bundle per project, independently of Git repository boundaries, as established by the bundle ADR.
- New implementation tickets use the [minimal tracker profile](../../docs/agents/issue-tracker.md#minimal-implementation-ticket-profile). This project's bundle is `.scratch/records/`, with a root `project.md` and tickets under `context-reader/issues/`. This spec stays at its existing path outside the bundle and is selected through an explicitly allowed source directory.
- Preserve stable project and work-item identities. This feature locates its starting ticket by the caller-supplied path, without adding identity-based lookup.
- Relationship sections use exactly the level-two headings `Spec`, `Blocked by`, and `Context`, as a concrete convention within the relationship ADR. Other headings are not interpreted as relationship kinds.
- A recognized section ends at the next level-one or level-two heading. Nested subsections remain inside it. Combine repeated recognized sections in document order.
- Recognize inline and reference-style Markdown links in relationship sections. Ignore images, links inside code, and apparent headings inside code fences. An unresolved reference-style link in a recognized section produces a diagnostic and an incomplete result.
- Select the starting ticket, follow `Blocked by` relationships recursively, and include each selected ticket's Spec and Context links.
- A Spec section is optional. A broken link in a present Spec section makes the result incomplete.
- Other links in document prose remain references. Including a document does not select every document linked from its prose.
- Deduplicate by resolved absolute path. Retain every distinct inclusion reason, including the referring file and original link. The requested ticket has a root-selection reason. Separate files are not merged merely because their contents match.
- A fragment selects the whole target file. Preserve the original link, including its fragment, for provenance. Heading-existence validation is outside this milestone.
- The reader loads local files only. HTTP links in relationship sections produce an unsupported-source diagnostic and an incomplete result. Web links in ordinary prose remain references. Do not make network requests.
- Automatic parent-chain traversal is later work. A needed parent document can be linked explicitly as context. Siblings are not implicitly selected.

### Filesystem and content

- Read current on-disk files, including uncommitted edits and new files. Do not substitute committed versions.
- Include a SHA-256 digest of the exact bytes read for each returned source, computed from the same bytes used for its returned text. Git metadata is deferred. Per-source digests do not imply an atomic snapshot across files.
- Resolve the CLI's relative project directory against the caller's working directory, the relative starting ticket against the project directory, and relative document links against their containing document's directory.
- Resolve relative `--allow-source` directories against the caller's working directory. A document link beginning with `/` resolves from the selected bundle root, not the operating-system root. External documents use ordinary relative links and remain subject to the allowed-directory checks.
- Require an existing project directory, which locates records and need not be a Git root. The reader's repository is not implicitly the selected project.
- Ticket and blocker files must remain inside the project bundle. Linked documents may be outside the bundle only within caller-approved source directories. Check resolved paths, including symlinks, against these boundaries. Explicitly linked plain Markdown sources in allowed directories are supported.
- Do not require full OKF validation or automatically include the project manifest. Bundle structure remains governed by its ADR; reading selected sources does not certify that structure.
- Preserve whole source files. Do not summarize, rewrite, or truncate them to fit limits.
- Reading does not write records, migrate content, or initialize project metadata.

### CLI contract

- Provide a `context` subcommand. The executable's name remains undecided.
- Require `--project` and `--ticket`.
- Accept repeatable `--allow-source` arguments for external source directories.
- Disable slice-value separator handling so each allowed-directory argument remains literal, including paths containing commas. Do not attach environment or configuration fallbacks to the explicitly supplied project, ticket, or allowed-directory flags.
- Accept optional `--max-files` and `--max-bytes` arguments. Both require positive integers; zero, negative, and malformed values are invocation errors.
- Write one JSON result to standard output. JSONL streaming and a separate human-readable rendering are outside this milestone.
- Report invocation errors and failures to run the operation on standard error. Do not mix human commentary into the JSON output.
- Configure urfave's output writers, usage-error handling, and exit handling to preserve this output contract and the defined exit codes. The application operation returns results without printing or exiting the process.

### Selection order and limits

- Select the starting ticket first, followed by its Spec sources and then its Context sources.
- Visit blocker tickets breadth-first. After each blocker ticket, include its Spec sources and then its Context sources before advancing to the next blocker. Preserve link order within each relationship kind, including repeated sections.
- Default to 100 files and 1,048,576 bytes of source content. Count actual file bytes before JSON encoding, counting each included file once.
- Stop adding sources at the first limit breach. Do not skip a large source to make room for smaller sources. Preserve whole files.
- Report the source that breached the limit and any already-known pending sources. Mark traversal unfinished. Do not claim an exhaustive list or count of omitted sources when their relationships were not discovered.
- If the starting ticket alone exceeds the byte limit, return no sources and explain the incomplete result.

### JSON result contract

| Field | Meaning |
| --- | --- |
| `schemaVersion` | Integer, initially `1` |
| `complete` | Whether all selected sources were included |
| `traversalComplete` | Whether relationship discovery finished |
| `sources` | Included sources, each with its resolved absolute path, original text, SHA-256 content digest, and distinct inclusion reasons |
| `diagnostics` | Diagnostics with a stable code, severity, message, and relevant source/link information |

- Completeness is separate from OKF validity, work readiness, and whether authored links identify every relevant document.
- A missing or unreadable Spec or Context document alone makes `complete` false while traversal still finishes, because its relationships are not followed. An unavailable starting ticket or blocker also makes `traversalComplete` false because its relationships cannot be discovered. Continue processing other known sources. A limit that stops discovery makes both fields false.
- Missing or unreadable selected sources produce available context with identifying diagnostics. An absent starting ticket is a missing selected source, not an invocation error.
- Dependency cycles produce warnings. If all selected sources are included, the result remains complete and succeeds.

### Exit contract

| Exit code | Meaning and output |
| --- | --- |
| `0` | Complete JSON context result, including results with cycle warnings |
| `1` | Incomplete JSON result containing available sources and diagnostics |
| `2` | Invalid invocation or failure to run the operation, explained on standard error |

Missing required CLI arguments are invocation errors. Invalid or ambiguous invocation fails noninteractively instead of guessing a remembered project.

### Implementation choices

Exact package names, dependency version pins, the executable name, diagnostic code identifiers, and nested JSON field naming remain implementation choices. Package responsibilities and selected dependencies are defined above. Document the implemented schema and codes and test them as public contracts. These choices must preserve the behavior above and the existing ADRs.

## Testing Decisions

### Main test boundary

Exercise the complete context-assembly operation against real temporary project directories. Explicit project scope, a starting ticket, allowed directories, and limits go in; selected sources, reasons, completeness, and diagnostics come out.

Use real Markdown and filesystem relationships for normal cases. Do not mock the parser or traversal, which would bypass the behavior being tested. A narrow controllable read failure may be needed for reliable unreadable-source tests where permission changes do not prevent reads.

Add a small set of CLI tests for argument handling, structured-output parsing, and exit status. Do not repeat the entire traversal suite through the CLI or expose each internal helper as a separate testing interface.

Use Go's testing package with go-cmp for structured result comparisons and testscript for the small CLI suite. Keep normal application tests on real temporary project directories rather than substituting fake readers or adding a mocking framework.

The user confirmed this testing boundary during the reader-contract review.

### Test quality

- Assert observable results, not internal calls, parser nodes, or private data structures.
- Use small fixture projects with explicit expected sources and reasons.
- Exercise a second project from outside its directory to expose assumptions about working directory and repository layout.
- Test limits at boundaries, checking whole-file preservation and explicit omissions.
- Verify that source records remain unchanged after successful and incomplete runs.
- Keep tests independent of a model, network service, or external tracker.

### Required behavioral coverage

| Scenario | Expected behavior |
| --- | --- |
| Self-contained ticket | Include its text; no Spec section is required |
| Recursive blockers | Include selected tickets and each ticket's Spec and Context sources |
| Shared document, repeated blocker, or symlink alias | Include source text once by resolved path and retain distinct referring files and original links |
| Separate files with identical contents | Keep separate sources |
| Repeated relationship headings and nested subsections | Follow the agreed section boundaries and preserve link order |
| Inline and reference-style links | Select both forms; an unresolved reference in a recognized section makes the result incomplete |
| Images, code links, and fenced heading examples | Do not treat them as relationships |
| YAML frontmatter containing apparent headings or links | Extract relationships only from the Markdown body |
| Valid frontmatter with unknown fields or types | Accept metadata without dropping or rewriting original source text |
| Malformed, unterminated, or non-mapping ticket frontmatter | Preserve the source, report incomplete discovery and context, and continue other known sources |
| Fragment link | Include the whole file, preserve the original link, and do not validate heading existence |
| HTTP link in a relationship section | Report an unsupported source and incomplete context without network access |
| Ordinary prose link | Do not follow it as a selection relationship |
| Explicitly linked parent document | Include it through the ordinary Context relationship |
| Unlinked parent or sibling | Do not import it automatically |
| Missing linked specification | Return available context, incomplete status, identifying diagnostic, and nonzero exit status |
| Unreadable selected source | Preserve available sources and report incompleteness |
| Cycle with all sources available | Terminate, warn, and return complete context with success |
| Uncommitted edit and new linked file | Return current on-disk contents and a SHA-256 digest matching each source's exact bytes |
| Allowed external plain Markdown | Include text, path, and reason |
| External source outside allowed scope | Do not read beyond supplied scope or silently claim complete selected context |
| Blocker outside the bundle, even in an allowed document directory | Do not traverse it as a ticket |
| Symlink outside allowed scope | Enforce containment using the resolved target |
| Relative project, ticket, and document links | Resolve each against its specified base |
| Repeated allowed-directory flags with commas in paths | Preserve each argument as one literal directory path |
| Relative allowed-source directory | Resolve against the caller's working directory, including when it differs from the selected project |
| Document link beginning with `/` | Resolve from the selected bundle root and enforce resolved-path containment |
| File-count limit | Preserve whole files, stop at first breach, report known omissions, and mark traversal unfinished |
| Total-source-byte limit | Count actual source bytes once before JSON encoding and stop at the first breach |
| Starting ticket exceeds byte limit | Return no sources and an incomplete result with a diagnostic |
| Stable selection order | Starting ticket and its documents precede breadth-first blockers and their documents |
| Default and overridden limits | Apply 100 files and 1,048,576 bytes by default; accept positive overrides |
| Zero, negative, or malformed limits | Return exit code 2 and explain the invocation error |
| Second project | Select its records without importing the reader repository's records |
| Successful and incomplete reads | Leave source records unchanged and create no project metadata |
| Invalid or ambiguous CLI input | Fail clearly without prompting or guessing scope |
| Missing selected source versus interrupted traversal | Distinguish completeness from completion of relationship discovery |
| Missing or unreadable starting ticket or blocker | Return available context with both completeness fields false and continue other known sources |
| Missing Spec or Context document alone | Return incomplete context with traversal complete when all ticket relationships were discovered |
| Structured CLI output | Return one JSON result with schema version 1 and preserve source text, digests, reasons, completeness, traversal status, and diagnostics |
| CLI exit status | Return 0 for complete results, 1 for incomplete results, and 2 for invocation/execution failures |
| Project manifest not selected | Do not automatically include it or require full OKF validation |

### Prior art

The repository contains documentation, ADRs, skills, and the first-reader draft, but no application implementation or test suite. There is no existing code-level testing boundary to reuse. The application operation is the highest useful shared boundary for this feature; CLI tests cover the remaining user-facing contract.

### Real-task acceptance

Complete the milestone only after the required tests pass and one real development task uses the reader through a skill. The skill invokes the reader and supplies its result to the coding agent. Verify that the agent receives the ticket's linked requirements, blockers, and documents without the user collecting them manually.

Record the task path, invocation, reader revision, result completeness, and observed outcome as acceptance evidence linked from the implementation ticket. Record relevant documents that lack authored links separately from reader defects. Automatic discovery of those documents remains outside this milestone.

## Out of Scope

- Session orientation, task discovery, and choosing ready work.
- Jira retrieval, parent-chain traversal, tracker synchronization, and write-back.
- Pickup triggers, readiness calculation, preparation policies, and implementation claims.
- Grooming execution, semantic review, and automatic missing-link discovery.
- Record editing, safe-write coordination, handoff capture, and requirement-change monitoring.
- Agent launching, scheduling, cancellation, or a custom runtime.
- Deployment, review collection, merge execution, and completion evaluation.
- A TUI, web interface, server, and organizational standards distribution.
- Full OKF validation, fragment-heading validation, generated summaries, relevance ranking, and implicit context expansion.
- JSONL streaming, network retrieval of linked documents, and a separate human-readable output format.

## Further Notes

- Keep [Koanf](https://github.com/knadh/koanf) as a candidate for later configuration shared by the CLI, TUI, and HTTP interface. It can combine defaults, files, environment variables, and explicit overrides under a defined precedence. Consider cli-altsrc for simpler flag-backed configuration. Neither is a first-reader dependency; choose when configuration requirements are specified.
- This spec implements the first reader in the [product vision](../../docs/vision.md), using the [domain glossary](../../CONTEXT.md).
- Governing decisions are [one OKF bundle per project](../../docs/adr/0001-one-okf-bundle-per-project.md) and [work relationships in Markdown sections](../../docs/adr/0002-work-relationships-in-markdown-sections.md).
- This document is the authoritative feature spec under the [local tracker conventions](../../docs/agents/issue-tracker.md). Update it in place rather than creating a duplicate backlog.
- The device Wi-Fi scenario motivates the product; this milestone requires a known ticket and explicit links rather than implementing the complete workflow.
- Skills use the reader as it becomes available and remain responsible for authored edits. A context result is not an instruction-enforcement mechanism.
- The user confirmed the reader contracts and testing boundary after the initial spec was published. The Implementation Decisions section now records those agreements rather than leaving them as open contract choices.
