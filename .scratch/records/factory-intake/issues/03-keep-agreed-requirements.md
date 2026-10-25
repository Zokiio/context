---
type: WorkItem
id: 199ffdf9-1144-4eff-8105-903887188b0e
title: Keep agreed requirements when the tracker changes
triage: ready-for-agent
execution: unstarted
---

Ticket 01 refreshes only items with no history, which is safe but leaves refined items frozen at their first snapshot. This ticket implements the ownership switch the user chose: the tracker owns criteria and blockers until a person agrees what done means, and the record owns them afterwards. It also handles tracker changes and closures without disturbing admitted or accepted work.

## Spec

- [Factory intake](../../../factory-intake/spec.md)

## Context

- [Tracker intake decision](../../../../docs/adr/0006-tracker-intake-ownership-and-delegated-pickup.md)
- [Acceptance fingerprints decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)

## Blocked by

- [Import a repository's open GitHub issues as draft work items](01-import-github-issues.md)

## Blocked by decisions

None.

## Acceptance criteria

- Accepting a refinement, giving the plan go-ahead, or starting the build on an imported item writes `requirementsAgreedAt` and a Comments line naming the person. The write happens before admission captures fingerprints.
- After agreement, refresh never rewrites acceptance criteria or blockers. Title and description still follow the tracker while the item is unstarted and nothing holds it.
- A tracker change to the description, criteria or blockers after agreement holds the item at a plan decision reading "Changed in GitHub since you agreed", or "in Jira" for a Jira item. Keeping the agreed version records the decision and releases the hold until the next tracker change. Taking the tracker's version rewrites those fields, clears the agreement and returns the item to planning. The autopilot does not start work on a held item.
- Refresh never rewrites an item that is in progress, completed or cancelled, or that a refinement, test authoring or job holds. It retains the snapshot and shows the change on the Task page. A run in progress keeps its admitted requirements and remains acceptable.
- An issue closed upstream archives its unstarted item with the reason, which removes it from the queue. Active work gets only a "Closed in GitHub" fact and keeps running. A reopened issue that is still in scope restores its item unqueued.
- This replaces ticket 01's rule that refresh rewrites only items with no history.
- Tests cover refresh before agreement, after agreement, during an active job and after acceptance, and confirm that an accepted item's acceptance stays valid after its issue changes. A real-browser check covers both outcomes of the plan decision.
