---
type: WorkItem
id: d2fcf5c7-71d5-4a9d-b119-063731677579
title: Move settings to the new shell and retire the legacy workspace
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Web workspace rework](../../../factory-webui/spec.md)

## Context

- [Factory HTTP and browser access](../../../../docs/factory-http.md)

## Blocked by

- [Approve and merge from the item page](05-item-page-approve-merge.md)

## Blocked by decisions

None.

## Acceptance criteria

- How much to ask you, Agents, Administration, project access and Insights render on the new shell with their current content and permissions.
- The legacy script, stylesheet, `/legacy/` routes and the fake-server browser regression are removed. Every screen has a browser check against the real service.
- The browser sections of the factory HTTP and platform documents describe the new workspace.
- `go vet`, the Go tests and the browser checks pass.
