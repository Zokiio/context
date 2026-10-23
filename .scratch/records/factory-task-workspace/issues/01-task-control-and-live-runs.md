---
type: WorkItem
id: d7621634-8707-425c-b93d-5fd7aac9e48d
title: Manage tasks and observe factory runs from a permanent workspace
triage: ready-for-agent
execution: in-progress
---

## Spec

- [Approved workspace scope](../../../factory-task-workspace/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Task, Run, and History remain reachable under the same task identity before, during, and after execution, with readable Markdown and responsive layouts.
- Draft save never launches workers; queue, start, and remove-from-queue explicitly control eligibility without changing prior record semantics.
- Editor state survives background updates and navigation within a task; discard and stale-save handling preserve the person's writing and prevent accidental overwrite.
- Archive, restore, and restricted draft deletion preserve references and history and refuse active or uncertain execution.
- The run view displays real activity and available output with freshness and missing-data states, preserves inspection state, and supports follow-active-step.
- Authorized intervention distinguishes requested, stopped, and uncertain outcomes; explicit stop prevents automatic restart.
- History exposes prior jobs, attempts, failures, decisions, and original requirement/revision identities.
- Service tests and browser scenarios verify the complete successful loop, failed check and repair, stopping, stale saves, removal, and reconnect behavior.

## Acceptance

## Comments

- 2026-10-23: The user approved implementation of the plan and requested one final combined PR.
