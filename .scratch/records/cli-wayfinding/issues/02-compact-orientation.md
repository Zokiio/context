---
type: WorkItem
id: 8dc99b87-b5b6-4cba-82be-dbd1dec85bfc
title: "Show compact project orientation"
triage: ready-for-agent
execution: completed
---

# Show compact project orientation

## What to build

Make default project orientation a compact operator overview of scope, goals, current work, and needed intervention, with expanded detail available separately and full JSON preserved.

## Slice boundary

This can proceed independently of the source-sharing and recovery work. Until the resume command lands, documentation must identify continuation through that command as forthcoming rather than executable functionality.

## Acceptance criteria

- [x] Default text includes authored goal text and source location, evaluation/inventory completeness, current commitments in authored order, needed intervention, in-progress work, the existing new-pickup shortlist, and relevant source/detail pointers.
- [x] Equivalent blocking causes are grouped by check/diagnostic code and resolved source or relationship identity, with all affected commitments and source references retained. Equal prose does not merge distinct causes, and grouping does not invent priority or assignment.
- [x] All warnings/errors and unknown conditions remain visible without silent truncation. Incomplete inventory suppresses the pickup shortlist while preserving independently known facts. In-progress work is not described as new pickup eligibility.
- [x] The detail option selects the full presentation; combining it with JSON or repeating it fails with exit status 2. Full orientation JSON keeps its version-1 fields and evaluation semantics. Workspace detail remains navigation rather than recursive evaluation.
- [x] Orientation performs no cache traversal and makes no claim that a recovery note was found. Continuation guidance identifies the proposed resume operation without changing record state.
- [x] CLI fixtures cover compact/detail/JSON agreement, grouping and order, unknown and partial results, empty projects, argument errors, unchanged workspace navigation, and read-only behavior.

## Blocked by

None

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

## Comments

2026-09-19: Reassessed the final reader/discovery reference with passing final checks. Prior decision: [Acceptance](../acceptances/02-compact-orientation-resume-20260919.md).

Created from the user-approved seven-ticket breakdown. Acceptance requires retained verification and an authored Acceptance record; unstarted execution and triage do not establish implementation eligibility.

Earlier acceptance, retained after the independent review found a decision-row ordering defect:
- [Acceptance](../acceptances/02-compact-orientation-20260919.md)

Earlier acceptance, retained during resumption integration reassessment:
- [Acceptance](../acceptances/02-compact-orientation-order-20260919.md)

## Acceptance

- [Acceptance](../acceptances/02-compact-orientation-final-20260919.md)
