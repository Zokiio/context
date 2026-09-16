---
type: WorkItem
id: e8f40c26-843e-4983-a7c4-279aed609f6f
title: "Bound collection and verify a real-task workflow"
triage: ready-for-agent
---

# 05: Bound collection and verify a real-task workflow

**What to build:** A caller can bound the reader's output without receiving truncated files or misleading completeness claims. The completed reader is then exercised through a skill on a real development task, recording whether it delivers the selected requirements and dependencies without manual collection.

## Scope

This ticket closes the first-reader milestone. Its dependency on external-context support is for the real-task trial, since this project's tickets link to specs and context outside the bundle. It does not add automatic context discovery or a custom agent runtime.

## Acceptance criteria

- [x] Accept optional --max-files and --max-bytes flags with defaults of 100 files and 1,048,576 source bytes. Require positive integers and reject zero, negative, or malformed values with exit code 2 and an invocation error.
- [x] Count the actual bytes read for each included file before JSON encoding, counting deduplicated sources once. Digests and returned text are derived from those same source bytes.
- [x] Apply limits to the established order: starting ticket, its Spec and Context documents, then breadth-first blockers with their documents. Stop adding sources at the first limit breach instead of skipping a large source to fit later smaller sources.
- [x] Include only whole files. If the starting ticket alone exceeds the byte limit, return no sources, an identifying diagnostic, both completeness fields false, and exit code 1.
- [x] Report the source that breached the limit and already-known pending sources. Mark traversalComplete false, and do not claim an exhaustive list or count of unknown downstream omissions.
- [x] Preserve available sources and diagnostics in the JSON result on limit failures. Missing linked documents, unavailable tickets, cycles, and frontmatter failures retain their distinct behavior from the earlier slices.
- [x] Behavioral tests cover exact-fit and first-overflow cases for both limits, multibyte source text and JSON escaping, repeated sources counted once, an oversized first ticket, and a blocker whose undiscovered descendants are not falsely enumerated.
- [x] Use a small CLI testscript suite to verify defaults, overrides, invalid flags, result fields, and exit codes. Run the complete required application and CLI checks before the milestone trial.
- [x] Use the reader through a skill for one real development task. The skill invokes the reader with explicit project, ticket, and allowed-source scope and supplies the result to the coding agent. Verify delivery of the task's linked requirements, blockers, and context without the user gathering them manually.
- [x] Link acceptance evidence from this ticket recording the real task, invocation, reader revision, result completeness, and observed outcome. Record relevant documents lacking authored links separately from reader defects; do not claim automatic relevance discovery.
- [x] All required tests and the real-task trial pass before this ticket and the first-reader milestone are considered complete. The trial must use actual recorded results rather than a fixture-only demonstration.

## Blocked by

- [03: Follow blocker tickets recursively](03-follow-blocker-tickets.md)
- [04: Allow authorized external context](04-allow-external-context.md)

## Spec

- [Context reader specification](../../../context-reader/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)

## Comments

- 2026-09-16: Assigned limits implementation to agent `requirements_review` after tickets 03 and 04 passed their checks. This assignment covers application limits, CLI flags, and behavioral tests. The coordinating agent owns the README, skill, and real-task acceptance trial. Trial criteria remain pending until recorded evidence is available.

- 2026-09-16 code verification: Implemented defaults of 100 files and 1,048,576 source bytes, positive CLI overrides, and application defaults when limits are omitted. Limits count each included resolved source once, preserve whole source text, and stop at the first breach. `source_limit_exceeded` identifies that source. `source_omitted` identifies unique known pending sources not already included, with referring paths and authored links. Both completeness fields become false. Current-ticket blockers are queued before its documents are read, so their known omissions remain visible.
- Verification passed against the uncommitted working tree based on `0c74f92506a67242247a4e76a40785561a561bb3`: `go test ./...`, `go vet ./...`, `go test -race ./...`, and `go build -o /tmp/context-ticket05 ./cmd/ctx`. Tests cover exact fit and overflow for both limits, default boundaries, byte counts with multibyte and JSON-escaped text, deduplication with cycles, oversized initial tickets, first-breach stopping, queued blocker omissions, undiscovered descendants, retained earlier errors, and CLI defaults, overrides, invalid values, JSON, and exits.
- Code/test ownership released. The last three acceptance criteria remain pending for the coordinating agent's real-task skill trial and recorded evidence. No commit or milestone completion is claimed by this code handoff.


### 2026-09-16 real-task acceptance

Agent `/root/milestone_trial` applied the [task-context skill](../../../../.agents/skills/task-context/SKILL.md) to this ticket after the required application and CLI checks passed. The reader delivered all 11 authored sources as full text, with complete context and matching digests. Seventeen real-task invocations verified defaults, exact-fit limits, first overflow, known omissions, and invalid flags. The independent verification found no defects.

[Acceptance evidence](../../../context-reader/acceptance-trial.md) records the exact commands, observed outcomes, and relevant unlinked documents. Its [digest manifest](../../../context-reader/acceptance-trial-manifest.json) identifies the uncommitted reader and delivered source revisions. The trial captured this ticket before this completion note and checklist update, so its recorded byte boundaries apply to that snapshot.

All acceptance criteria for this ticket and the first-reader milestone now have local evidence. The implementation and records remain uncommitted; no merge or external approval is claimed.


### 2026-09-16 verification after independent review

After the blocker-reference fix, all 17 real-task checks passed again against the updated reader. Full race tests, vet, build, and the Go 1.25.0 test suite also passed. The [review follow-up](../../../context-reader/acceptance-trial.md#independent-review-follow-up-2026-09-16) and [revised source manifest](../../../context-reader/review-verification-manifest.json) preserve this evidence separately from the original trial.
