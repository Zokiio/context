---
type: WorkItem
id: 6830cd10-667f-4a48-915a-fb8b9ffcb4b8
title: Build candidates in a Docker sandbox
triage: ready-for-agent
execution: unstarted
---

The builder agent and every command it runs work inside a container. The candidate comes back as a bundle that the service verifies and materializes itself, so verification, review and merge keep their current inputs.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0009-sandboxed-runs-exchange-git-bundles.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Limit sandbox egress to listed hosts and named secrets](03-sandbox-egress-secrets.md)

## Blocked by decisions

None.

## Acceptance criteria

- An executor profile accepts a `sandbox` only when its command is the built-in `ctx factory agent` adapter. Registration refuses a sandbox on any other executor command.
- For a sandboxed build, the service writes a bundle of the attempt's input revision. The worker creates the container and clones the bundle inside it, then runs the adapter on the host with the container's handle. The native agent and the commands it spawns run inside the container. The adapter commits there and reads any dispute report out as bounded bytes.
- The worker streams `git bundle create - <input>..HEAD` out of the container, stops at a size limit, and stores the bundle in the attempt's service-owned bundle directory. The service checks that the bundle contains exactly the reported candidate and that the candidate descends from the input. It then fetches the objects and materializes the attempt's worktree at the candidate. Existing candidate validation, protected paths, the change view, verification and merge run unchanged. No host Git command runs inside a directory the sandbox wrote.
- The worker's token, receipt directory and data directory never enter the container. Claude Code authenticates with a named secret. The reference builder profile allows only `api.anthropic.com` and sets `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`.
- The reference Dockerfile adds Node and Claude Code. `solo --sandbox-image <id>` registers a sandboxed Claude builder under a new profile key. The operator guide shows the sandboxed solo setup.
- Stub-runtime tests drive the adapter with a fixture agent. In the `WAYMARK_DOCKER_TESTS=1` integration test, a fixture agent inside the sandbox fails to write the project's `.git/hooks`, read the operator token, or reach an unlisted host. Its candidate still reaches verification.

## Comments

- 2026-10-25: The builder's allowed `go test` runs agent-written code with the worker's permissions, so the same token path that ticket 02 closes for checks also exists here.
