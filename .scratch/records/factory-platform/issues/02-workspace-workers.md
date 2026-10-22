---
type: WorkItem
id: 48957659-d97f-4ade-939b-62ec34abf6c1
title: Execute scoped assignments in isolated workspaces
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- Create a real Git worktree at an identified base without modifying or cleaning the primary checkout.
- Invoke an explicitly configured external worker through argument arrays with a retained scoped assignment and stable attempt identity.
- Retain candidate revisions, command outcomes, logs and execution observations; do not infer acceptance from process exit.
- Heartbeat, cancellation, failed process and uncertain outcome handling preserve evidence and prevent automatic replay.
- Offline fixture workers demonstrate success, failure, cancellation and restart behavior. No live model run is required or claimed.

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
