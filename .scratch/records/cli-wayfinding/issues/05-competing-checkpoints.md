---
type: WorkItem
id: 94acbf45-cc6b-4957-b3fe-3218e955ef3f
title: "Handle competing and interrupted checkpoints"
triage: ready-for-agent
execution: completed
---

# Handle competing and interrupted checkpoints

## What to build

Extend resume to evaluate complete checkpoint histories, expose competing candidates, and remain explicit about invalid or incomplete observations without selecting a false winner.

## Slice boundary

This completes the read-side recovery contract. Skills, rather than the CLI, perform the authoring and deliberate recovery procedures in the following ticket.

## Acceptance criteria

- [x] Evaluate all observed identity-matching notes as the directed predecessor graph specified by the contract. Preserve earlier observations, find all leaves only after complete valid inspection, and select candidates in lexical ID order without using timestamps.
- [x] Support linear history, multiple roots, competing successors, and explicit reconciliation referencing all incorporated candidates. Multiple valid leaves produce conflicting recovery with a separate comparison per candidate; a fully evaluated conflict may return exit status 0.
- [x] Dangling predecessors, self-links, duplicate identities, cycles, unreadable/unfinished notes, and incomplete inventory return their specified graph status and diagnostics with no selected candidates. Parsed observations and dangling references remain inspectable.
- [x] Only current candidates load snapshot bodies. Superseded snapshots remain not_loaded without making the report partial. A selected note can remain graph-available with an invalid snapshot while comparison and the overall report are incomplete.
- [x] Cache-budget exhaustion stops enumeration/reading within the documented lookahead bounds, exposes known omissions, and makes no global lexical-prefix or unique-candidate claim. Quarantine is excluded from discovery.
- [x] Remove intermediate single-note/history restrictions from prior slices. Verify all final nested JSON shapes, diagnostic attribution/deduplication/order, aggregate completeness, and exit statuses using real fixtures for every supplied example state.
- [x] Cooperative concurrent publication fixtures preserve both successors; an incomplete publication remains unknown, an explicit reconciled successor converges the graph, and a stopped-writer quarantine fixture permits complete inspection again without discarding finalized competitors. The reader itself performs no mutation.

## Blocked by

- [Resume from a retained checkpoint](04-retained-checkpoint.md)

## Blocked by decisions

None

## Spec

- [Local orientation and resumption](../../../cli-wayfinding/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Reader contracts](../../../../docs/readers.md)
- [Discovery contracts](../../../../docs/discovery.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)

## Acceptance

- [Local acceptance](../acceptances/05-competing-checkpoints-merge-review-20260919.md)

## Comments

Created from the user-approved implementation plan. Verification output and acceptance history are local files excluded from Git. The current Acceptance link requires those local records; see the [storage guidance](../../../../docs/agents/issue-tracker.md#local-verification-records).
