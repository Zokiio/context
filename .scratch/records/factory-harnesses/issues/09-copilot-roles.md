---
type: WorkItem
id: 8a8a34ef-ac14-4654-b6f0-0519911c0b80
title: Refine, write tests and judge with Copilot
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#role-profiles)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Build with GitHub Copilot](06-copilot-builder.md)
- [Choose the refiner, test author and judge per project](07-role-profiles.md)

## Blocked by decisions

None.

## Acceptance criteria

- `ctx factory refiner`, `ctx factory expectation-author` and `ctx factory judge` accept `--harness copilot`.
- The Copilot refiner and judge get read and search tools only, with no write or shell tool. Their final message must be exactly one JSON object that passes the role's existing checks. The judge rejects anything else, so the dispute goes to a person.
- The Copilot expectation author can write in its temporary worktree under the same deny-by-default rules as the Copilot builder.
- Each Copilot role run records its harness, CLI version, model, session and AI-credit usage, and stays within a `--max-ai-credits` cap.
- Solo registers Copilot role profiles when `copilot` is installed.
- Unit tests use a fake `copilot`. The live check gains Copilot role scenarios: a refiner proposal from a read-only run, a structured judge decision, and a read-only run that leaves the worktree unchanged.
