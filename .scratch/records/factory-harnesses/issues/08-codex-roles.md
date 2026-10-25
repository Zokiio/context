---
type: WorkItem
id: 9f45be60-bf06-4b32-b49b-75e277e14bf2
title: Refine, write tests and judge with Codex
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#role-profiles)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Report agent usage without inventing a cost](05-usage-without-invented-cost.md)
- [Choose the refiner, test author and judge per project](07-role-profiles.md)

## Blocked by decisions

None.

## Acceptance criteria

- `ctx factory refiner`, `ctx factory expectation-author` and `ctx factory judge` accept `--harness codex`. Claude remains the default.
- The Codex refiner and judge run `codex exec` with `--sandbox read-only`, `--output-schema` pointing to a private copy of the role's existing schema, and a fresh session. They return the same `RefinerOutput` and `JudgeOutput` as the Claude versions. The judge keeps its strict decision checks.
- The Codex expectation author runs with `--sandbox workspace-write` in its temporary worktree. The service still keeps its work only when every changed file is protected.
- Each Codex role run records its harness, CLI version, model, session and token usage, and the deadline bounds it because Codex has no dollar cap.
- Solo registers Codex role profiles when `codex` is installed.
- Unit tests use a fake `codex`. The live check gains Codex role scenarios: a refiner proposal from a read-only run, a structured judge decision, and a read-only run that leaves the worktree unchanged.
