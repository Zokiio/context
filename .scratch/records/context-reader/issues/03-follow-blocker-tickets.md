---
type: WorkItem
id: 162ad558-91e5-4c0b-b0b9-b8af2bfcc903
title: "Follow blocker tickets recursively"
triage: ready-for-agent
---

# 03: Follow blocker tickets recursively

**What to build:** A caller receives the complete selected dependency context for a ticket: blocker tickets at every depth and each ticket's linked Spec and Context sources. Shared dependencies and cycles terminate predictably, and unavailable tickets do not hide other known sources.

## Scope

Reuse the existing relationship parser, source reader, and JSON result. This is context collection, not evaluation of workflow state or readiness. Do not implement assignment, execution claims, or automatic parent traversal.

## Acceptance criteria

- [x] Follow Blocked by relationships recursively using breadth-first order. Include each visited blocker immediately followed by its Spec sources and then its Context sources before advancing to the next blocker.
- [x] Preserve authored link order within each relationship kind. Repeated and shared blocker paths do not duplicate source text, but their distinct inclusion reasons remain available.
- [x] Ticket and blocker targets stay inside the bundle, including after symlink resolution. External-document authorization, if present, does not authorize external blocker tickets.
- [x] Detect dependency cycles without unbounded traversal and report cycle warnings. A cycle with every selected source present returns complete context and exit code 0; a shared dependency without a cycle is not reported as cyclic.
- [x] An unavailable starting ticket or blocker makes both complete and traversalComplete false because its relationships cannot be discovered. Preserve available context and continue processing other known sources.
- [x] A blocker with malformed, unterminated, or non-mapping frontmatter remains in the result with its original text and digest; report incomplete discovery, skip its relationship extraction, and continue other known sources.
- [x] Missing Spec or Context documents on any selected ticket make context incomplete without independently preventing traversal from finishing. CLI exit code 1 preserves the JSON result and diagnostics.
- [x] If a file already included as a linked document is later reached as a blocker, traverse it in its ticket role without duplicating its source. Keep source deduplication distinct from whether ticket relationships have been explored.
- [x] Application tests exercise a branching dependency graph, a shared dependency, self and multi-ticket cycles, nested missing sources, malformed blocker metadata, and deterministic order. Thin CLI tests demonstrate recursive context and cycle-warning success.

## Blocked by

- [02: Include linked project documents](02-include-project-documents.md)

## Spec

- [Context reader specification](../../../context-reader/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)

## Comments

### 2026-09-16: Implementation assignment

The acceptance_review agent owns ticket 03 after ticket 02 passed its checks. Work covers breadth-first blocker collection and traversal tests in `internal/taskcontext/context.go`, `internal/taskcontext/traversal_test.go`, and `internal/cli/testdata/blockers.txt`. Ticket 04 independently owns external-source resolution.

### 2026-09-16: Completion evidence

Implemented breadth-first blocker collection with each ticket's Spec and Context sources selected before the next ticket. Source deduplication retains distinct `blocked_by` reasons independently of ticket exploration. Ticket scope is checked before deduplication, including documents already selected from an allowed external directory. Relationship parsing uses the retained source text.

The new `dependency_cycle` diagnostic is a warning with the referring file, original link, and target path. Graph reachability detects self, multi-ticket, and cross-branch cycles while shared blockers remain warning-free. Unavailable or malformed blockers preserve other branches and mark traversal incomplete; missing documents alone preserve completed traversal.

Acceptance coverage is in [application traversal tests](../../../../internal/taskcontext/traversal_test.go) and [CLI blocker tests](../../../../internal/cli/testdata/blockers.txt). Tests cover branching order, repeated and shared dependencies, symlink aliases, source digests and unchanged records, malformed metadata, nested missing/unreadable sources, document-to-ticket traversal, external blocker rejection after document inclusion, and successful cycle-warning exits.

Validation against the current working tree passed: `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...`. This evidence covers ticket 03; the milestone real-task trial remains owned by ticket 05.
