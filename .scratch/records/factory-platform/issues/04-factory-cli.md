---
type: WorkItem
id: de7091f5-6926-4e6e-8fb1-1a227fb5e57d
title: Operate the factory through the ctx CLI
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- Provide discoverable commands for service startup, user and project setup, job submission and inspection, workers, and authorized lifecycle operations.
- Use the same service state and policy as the browser rather than independent client-side decisions.
- Preserve the existing reader subcommands and their JSON and exit-status contracts.
- CLI integration tests exercise actual HTTP requests and retain nonzero outcomes for denied, invalid and incomplete operations.
- Provide runnable offline setup and fixture-worker examples with honest limits.

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
