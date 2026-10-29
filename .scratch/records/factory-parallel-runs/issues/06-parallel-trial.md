---
type: WorkItem
id: ce9e2a02-38ee-425d-b11b-d47e693699b6
title: Trial parallel builds and splits on a real backlog
triage: ready-for-human
execution: unstarted
---

Fixture runs establish the mechanics. Whether parallel builds and splits save time without raising conflicts, approvals or test volume needs a real backlog and live builders. The open questions in the spec's Not now table are settled by this trial.

## Spec

- [Planned steps and parallel builds](../../../factory-parallel-runs/spec.md)

## Context

- [Pipeline shape decision](../decisions/01-pipeline-shape.md)

## Blocked by

- [Build several ready work items at once](02-parallel-builds.md)
- [Rebuild a change that no longer merges](03-rebuild-conflicting-change.md)
- [Split a work item from the board](05-split-on-board.md)

## Blocked by decisions

None.

## Acceptance criteria

- The trial runs a real project with live builders: at least four independent ready items with a build limit of 2 or 3, and at least one item that the person splits.
- The same project records a comparison run with a build limit of 1, or a stated reason why a comparison was not possible.
- The trial records, per merged change: lead time from admission to merge, merge conflicts and rebuilds, human and delegated decisions, agent cost, and production and test lines changed.
- For each split, the trial records whether every child landed by itself and whether the person's approval of an early child changed a later child.
- The comments state a verdict on each reopening condition in the spec's Not now table, and on [decision 01](../decisions/01-pipeline-shape.md) if the trial showed a pipeline need.
- Evidence stays in the ignored local evidence location, and a summary is recorded under Comments.
