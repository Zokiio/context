---
type: WorkItem
id: 0ddb768d-f1eb-470f-abbb-6eea7e791b94
title: "Resume from a retained checkpoint"
triage: ready-for-agent
execution: completed
---

# Resume from a retained checkpoint

## What to build

Let a fresh session retrieve one finalized root checkpoint, validate its retained task context, and compare the actual earlier and current source text through the resume command.

## Slice boundary

Supports a single finalized root observation end to end. Complete history traversal, competing leaves, and reconciliation are delivered by the next slice.

## Acceptance criteria

- [x] Resolve the project/task namespace using stable effective IDs. Validate the RecoveryNote profile, all required headings, UUID and identity agreement, timestamps, digests, duplicate keys/headings, and metadata preservation. Return the note body and typed fields with the specified JSON shapes.
- [x] Apply bounded observation enumeration and fixed-file reading with the specified lookahead limits. Sort only observed names, distinguish complete from partial discovery, enforce physical-path containment, and neither follow arbitrary note links nor enumerate unrelated project/task namespaces.
- [x] For the supported single-root observation, validate exact retained context bytes, schema, source digests, and root task identity. Keep graph validity independent of snapshot status; an invalid, missing, partial, or withheld snapshot produces the specified partial comparison and diagnostics.
- [x] Return source differences with the specified matching, source-order, nullability, authorization, and completeness rules, including uncommitted text, stable-ID root moves, metadata-only text changes, and unknown unmatched sources when either selection is incomplete.
- [x] Historical paths never expand current source authorization. Invalid snapshot text is not emitted; withheld text remains null. A missing baseline never implies unchanged work, and a retained local test claim never certifies current code or full acceptance.
- [x] Malformed or unfinished observations remain explicit failures. Observation stores requiring multi-node history evaluation fail explicitly in this intermediate slice rather than choosing a current note by timestamp, ignoring extra notes, or claiming unique candidacy. The next ticket removes that temporary limitation.
- [x] End-to-end fixtures cover valid root checkpoint, changed/unchanged sources, authorization loss, missing previous/current sources, invalid digests, bounded reads, symlink escape, exact JSON field/null rules, and read-only behavior. Generate real fixture bytes and digests rather than copying illustrative example hashes.

## Blocked by

- [Resume a task without recovery notes](03-resume-without-notes.md)

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

- [Acceptance](../acceptances/04-retained-checkpoint-final-20260919.md)

## Comments

2026-09-19: Reassessed the final reader/discovery reference with passing final checks. Prior decision: [Acceptance](../acceptances/04-retained-checkpoint-review-20260919.md).

Created from the user-approved seven-ticket breakdown. Acceptance requires retained verification and an authored Acceptance record; unstarted execution and triage do not establish implementation eligibility.

2026-09-19: Reassessed timestamp and effective-type corrections after independent review. Prior decision: [Initial acceptance](../acceptances/04-retained-checkpoint-20260919.md).
