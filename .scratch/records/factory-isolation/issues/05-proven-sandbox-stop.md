---
type: WorkItem
id: 0c3dd337-36a1-40bd-aed6-b8afafe80777
title: Confirm that a stopped sandboxed run ended
triage: ready-for-agent
execution: unstarted
---

Stopping a sandboxed build or check ends as `cancelled` when the container runtime confirms that the container's processes ended. Without that confirmation it stays `unknown`, as today.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0006-sandboxed-runs-exchange-git-bundles.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Build candidates in a Docker sandbox](04-sandboxed-builds.md)

## Blocked by decisions

None.

## Acceptance criteria

- On a cancellation, deadline or halt, the worker closes the run's egress, sends `SIGTERM` and then `SIGKILL` through the runtime, waits until the runtime reports the container not running, and removes it. It then reports `cancelled` with a `stop-proof.json` artifact naming the container ID, image, runtime-reported state and exit time, the removal result and the egress closure.
- If the runtime is unreachable, or still reports the container running after `SIGKILL`, the worker reports `unknown` with the steps it tried.
- The controller accepts `cancelled` for a non-fixture attempt only when the assignment's profile is sandboxed and the report carries a stop proof. It refuses any other non-fixture `cancelled` report. Unsandboxed native runs keep their current behavior.
- A proven stop releases the claim and produces the job state that an administrator's `stopped_cancelled` resolution produces today. The work item keeps its stop hold.
- Stub-runtime tests cover each runtime state. The `WAYMARK_DOCKER_TESTS=1` integration test starts a process inside the container that calls `setsid` and keeps writing. It shows that the proof is reported only after that process is gone.

## Comments

- 2026-10-25: Every non-fixture cancellation ends `unknown` in `dev` at `afe9989`. Open-Inspect's process-containment tests accept that a killed supervisor is not containment proof. Fabro's Docker stop never re-checks the final state. A container's PID namespace gives the factory a state it can re-check.
