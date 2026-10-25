---
type: WorkItem
id: e1e0efd7-7af0-4a44-bb24-bf4b4748378d
title: Run sandboxed attempts on a worker on another host
triage: needs-triage
execution: unstarted
---

A worker with no access to the service's paths can claim a sandboxed build or check, download its input as a bundle, and upload its candidate as a bundle.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0006-sandboxed-runs-exchange-git-bundles.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory HTTP and browser access](../../../../docs/factory-http.md)

## Blocked by

- [Settle an interrupted sandboxed run from its container](06-settle-from-container.md)

## Blocked by decisions

None.

## Acceptance criteria

- A remote assignment carries no host paths. The worker downloads its input bundle from `GET /v1/worker/attempts/{id}/bundle`, authorized only for the assigned worker and dispatch.
- The worker uploads its candidate bundle to `PUT /v1/worker/attempts/{id}/candidate` within a size limit. A repeated upload with the same digest is accepted as the same upload. The terminal report names the candidate revision and the bundle's SHA-256, and the service handles it like a local bundle.
- The worker refuses cleartext outside loopback, as the client already does, and keeps secrets on its own host.
- Receipts, stop proofs and settlement from a container behave as they do on the service host. The service never moves an `unknown` attempt to another worker.
- An integration test runs one service and two workers with separate data directories and no shared paths. Together they complete a sandboxed build and check.

## Comments

- 2026-10-25: The user asked for this protocol to be specified now and built when a pilot needs a second machine. Keep this ticket in `needs-triage` until then.
