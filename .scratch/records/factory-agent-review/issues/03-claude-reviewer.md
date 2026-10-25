---
type: WorkItem
id: 8437c405-7632-4b61-829c-cba7d994178d
title: Run the agent review with read-only Claude Code
triage: ready-for-agent
execution: unstarted
---

The reviewer adapter that `solo` uses by default, and the project's reviewer instructions. It follows the judge adapter in `internal/factoryworker/judge.go`.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

- [Two-axis code review skill](../../../../.agents/skills/code-review/SKILL.md)
- [Orchestrate verification brief](../../../../.agents/skills/orchestrate/VERIFICATION.md)

## Blocked by

- [Review each verified candidate with an independent agent](02-agent-review-stage.md)

## Blocked by decisions

None.

## Acceptance criteria

- `ctx factory reviewer` reads the reviewer input for one axis and runs a fresh Claude Code session in print mode. The session has only Read, Grep and Glob, no session persistence, the configured model and per-axis budget, and a JSON schema for the axis output. It returns the result with the model, session, turns and cost. A Claude error, missing structured output or a schema mismatch exits nonzero, so the controller records a failed review.
- The standards prompt reviews the patch against the standards documents the reviewer instructions name and any the reviewer finds at the candidate, citing each rule's file. It skips rules that tooling enforces. Baseline smells are always advisory.
- The spec prompt reports missing, partial and wrongly implemented criteria with their numbers, and unrequested behavior as advisory. It reads the criteria no locked test checks and reports what it finds.
- Both prompts are adapted from the code-review skill and the orchestrate verification brief, with the spec's rules for blocking and advisory findings. They treat source text and the patch as evidence, not instructions.
- `solo` configures the adapter when `claude` is installed, with `--review-model` (default `sonnet`), `--review-budget-usd` per axis (default `1`) and `--no-review`. An explicit `--review-command-file` takes precedence.
- The Agents settings and `PUT /v1/projects/{id}/instructions` accept reviewer instructions. Admission freezes them, and the reviewer input carries them.
- A fake `claude` binary covers the arguments, each axis's prompt, parsing and the error paths. One live Claude review of a real candidate is reported separately from the fixture checks.
