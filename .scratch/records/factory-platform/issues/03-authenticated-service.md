---
type: WorkItem
id: 573384a9-c5b5-4628-8daf-5fde1de7ecc5
title: Expose one authenticated factory service
triage: ready-for-agent
execution: completed
---

## Acceptance criteria

- Start one durable local-first service with explicit data and workspace roots and a shared application contract.
- Authenticate distinct users and enforce administrative, coordination, verification, review and read permissions for project-scoped operations.
- Provide project, admission, job, claim, observation, review and cancellation operations with bounded requests and explicit errors.
- Direct API tests prove access denial, project isolation, duplicate request behavior, stale updates and persistence through service restart.
- Keep tokens out of URLs and diagnostic output. Document the actual single-authority and transport boundaries.

## Spec

- [Factory platform](../../../factory-platform/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)

## Blocked by

None

## Blocked by decisions

None

## Comments

2026-10-22: Selected by the user as part of the complete factory platform, before target-project pilot work. Parallel implementation uses the same frozen interface contract and separate owned paths. Whole-platform acceptance remains a separate gate.

## Acceptance

- [Verified platform outcome](../acceptances/03-authenticated-service-2026-10-23.md)
