---
type: WorkItem
id: c00aa7be-d641-45a8-b66d-9dca6c576ed5
title: Check the native adapters against live providers on demand
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#live-validation)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- An opt-in Go test runs `ctx factory agent` for each harness named in an environment variable, for example `WAYMARK_LIVE_HARNESSES=claude,codex`, against a disposable Git repository with a small Go task. Without the variable, `go test ./...` skips it, and CI never sets it.
- For each harness the test checks four things. Success produces one candidate commit whose message names the job, attempt, harness, model and session. A run that changes nothing fails without a commit. An instruction to write outside the workspace leaves no file there. A dispute report reaches the dispute output without a commit.
- The test logs each harness's CLI version and whether the run captured a session ID, a model and usage.
- A sanitized JSONL transcript per harness is committed as test data, and parser unit tests read it without a provider.
- Codex adapter defects that block a scenario are fixed in this PR, for example prompt input, model capture, or a Go build cache outside the sandbox. Any other defect becomes its own ticket.
- `docs/factory-platform.md` says which harness versions passed the live check and how to run it. It no longer says that Claude ran once and Codex only through argument construction.
- The PR description summarizes the live results per harness and scenario. Live output stays out of Git.

## Comments

- 2026-10-25: Joakim chose an opt-in live test followed by one real Codex run. The test uses his provider logins and runs only when he sets the variable.
