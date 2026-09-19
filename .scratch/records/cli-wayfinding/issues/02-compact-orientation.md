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

- [x] Default text includes authored goal text and the declared record-store root, evaluation/inventory completeness, current commitments in authored order, needed intervention, in-progress work, the existing new-pickup shortlist, and relevant source/detail pointers.
- [x] Equivalent blocking causes are grouped by check/diagnostic code and resolved source or relationship identity, with all affected commitments and actionable source locations retained. Compact paths are relative to the declared record-store root where possible; full relationship provenance remains in detail and JSON. Equal prose does not merge distinct causes, and grouping does not invent priority or assignment.
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

Created from the user-approved implementation plan. Verification output and acceptance history are local files excluded from Git. The current Acceptance link requires those local records; see the [storage guidance](../../../../docs/agents/issue-tracker.md#local-verification-records).

## Acceptance

- [Local acceptance](../acceptances/02-compact-orientation-compact-paths-20260919.md)
