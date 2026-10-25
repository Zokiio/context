---
type: WorkItem
id: ed2015d4-0ec4-46cd-96dc-4dbb2e7d95aa
title: Stop a disconnected remote worker's sandboxes
triage: needs-triage
execution: unstarted
---

A remote worker that loses the service for longer than its configured limit stops its sandboxes with proof. A partitioned host therefore cannot keep running work that the halt cannot reach.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Run sandboxed attempts on a worker on another host](11-remote-workers.md)

## Blocked by decisions

None.

## Acceptance criteria

- The worker accepts `--disconnect-limit`, separate from the service lease. After that long without a successful heartbeat, it stops each running sandbox with proof and keeps its receipt.
- When contact returns, the worker reports the proven stops, and the service settles them as it settles a restarted worker's attempts.
- A service outage shorter than the limit does not stop running builds.
- Tests with an injected clock and an unreachable service cover stopping at the limit, reporting after reconnection, and surviving a short outage.

## Comments

- 2026-10-25: Specified with the remote worker protocol. Keep this ticket in `needs-triage` until remote workers are scheduled.
