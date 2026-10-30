---
type: WorkItem
id: bb248cdd-dfb7-43e2-9721-71ebbeb64ca3
title: Open the workspace on what needs you
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Web workspace rework](../../../factory-webui/spec.md)

## Context

- [Factory platform design](../../../../docs/factory-platform-design.md)

## Blocked by

- [Show every work item's pending decision on the board](01-board-pending-decisions.md)
- [Build the browser workspace from components](02-frontend-toolchain.md)

## Blocked by decisions

None.

## Acceptance criteria

- Home opens with the decisions waiting for the signed-in person, oldest first, one row each with the decision asked, the item title and the waiting time. The row opens the item.
- The remaining work is grouped under Running, Up next, Drafts and Done, each collapsed to a count with its latest items visible. Unqueued drafts appear only under Drafts.
- Cards show title, state and one status line. No cost, revision, score or step bar appears on a card.
- The graph view is reachable from Home and unchanged. The new shell has no list view, task library or drawer.
- At 375 pixels wide the top bar is one row, nothing scrolls sideways, and every control is reachable.
- A browser check against the real service with fixture workers shows a pending plan decision, a running job and a merged job on Home, in light and dark, at desktop and phone width.
