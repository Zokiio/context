---
type: WorkItem
id: 0ea2831b-a08a-4e21-8fd7-382eaf43f016
title: Build one real Waymark ticket with Codex
triage: ready-for-human
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#live-validation)

## Context

- [Run a local factory](../../../../docs/factory-platform.md)

## Blocked by

- [Check the native adapters against live providers on demand](03-live-adapter-check.md)

## Blocked by decisions

None.

## Acceptance criteria

- Joakim chooses a small ready Waymark ticket and runs it through `ctx factory solo` with `codex-agent` as the builder, while he watches.
- The job reaches Approve or stops with a recorded failure, and the real Codex CLI did the build and any repairs.
- This ticket's comments record the Codex version, model, outcome of each stage, repairs, usage and every problem seen. Each defect becomes its own ticket.
- The run decides nothing about the chosen ticket. Its change is accepted only through the normal approval.
