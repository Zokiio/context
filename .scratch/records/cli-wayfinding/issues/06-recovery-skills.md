---
type: WorkItem
id: 7ce25cb8-22e4-4012-9ba4-dee5095a5c6f
title: "Teach skills to maintain and recover working notes"
triage: ready-for-agent
execution: completed
---

# Teach skills to maintain and recover working notes

## What to build

Provide usable workflow skills/instructions for writing accurate recovery checkpoints and continuing from the CLI facts, including deliberate recovery from interrupted writes and excessive history.

## Slice boundary

Review and follow the repository guidance for writing agent instructions. This ticket verifies instruction-driven behavior; it does not turn skills into enforced filesystem access controls.

## Acceptance criteria

- [x] The workflow obtains current context and uses the specified ticket/checkout selection. Correct stale direct-project-selector examples to the current direct-bundle contract while preserving user-provided source scope.
- [x] Checkpoint authoring retains the exact context output actually used, its digest, required note fields/sections, meaningful progress, failed approaches, questions, checks and tested identities, and next steps. Do not replace the baseline with unseen newer requirements at publication time.
- [x] Publication uses unique directories and final note publication without overwriting finalized observations. Predecessors identify observations actually incorporated, not merely the newest notes discovered before writing. Reconciliation explicitly incorporates competing accounts after inspecting current code and evidence.
- [x] Continuation refreshes requirements and files, reconstructs absent notes, preserves conflicts, distinguishes reported tests from evidence, and asks only for unresolved decisions needed for sound continuation. Record blocking questions against authoritative work and promote settled decisions/accepted evidence out of the cache.
- [x] Provide the specified unfinished-publication quarantine and whole-task reset procedures with stopped-writer checks, rechecks, no-overwrite destinations, recovery records, retained archives, and explicit loss of automatic continuity. Age alone never establishes abandonment; unknown or live writers prevent retirement.
- [x] Explain history-limit overrides, cache disposal, and exclusion from Git without making Git mandatory. No automatic cleanup, CLI record writes, exclusive claims, or model-understanding guarantees are introduced.
- [x] Exercise instructions against actual filesystem examples and the implemented reader: checkpoint publication, two successors and reconciliation, interrupted publication then confirmed-stop quarantine, refusal for unknown/live writers, history reset, and absent-cache reconstruction. Retain observed results and instruction limitations.

## Blocked by

- [Handle competing and interrupted checkpoints](05-competing-checkpoints.md)

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

- [Local acceptance](../acceptances/06-recovery-skills-merge-review-20260919.md)

## Comments

Created from the user-approved implementation plan. Verification output and acceptance history are local files excluded from Git. The current Acceptance link requires those local records; see the [storage guidance](../../../../docs/agents/issue-tracker.md#local-verification-records).
