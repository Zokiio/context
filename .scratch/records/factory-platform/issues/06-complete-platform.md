---
type: WorkItem
id: 18d40a22-f648-4ee6-9c38-ab8f22db5d2f
title: Verify and document the complete factory platform
triage: ready-for-agent
execution: completed
---

## Acceptance criteria

- Run the complete CLI, service and browser workflow against real temporary records and Git workspaces with distinct user identities and offline fixture workers.
- Prove duplicate admission, claim contention, access denial, restart, uncertain cancellation, stale requirements, candidate changes, missing checks and bounded repairs prevent false acceptance or duplicate execution.
- Read the resulting attributed Acceptance record through the existing Waymark reader and preserve the requirement and evidence snapshots used.
- Run the existing suite and meaningful new focused and integrated checks. Preserve generated verification and acceptance output outside Git.
- Document installed behavior, actual trust boundaries, startup, users, worker configuration, recovery and remaining limitations. Finish this platform gate before starting Gups pilot work.

## Spec

- [Factory platform](../../../factory-platform/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)

## Blocked by

- [Persist factory jobs and enforce their lifecycle](01-durable-controller.md)
- [Execute scoped assignments in isolated workspaces](02-workspace-workers.md)
- [Expose one authenticated factory service](03-authenticated-service.md)
- [Operate the factory through the ctx CLI](04-factory-cli.md)
- [Operate shared projects and jobs in the web workspace](05-web-workspace.md)

## Blocked by decisions

None

## Comments

2026-10-22: Selected by the user as part of the complete factory platform, before target-project pilot work. Parallel implementation uses the same frozen interface contract and separate owned paths. Whole-platform acceptance remains a separate gate.

## Acceptance

- [Verified complete platform](../acceptances/06-complete-platform-2026-10-23.md)
