---
type: WorkItem
id: 821f5421-83e8-4104-a693-4e3c7d0522a4
title: Build several ready work items at once
triage: ready-for-agent
execution: unstarted
---

The autopilot starts one build per project at a time, and `ctx factory solo` runs one implementation worker and one verification worker. Independent ready work waits in line.

## Spec

- [Planned steps and parallel builds](../../../factory-parallel-runs/spec.md)

## Context

- [Work item is the factory's unit](../../../../docs/adr/0006-work-item-is-the-factory-unit.md)

## Blocked by

- [Merge one change per branch at a time](01-merge-queue.md)

## Blocked by decisions

None.

## Acceptance criteria

- A project setting limits how many of the project's builds run at once. It defaults to 1, which keeps today's behavior, and a project administrator can change it through the CLI, HTTP and the settings page.
- With the limit above 1, the autopilot starts builds for ready, queued work items in commitment order until the limit is reached. A dependent item still waits until its prerequisite is merged into its base.
- Refinement and test writing remain one run each per project.
- `ctx factory solo --builders N` runs N implementation workers and N verification workers, each with its own identity. Without the flag, solo runs one of each.
- Each running build keeps its own implementation claim, worktree and protected paths. Builds of different items never share a worktree.
- The board shows every running build in the Build column, and project metrics report how many builds ran at the same time.
- Service and solo tests run three independent ready items with a limit of 2. They show two builds running together, the third starting when a slot frees, and all three merging through the merge queue.
