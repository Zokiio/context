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

- [ ] Accept optional --max-files and --max-bytes flags with defaults of 100 files and 1,048,576 source bytes. Require positive integers and reject zero, negative, or malformed values with exit code 2 and an invocation error.
- [ ] Count the actual bytes read for each included file before JSON encoding, counting deduplicated sources once. Digests and returned text are derived from those same source bytes.
- [ ] Apply limits to the established order: starting ticket, its Spec and Context documents, then breadth-first blockers with their documents. Stop adding sources at the first limit breach instead of skipping a large source to fit later smaller sources.
- [ ] Include only whole files. If the starting ticket alone exceeds the byte limit, return no sources, an identifying diagnostic, both completeness fields false, and exit code 1.
- [ ] Report the source that breached the limit and already-known pending sources. Mark traversalComplete false, and do not claim an exhaustive list or count of unknown downstream omissions.
- [ ] Preserve available sources and diagnostics in the JSON result on limit failures. Missing linked documents, unavailable tickets, cycles, and frontmatter failures retain their distinct behavior from the earlier slices.
- [ ] Behavioral tests cover exact-fit and first-overflow cases for both limits, multibyte source text and JSON escaping, repeated sources counted once, an oversized first ticket, and a blocker whose undiscovered descendants are not falsely enumerated.
- [ ] Use a small CLI testscript suite to verify defaults, overrides, invalid flags, result fields, and exit codes. Run the complete required application and CLI checks before the milestone trial.
- [ ] Use the reader through a skill for one real development task. The skill invokes the reader with explicit project, ticket, and allowed-source scope and supplies the result to the coding agent. Verify delivery of the task's linked requirements, blockers, and context without the user gathering them manually.
- [ ] Link acceptance evidence from this ticket recording the real task, invocation, reader revision, result completeness, and observed outcome. Record relevant documents lacking authored links separately from reader defects; do not claim automatic relevance discovery.
- [ ] All required tests and the real-task trial pass before this ticket and the first-reader milestone are considered complete. The trial must use actual recorded results rather than a fixture-only demonstration.

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
