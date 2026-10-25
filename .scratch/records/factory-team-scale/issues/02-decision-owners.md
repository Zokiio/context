---
type: WorkItem
id: 4b586d5f-1438-4092-9f65-5053fbc54370
title: Show who each decision waits on
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Factory architecture](../../../../docs/factory-platform-design.md)
- [Factory HTTP and browser access](../../../../docs/factory-http.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Every card that waits on a person names who can decide it. Plan and fix decisions wait on the work item's owner. Approval waits on reviewers. A merge that is ready or failed waits on the approver if they hold merge, otherwise on anyone who holds merge.
- The owner is the current job's coordinator. Without a job, it is the person who requested the current refinement or locked tests, or who gave the go-ahead. Before any of these, the card waits on anyone who holds coordinate.
- A reviewer is an enabled human with review and accept on the project who passes the same independence rule that approval enforces (`independentReviewer`): never the implementer, and not the coordinator unless the project allows coordinator review.
- The board response marks which cards the caller can decide. The Needs-you count and label include only those. Other waiting cards say whom they wait on, for example "Waits for alice" or "Waits for a reviewer". A card that no enabled user can decide says so.
- Card derivation stays independent of the viewer, and the autopilot's behaviour is unchanged. The owner and decider sets come from stored refinements, expectations, plan consents, jobs and grants, with no new stored state.
- Service and browser tests show an owner, a teammate and a read-only user seeing different Needs-you sets on the same board. The coordinator never sees their own approval as theirs, and a grant change shows on the next poll.

## Comments

- 2026-10-25: Approvals go to the pool of reviewers rather than a named teammate. Revisit named review requests only if pilot approvals sit unclaimed.
