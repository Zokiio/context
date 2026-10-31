---
type: WorkItem
id: 19f77c2c-f9ba-4b9b-be76-daa6383ba4a1
title: Decide what done means on the item page
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Web workspace rework](../../../factory-webui/spec.md)
- [Task workspace scope](../../../factory-task-workspace/spec.md)

## Context

- [Factory platform design](../../../../docs/factory-platform-design.md)

## Blocked by

- [Open the workspace on what needs you](03-home.md)

## Blocked by decisions

None.

## Acceptance criteria

- One route per work item. When a plan decision is pending it is the first section and holds the actions: review and edit a proposed refinement, answer its questions, sharpen again, accept proposed tests, and agree and start with the existing consent for locked tests.
- The decision shows for an unqueued item. Agreeing queues the item as part of the same action and says so.
- The task text, requirements, dependencies and links follow, with the existing editor and its save, stale-save and freeze behaviour.
- The run section lists only the steps that exist for this item, with the live step open and unstarted steps as one line each. Output, artifacts, check results, disputes and intervention controls keep their current behaviour.
- History lists previous jobs, attempts, proposals and decisions as expandable rows.
- A browser check against the real service with fixture workers reviews a proposal, answers a question, agrees and starts, and follows the run to completion on the item page.
