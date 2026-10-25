---
type: WorkItem
id: 9368bc30-f3d5-4c5f-9745-eeefb3381cb1
title: Rebuild a change that no longer merges
triage: ready-for-agent
execution: unstarted
---

When another change lands first, a candidate built from an older base may no longer merge cleanly. Today that leaves the card stuck at Merge. Acceptance is published at approval, before the merge, and sets the ticket's execution to completed, so a conflict after approval also leaves an accepted record for a change that never landed.

## Spec

- [Planned steps and parallel builds](../../../factory-parallel-runs/spec.md)

## Context

- [Work item is the factory's unit](../../../../docs/adr/0006-work-item-is-the-factory-unit.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Merge one change per branch at a time](01-merge-queue.md)

## Blocked by decisions

None.

## Acceptance criteria

- After a candidate's checks pass, the factory checks that it merges cleanly into the current branch tip before the card asks for approval. If it doesn't, the factory starts a linked replacement run from the tip with the same admitted requirements and accepted locked tests, without asking anyone.
- After two automatic rebuilds of one item, the card asks the person instead of rebuilding again.
- Approval repeats the merge check before publishing anything. When the candidate no longer merges cleanly, no review, acceptance or merge is recorded, and the rebuild starts.
- When a queued merge finds a conflict after approval, the card offers a rebuild and the autopilot takes it. The rebuild supersedes the unmerged acceptance: the ticket's Acceptance link and execution state return to their state before that acceptance through an expected-revision write, and the old decision and evidence remain retained. The rebuilt candidate needs approval again, from the person or the ask policy.
- A rebuild never relaxes requirements. If the work item's requirements changed since admission, the item returns to Plan. If the locked tests don't apply cleanly to the tip, the card asks the person.
- History shows each rebuild with the original candidate, the branch tip it conflicted with, and whether it happened before or after approval.
- Service tests cover a conflict found before approval, a conflict found at approval, a conflict after approval with acceptance superseded, the two-rebuild limit, and changed requirements.
