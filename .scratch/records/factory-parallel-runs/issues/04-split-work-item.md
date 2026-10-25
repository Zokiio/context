---
type: WorkItem
id: 850c218a-b140-4533-8547-fbc31b4927d2
title: Propose and accept a split of a work item
triage: ready-for-agent
execution: unstarted
---

Grooming is meant to propose ticket splits around independently verifiable outcomes, and the person decides. The refiner can rewrite criteria and ask up to three questions, but it can't propose a split, and the factory can't carry one out.

## Spec

- [Planned steps and parallel builds](../../../factory-parallel-runs/spec.md)

## Context

- [Work item is the factory's unit](../../../../docs/adr/0008-work-item-is-the-factory-unit.md)
- [Product vision](../../../../docs/vision.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- The refiner's output may include a split of two to five children. Each child has a title, a description, at least one criterion, and blockers among earlier siblings only. The refiner's instructions say to split only an item that holds several independently verifiable outcomes and to keep tightly coupled changes together. A person can request a split proposal for an item.
- Every original criterion maps to at least one child or is listed as dropped with a reason. The service refuses a proposal with an unmapped criterion.
- Only a person can accept a split, whatever the project's ask policy. The person may edit child titles, descriptions, criteria and blockers before accepting. Dropped criteria are listed in the decision.
- Acceptance writes one retained publication plan, and an interrupted plan is reconciled on restart without overwriting divergent content. The plan creates the children as queued work items that link the original under Context, changes each item blocked by the original to be blocked by all of the children, and archives the original with a comment naming the children and the criterion mapping. The original's ID and history remain.
- Accepting with a go-ahead gives each child the go-ahead. For each child it stops at a new question, a flagged criterion, or a test author's concern.
- A split is refused while the original has an active or uncertain refinement, test-writing run, job or verification. Rejecting a split leaves the original unchanged.
- CLI and HTTP expose requesting, showing, accepting with edits, and rejecting a split, with request IDs and expected revisions.
- Service tests with a fixture refiner cover a split with a dependent item, a dropped criterion, an unmapped criterion refused, an edit before acceptance, an interrupted publication, and refusal during an active job.
