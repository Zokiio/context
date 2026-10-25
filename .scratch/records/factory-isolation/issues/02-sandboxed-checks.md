---
type: WorkItem
id: 50425021-3c7e-4dbd-9898-c2f72dd947cc
title: Run required checks in a Docker sandbox
triage: ready-for-agent
execution: unstarted
---

A required check can run the candidate's code in a container that cannot see host files, tokens, environment or network. Code reaches the container only as a Git bundle.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0006-sandboxed-runs-exchange-git-bundles.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Check profiles accept an optional `sandbox` with backend `docker`, an image pinned by image ID or registry digest, and required `cpus`, `memoryMB` and `pids` limits. Registration refuses a mutable tag, missing limits or an unknown backend. `checks register` and the HTTP API expose the field.
- For a sandboxed check, the service writes a bundle of the candidate. The verification worker mounts only that bundle, read-only, clones it inside the container, confirms that HEAD equals the candidate and is clean, and runs the check argv there.
- The container runs as a non-root user with no added capabilities, `no-new-privileges`, `--network none`, the declared limits, no host path except the bundle, and only the `CTX_FACTORY_*` variables. Its name derives from the dispatch ID, and its labels name the installation, worker, attempt and dispatch. The worker removes it after the run.
- The check result's `environment` records the image, and the attempt keeps the bounded log as today. Unsandboxed checks behave exactly as before. A cancelled or timed-out sandboxed check still ends `unknown`; [proven stops](05-proven-sandbox-stop.md) change that.
- Tests with a stub `docker` executable cover argv, environment, labels, cleanup and runtime failures. An integration test, run only when `WAYMARK_DOCKER_TESTS=1`, shows that a sandboxed check cannot read a token file in the worker's data directory, sees no inherited variable and cannot open a network connection.
- The repository ships a reference Dockerfile with Git and the Go toolchain. The operator guide shows how to build it and register a sandboxed check.

## Comments

- 2026-10-25: Verification runs agent-written code as the operator's user with the worker's full environment (`factoryworker/process.go`), so in solo mode a check can read the operator token. This ticket closes that path first because checks need no agent, egress or secrets.
