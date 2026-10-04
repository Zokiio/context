---
type: WorkItem
id: b08c271a-47a9-4d20-9fa5-168c140e2875
title: Dispatch checkpoint work by pointer
triage: ready-for-agent
execution: completed
---

# Dispatch checkpoint work by pointer

Proposed on 2026-10-04 in the workflow adaptation specification. Axpilot #264 wrote 554 KB of dispatch packets for 20 KB of checkpoint files. [PR #30](https://github.com/Zokiio/context/pull/30) merged after human review on 2026-10-04.

## Scope

Change the orchestrate templates so the checkpoint file is the implementing agent's packet and dispatches carry pointers with digests. Keep the exclusions, one active writer, fresh contexts, and retained dispatch records. Parallel writers and changes to gates are outside scope.

## Acceptance criteria

- D1: The orchestrate skill requires each checkpoint file to hold the outcome, requirements, approved seams, gates, permitted paths, and dependency contract that an implementing agent needs.
- D2: Implementation, repair, runner, and review dispatches name the role, the checkpoint path and digest, the starting or evaluated revision, shared guidance by path and digest, and the findings file for a repair. They do not repeat checkpoint text.
- D3: The rule that a pointer list alone is insufficient is removed. The exclusion of the whole plan, sibling checkpoints, full task-context JSON, and other workers' conversations remains, and the orchestrator retains each dispatch with its digests.
- D4: The canonical templates and generated development copies agree, installed links resolve, and `go test ./internal/initialization ./internal/cli` passes.

## Spec

- [Workflow adaptation](../../../workflow-adaptation/spec.md)

## Blocked by

None

## Blocked by decisions

None

## Context

- [Orchestration design](../../../workflow-improvements/orchestration-design.md)
- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-10-04: Completed after human review and merged publication in [PR #30](https://github.com/Zokiio/context/pull/30), covering D1 through D4. Independent Standards and Specification reviews, their follow-up renewals, and CI passed before merge. The closeout acceptance uses retained verification at the merged head. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

## Acceptance

[Closeout acceptance](../acceptances/01-dispatch-by-pointer-closeout-20261004.md)
