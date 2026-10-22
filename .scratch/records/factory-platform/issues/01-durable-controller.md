---
type: WorkItem
id: 705d97ee-b372-4c49-8378-d4fac0c2c24f
title: Persist factory jobs and enforce their lifecycle
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- Admit an explicitly scoped ready work item with immutable context, requirement fingerprints, base revision, checks, executor, deadline, and repair bounds.
- Persist jobs, claims, attempts, events, observations, and actor attribution across restart with one active implementation claim per work item and idempotent duplicate requests.
- Keep missing or negative required checks, missing review, changed candidate, and stale requirements from reaching acceptance.
- Represent uncertain cancellation and interrupted dispatch explicitly, require reconciliation before repeat, and enforce repair and deadline bounds.
- Focused tests exercise lifecycle, contention, persistence errors and restart using temporary records and repositories.

## Spec

- [Factory platform](../../../factory-platform/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)

## Blocked by

None

## Blocked by decisions

None

## Comments

2026-10-22: Selected by the user as part of the complete factory platform, before target-project pilot work. Parallel implementation uses the same frozen interface contract and separate owned paths. Whole-platform acceptance remains a separate gate.
