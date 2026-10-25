---
type: WorkItem
id: 5a938431-0b87-4504-a594-f3cd5ef3b94d
title: Merge one change per branch at a time
triage: ready-for-agent
execution: unstarted
---

Two approved changes in one project build their merge commits from the same branch tip. The first moves the branch. The second fails with "the branch moved during verification" and is retried by the autopilot. A candidate that no longer merges cleanly is also treated as retryable. Parallel builds make both more frequent.

## Spec

- [Planned steps and parallel builds](../../../factory-parallel-runs/spec.md)

## Context

- [Work item is the factory's unit](../../../../docs/adr/0006-work-item-is-the-factory-unit.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- While one merge into a branch is verifying, another approved merge into the same branch waits and is shown as waiting on the board. It starts by itself when the one ahead merges or fails, builds its merge commit from the branch tip at that moment, and runs the required checks on it.
- Waiting merges start in approval order, and a restart of the service keeps them waiting in that order.
- Two changes approved together from the same base both merge without a "branch moved" failure, and the second merge commit contains the first.
- A candidate that doesn't merge cleanly into the branch is recorded as a conflict, distinct from a failed check and a moved branch. The branch stays where it was, the board shows the conflict in the Merge column, and the autopilot doesn't retry it.
- A branch moved from outside the factory keeps today's bounded automatic retry.
- Project metrics count merge conflicts.
- Service tests cover two changes touching different files, two touching the same lines, and a restart with a waiting merge, using real Git repositories and fixture workers.
