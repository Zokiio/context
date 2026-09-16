---
type: WorkItem
id: c810606f-6a64-460d-9223-8a8515fdcd65
title: "Explain blocking decisions"
triage: ready-for-agent
---

# 03: Explain blocking decisions

## What to build

Orientation shows unresolved project decisions and explains which tickets they block. A resolved decision releases its condition only when its authoritative record contains an answer.

## Scope

Evaluate Decision records selected by the project index or explicit ticket blockers. Keep detailed task-context selection unchanged. Work-prerequisite satisfaction and acceptance validation remain assigned to later tickets.

## Acceptance criteria

- [ ] Resolve Decision records from Open decisions and Blocked by decisions links within the selected bundle. Validate stable identity, title, and decisionState using the agreed open and resolved values. Reuse captured source bytes, ordering, limits, and provenance.
- [ ] An open decision fails the decision check only for tickets that explicitly link it as blocking. A project-level open decision without that relationship does not block otherwise eligible work.
- [ ] A resolved decision requires a nonempty authored Resolution section. A valid answer passes that decision's condition; missing state, invalid state, or a missing answer makes the effect unknown.
- [ ] The decision record owns its state. An outdated Open decisions index link must not turn a resolved record into an open blocker. Preserve the reference without misrepresenting its authoritative state.
- [ ] Missing, unreadable, ambiguous, wrong-type, or out-of-bundle decision targets produce identifying diagnostics and unknown affected checks. Allowed document roots do not authorize external Decision records.
- [ ] Aggregate multiple decisions without discarding unknown conditions when another decision is a known blocker. Continue evaluating unrelated work. Known open decisions alone can produce a complete report and exit 0; unavailable required information produces exit 1.
- [ ] Text and JSON show the authoritative decision state, affected ticket references, readiness effects, and sources. Resolving a blocking decision can add a ticket to the shortlist only when its other eligibility conditions pass.
- [ ] Verify the authoring guidance requires implementation-relevant decision answers to be linked through Spec or Context as well. Do not silently extend the existing task-context traversal rules.
- [ ] Public-operation tests cover open and resolved decisions, absent answers, repeated references, stale index entries, unlinked project decisions, mixed known and unknown blockers, duplicate IDs, and external targets. Small CLI assertions verify decisions and their effects in both formats.
- [ ] Retain actual verification evidence and tested revision using the established completion procedure.

## Blocked by

- [02: List eligible work with reasons](02-list-eligible-work.md)

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
