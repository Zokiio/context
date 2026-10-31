---
type: WorkItem
id: 11995884-c3d8-449f-b9e8-932502e52e7b
title: Approve and merge from the item page
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Web workspace rework](../../../factory-webui/spec.md)

## Context

- [Deliver accepted work through the code host](../../../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md)

## Blocked by

- [Decide what done means on the item page](04-item-page-plan.md)

## Blocked by decisions

None.

## Acceptance criteria

- A job ready for review or acceptance shows the approve decision first on its item page: the requirements with their test coverage, the change summary, the findings tags and the merge option, with the existing independence and permission rules.
- An accepted unmerged job shows the merge decision, and a merging job shows its steps and their outcomes. Failed, stale and unknown outcomes show the fix section with its controls.
- Disputes are reviewed and routed from the same page.
- The separate job page is removed. A link to a job opens its item with the run section expanded.
- A browser check against the real service with fixture workers approves a candidate, merges it and sees the merged state, and separately sees a failed check with its repair controls.
