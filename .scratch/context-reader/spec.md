# Assemble context for an explicit work item

Status: ready-for-agent

## Problem Statement

Starting a coding agent on a known work item requires the user to gather requirements, dependencies, and relevant project documents by hand. Missing context can cause repeated investigation or work that contradicts an established decision. Passing an entire repository makes the relevant material harder to identify.

The first user is the project's author. A representative task is delivering backend configuration to a device within a larger Wi-Fi feature. The backend ticket needs its own requirements, blocker tickets, and explicitly linked project documents. This feature addresses collecting those sources, rather than the entire grooming-to-merge workflow.

## Solution

Provide a read-only CLI backed by a reusable Go context-assembly operation. Given an explicit project directory and ticket path, assemble the ticket, its recursively selected blockers, and each selected ticket's Spec and Context sources.

Return whole source files, their paths, and reasons for inclusion in structured output. Read current on-disk content, including uncommitted changes and new files. If selected content is unavailable or exceeds configured limits, return available context marked incomplete, explain omissions, and exit unsuccessfully.

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

## Implementation Decisions

### Application boundary

- A Go application operation owns selection, recursive blocker traversal, deduplication, completeness, limits, and diagnostics.
- The CLI resolves explicit project scope, invokes that operation, renders its result, and maps the outcome to an exit status.
- Filesystem reading and Markdown relationship extraction support the operation. Internal package layout and library choices are not fixed by discovery.
- No process-global current project, required server, external tracker dependency, or LLM dependency is introduced.

### Records and relationships

- Respect one OKF bundle per project, independently of Git repository boundaries, as established by the bundle ADR.
- Preserve stable project and work-item identities. This feature locates its starting ticket by the caller-supplied path, without adding identity-based lookup.
- Spec, blocker, and Context relationships are Markdown file links in named body sections, as established by the relationship ADR.
- Select the starting ticket, follow blockers recursively, and include each selected ticket's Spec and Context links.
- A Spec section is optional. A broken link in a present Spec section makes the result incomplete.
- Other links in document prose remain references. Including a document does not select every document linked from its prose.
- Include each selected file's source text once, with an explanation of its selection.
- Automatic parent-chain traversal is later work. A needed parent document can be linked explicitly as context. Siblings are not implicitly selected.

### Filesystem and content

- Read current on-disk files, including uncommitted edits and new files. Do not substitute committed versions.
- The project directory locates records and need not be a Git root. The reader's repository is not implicitly the selected project.
- Caller-supplied allowed directories permit external context. Explicitly linked plain Markdown sources in those directories are supported.
- Preserve whole source files. Do not summarize, rewrite, or truncate them to fit limits.
- Reading does not write records, migrate content, or initialize project metadata.

### Result and failure contract

- Structured output contains source text, paths, inclusion reasons, completeness, and diagnostics.
- Completeness means every selected source is included. It is separate from OKF validity, work readiness, and whether authored links identify every relevant document.
- Missing or unreadable selected files produce available context, incomplete status, identifying diagnostics, and a nonzero exit status.
- Dependency cycles produce warnings. If all selected sources are included, the result remains complete and succeeds.
- File-count and total-source-byte limits are configurable. Limit omissions are explicit and make the result incomplete with a nonzero exit status.
- Invalid or ambiguous invocation fails clearly and noninteractively instead of guessing a remembered project.

### Implementation latitude

Discovery settled the behaviors above, but did not choose exact heading grammar, path syntax, output field names, diagnostic codes, limit defaults, or command names. These remain implementation choices, not previously approved contracts.

The implementing agent must define and document these conventions consistently with this spec and the ADRs before dependent callers are built. In particular, specify malformed-link handling, fragments, path containment and symlinks, deterministic selection order, byte accounting, and exact exit codes. Test their observable boundary behavior. Changing the agreed selection or failure behavior requires a specification change.

## Testing Decisions

### Main test boundary

Exercise the complete context-assembly operation against real temporary project directories. Explicit project scope, a starting ticket, allowed directories, and limits go in; selected sources, reasons, completeness, and diagnostics come out.

Use real Markdown and filesystem relationships for normal cases. Do not mock the parser or traversal, which would bypass the behavior being tested. A narrow controllable read failure may be needed for reliable unreadable-source tests where permission changes do not prevent reads.

Add a small set of CLI tests for argument handling, structured-output parsing, and exit status. Do not repeat the entire traversal suite through the CLI or expose each internal helper as a separate testing interface.

The testing-boundary check was requested during spec synthesis. Its confirmation is pending.

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
| Shared document or repeated blocker | Include source text once and explain selection |
| Ordinary prose link | Do not follow it as a selection relationship |
| Explicitly linked parent document | Include it through the ordinary Context relationship |
| Unlinked parent or sibling | Do not import it automatically |
| Missing linked specification | Return available context, incomplete status, identifying diagnostic, and nonzero exit status |
| Unreadable selected source | Preserve available sources and report incompleteness |
| Cycle with all sources available | Terminate, warn, and return complete context with success |
| Uncommitted edit and new linked file | Return current on-disk contents |
| Allowed external plain Markdown | Include text, path, and reason |
| External source outside allowed scope | Do not read beyond supplied scope or silently claim complete selected context |
| File-count limit | Preserve whole files, report omissions, and return incomplete context |
| Total-source-byte limit | Preserve whole files, report omissions, and return incomplete context |
| Second project | Select its records without importing the reader repository's records |
| Successful and incomplete reads | Leave source records unchanged and create no project metadata |
| Invalid or ambiguous CLI input | Fail clearly without prompting or guessing scope |
| Structured CLI output | Preserve source text, reasons, completeness, and diagnostics from the application result |

### Prior art

The repository contains documentation, ADRs, skills, and the first-reader draft, but no application implementation or test suite. There is no existing code-level testing boundary to reuse. The application operation is the highest useful shared boundary for this feature; CLI tests cover the remaining user-facing contract.

## Out of Scope

- Session orientation, task discovery, and choosing ready work.
- Jira retrieval, parent-chain traversal, tracker synchronization, and write-back.
- Pickup triggers, readiness calculation, preparation policies, and implementation claims.
- Grooming execution, semantic review, and automatic missing-link discovery.
- Record editing, safe-write coordination, handoff capture, and requirement-change monitoring.
- Agent launching, scheduling, cancellation, or a custom runtime.
- Deployment, review collection, merge execution, and completion evaluation.
- A TUI, web interface, server, and organizational standards distribution.
- Full OKF validation, generated summaries, relevance ranking, and implicit context expansion.

## Further Notes

- This spec implements the first reader in the [product vision](../../docs/vision.md), using the [domain glossary](../../CONTEXT.md).
- Governing decisions are [one OKF bundle per project](../../docs/adr/0001-one-okf-bundle-per-project.md) and [work relationships in Markdown sections](../../docs/adr/0002-work-relationships-in-markdown-sections.md).
- This document is the authoritative feature spec under the [local tracker conventions](../../docs/agents/issue-tracker.md). Update it in place rather than creating a duplicate backlog.
- The device Wi-Fi scenario motivates the product; this milestone requires a known ticket and explicit links rather than implementing the complete workflow.
- Skills use the reader as it becomes available and remain responsible for authored edits. A context result is not an instruction-enforcement mechanism.
- Exact conventions listed under Implementation Latitude must be made concrete during implementation. The requested triage status does not imply that those conventions were decided during discovery.
