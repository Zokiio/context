---
type: WorkItem
id: 0ab38e7a-c417-4f5d-a1e6-3d012b785952
title: "Share captured sources between readers"
triage: ready-for-agent
execution: completed
---

# Share captured sources between readers

## What to build

Prepare the existing task-context and orientation operations to consume the same captured source bytes and collection budget within one caller operation, while preserving their standalone command behavior.

## Slice boundary

This is the prerequisite refactor approved in the breakdown. It adds no user-facing command or recovery-note behavior.

## Acceptance criteria

- [x] Existing standalone context and orientation commands retain their selection order, authorization, output schemas, diagnostics, limits, and exit behavior. Workspace navigation and setup remain unaffected.
- [x] A caller can compose Project-manifest capture, task-context collection in its existing traversal order, and remaining orientation collection in its existing order using one source capture and one shared file/byte budget.
- [x] Each physical current source is captured and counted once within the composed operation; all parsing, digests, context text, readiness, and acceptance checks use those captured bytes. Role-specific source authorization still applies even to already captured bytes.
- [x] Shared-limit exhaustion preserves available task context and orientation with separate completeness flags, explicit omissions, and no false complete project evaluation. Cancellation and source failures retain the specified error behavior.
- [x] Focused filesystem/integration checks demonstrate reuse despite a source changing between the two reader phases, correct counting of shared and distinct sources, unchanged standalone results, and no writes. The capture is not presented as an atomic filesystem snapshot.

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

- [Local acceptance](../acceptances/01-share-captured-sources-compact-paths-20260919.md)
