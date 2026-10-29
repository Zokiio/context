---
type: WorkItem
id: d618c987-ccd5-4bc6-bd0f-c2b62da9655b
title: Build with GitHub Copilot
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#why-the-copilot-cli-and-not-acp-ahp-or-the-sdk)

## Context

- [Copilot orchestration reference](../../../../.agents/skills/orchestrate/COPILOT.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Check the native adapters against live providers on demand](03-live-adapter-check.md)
- [Report agent usage without inventing a cost](05-usage-without-invented-cost.md)

## Blocked by decisions

None.

## Acceptance criteria

- `ctx factory agent --harness copilot` runs the Copilot CLI in programmatic mode in the assigned workspace with the prompt on stdin, JSONL events and a fresh `--session-id` that Waymark generates.
- Permissions are deny by default. The profile allows `write` and its listed shell patterns, denies `git commit` and `git push`, and enables `--sandbox` where the host supports it. It also passes `--disable-builtin-mcps`, `--no-ask-user` and `--no-auto-update`, a `--max-ai-credits` cap from the profile, and a private `--usage-output-file`.
- The adapter never passes `--allow-all`, `--allow-all-tools`, `--allow-all-paths` or `--yolo`. It removes `COPILOT_ALLOW_ALL` from the agent's environment.
- The adapter records the session ID, model, CLI version, progress notes, final message, errors and AI-credit usage. It commits the candidate under the same rules as Claude and Codex.
- Solo registers a `copilot-agent` executor profile when `copilot` is installed.
- Unit tests use a fake `copilot` and a committed sanitized transcript. The live check gains a `copilot` harness that passes the same four scenarios.
- `docs/factory-platform.md` documents the Copilot profile and its permissions.

## Comments

- 2026-10-25: Joakim made Copilot a required harness and chose the programmatic CLI over ACP, the Agent Host Protocol and the Go SDK.
