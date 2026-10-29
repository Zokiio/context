---
type: WorkItem
id: d0c0add1-e5dc-46b9-ae8d-fbf2525540a6
title: Write holdout expectations in a sandbox
triage: ready-for-agent
execution: unstarted
---

The expectation author writes and runs its tests inside a container. Its protected patch returns as a bundle that the service checks before keeping it.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0009-sandboxed-runs-exchange-git-bundles.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Confirm that a stopped sandboxed run ended](05-proven-sandbox-stop.md)

## Blocked by decisions

None.

## Acceptance criteria

- When the service has an expectation sandbox configured, it writes a bundle of the base revision and runs the author's Claude Code inside a container with the same properties, egress and secrets as sandboxed builds.
- The author's result returns as a bounded bundle. The service keeps it only if it descends from the base and changes only protected paths, as it requires of today's worktree result.
- A cancellation, halt or interruption follows the stop-proof procedure. An unconfirmed stop keeps the expectation run held for review, as a restart during the test-author handoff does today.
- `solo --sandbox-image` also configures the expectation sandbox. The refiner and judge are unchanged.
- Stub-runtime tests cover the handoff. The `WAYMARK_DOCKER_TESTS=1` integration test shows that the author's tests cannot read the operator token.

## Comments

- 2026-10-25: The expectation author runs inside the service process with `acceptEdits` and `go test`, so it executes agent-written code without a worker boundary. The refiner and judge get only `Read`, `Grep` and `Glob`, and stay out of scope.
