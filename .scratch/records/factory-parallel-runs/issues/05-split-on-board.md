---
type: WorkItem
id: ba70efef-22e0-475a-a473-817c41f4e52c
title: Split a work item from the board
triage: ready-for-agent
execution: unstarted
---

A split is part of agreeing what done means, which happens in the board's plan panel. The panel needs to show a proposed split and let the person accept it there.

## Spec

- [Planned steps and parallel builds](../../../factory-parallel-runs/spec.md)

## Context

- [Work item is the factory's unit](../../../../docs/adr/0006-work-item-is-the-factory-unit.md)

## Blocked by

- [Propose and accept a split of a work item](04-split-work-item.md)

## Blocked by decisions

None.

## Acceptance criteria

- The plan panel offers to propose a split for an item in Up next or Plan.
- A proposed split shows each child's title, criteria and blockers, which original criteria each child covers, and any dropped criteria with their reasons. Technical detail stays folded away, as elsewhere on the board.
- The person can edit the children, then accept with or without a go-ahead, or reject. Accepting is a single decision.
- After acceptance, the original leaves the board and the children appear as cards that say which item they were split from. The original's history shows the split decision.
- Unsaved edits to a proposal survive background updates, and a stale save keeps the person's edits, as the task workspace already does.
- Browser checks cover proposing, editing, accepting with a go-ahead, rejecting, a stale save, and narrow layouts in light and dark mode.
