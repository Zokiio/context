---
type: WorkItem
id: 3319dcea-597f-4ab9-bb38-2d99bafede19
title: Settle an interrupted sandboxed run from its container
triage: ready-for-agent
execution: unstarted
---

A worker that restarts can settle its own `unknown` sandboxed attempts by finding each one's container and proving it stopped. An administrator no longer has to resolve them by hand.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0006-sandboxed-runs-exchange-git-bundles.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Confirm that a stopped sandboxed run ended](05-proven-sandbox-stop.md)

## Blocked by decisions

None.

## Acceptance criteria

- On start, the worker finds the container for each of its `unknown` sandboxed receipts by label. When the container is absent or exited, or the worker can stop it with proof, the worker reports a proven stop for that attempt as the same worker identity.
- The service moves the attempt from `unknown` to `cancelled` when cancellation was requested, and to `failed` otherwise, never to `succeeded`. It records the worker, the proof and the prior state as an event. It refuses such a report from another worker identity or without a proof.
- A container that the worker cannot stop, or whose state the runtime cannot report, leaves the attempt `unknown`. Administrator resolution is unchanged.
- The service never assigns an `unknown` attempt to another worker.
- Tests cover restart with a running container, an exited container, a missing container and an unreachable runtime, and a report from the wrong worker identity.

## Comments

- 2026-10-25: Today an interrupted native run stays `unknown` until a project administrator resolves it with evidence. A labelled container is evidence the worker can gather itself.
