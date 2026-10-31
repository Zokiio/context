---
type: WorkItem
id: ca735279-e149-4b93-83ac-54ae082560c1
title: Show every work item's pending decision on the board
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Web workspace rework](../../../factory-webui/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory HTTP and browser access](../../../../docs/factory-http.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- The board response carries, for every work item whether queued or not, the pending refinement or expectation identity and the decision the person is asked to make: review the proposal, answer questions, agree and start, approve, or merge.
- A card whose refinement is proposed is in the needs-you state and names that decision; it does not read as a plain draft.
- Cards carry the record's real `updatedAt`. A card without one sorts by title. A zero time never reaches a client.
- The current browser shows the proposal for an unqueued work item from its card and its decision view, using the new board fields.
- Service tests cover an unqueued work item with a proposed refinement, a queued one, and one with no pending work. `ctx factory projects board` prints the new fields.

## Comments

- 2026-10-30: Recorded from the rework assessment. In the Opsbase grooming trial every item had a proposed refinement and none was visible from the board.
