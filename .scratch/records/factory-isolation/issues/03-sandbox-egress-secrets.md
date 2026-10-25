---
type: WorkItem
id: 06bdd09f-9fcf-4cb9-a787-922f9c3e33a4
title: Limit sandbox egress to listed hosts and named secrets
triage: ready-for-agent
execution: unstarted
---

A sandbox can reach only the hosts its profile lists, through a Waymark proxy that logs every connection and closes on cancel or halt. It receives only the secrets its profile names, from a file on the worker host.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0009-sandboxed-runs-exchange-git-bundles.md)

## Blocked by

- [Halt all factory execution from one switch](01-installation-halt.md)
- [Run required checks in a Docker sandbox](02-sandboxed-checks.md)

## Blocked by decisions

None.

## Acceptance criteria

- Sandbox profiles accept `egress`, a list of hostnames reachable on port 443, and `secrets`, a list of variable names. Both default to none, and an attempt with no egress keeps `--network none`.
- An attempt with egress runs on its own `--internal` Docker network, with a proxy sidecar built from this repository's Dockerfile. The sandbox reaches listed hosts only through the proxy. Unlisted hosts and direct connections fail. An `egress.log` artifact records each connection's time, host, port, decision and byte counts.
- Cancelling the attempt or engaging the halt makes the proxy refuse new connections and close open tunnels. The worker records whether it confirmed the closure.
- The worker reads `--secrets-file`, a regular private bounded file holding a JSON object of names and values. It passes values to `docker run` only through the CLI's environment. No value appears in argv, assignments, prompts, logs, artifacts or the service database. A missing named secret ends the attempt before any container starts, and the service records that reason.
- The worker removes the network and the proxy after the attempt.
- Stub-runtime tests show that no secret value appears in any argv or record. The `WAYMARK_DOCKER_TESTS=1` integration test reaches a listed fixture host through the proxy, is refused for an unlisted host, fails a direct connection, finds a secret only in its named container, and records whether the runtime answers external DNS queries on internal networks.

## Comments

- 2026-10-25: The user chose deny-by-default egress with a proxy allowlist and worker-held secrets. Fabro's Docker sandbox supports only all-or-nothing networking at the researched revision, so this proxy is Waymark's own.
