---
type: WorkItem
id: 27579a5a-d283-4d43-9879-b993e6deee60
title: Start every agent run without the repository's or operator's configuration
triage: ready-for-agent
execution: unstarted
---

Every Claude Code role starts `claude -p` with no isolating flag: the builder (`internal/factoryworker/agent.go`), the expectation author (`internal/factoryworker/expectations.go`), the refiner (`internal/factoryworker/refiner.go`) and the judge (`internal/factoryworker/judge.go`). The builder and expectation author run in worktrees. The refiner and judge run in the project's main checkout. Claude Code never shows its workspace trust dialog in `-p`, so each run loads the target repository's hooks, `env` block, project skills' `allowed-tools` and `.mcp.json` servers, CLAUDE.md, and the operator's own hooks and MCP servers. The Codex builder (`codex exec --sandbox workspace-write`) loads the operator's `config.toml` and `.rules`. An `allow` rule there runs the command outside the sandbox.

A local probe on 29 October 2026 confirmed this with Claude Code 2.1.308 and Codex 0.165.0, using Waymark's own arguments for each role in a scratch repository:

- A repository `PreToolUse` hook that returned `allow` let the builder run a command its allowed tools didn't cover. With hooks off, the same command was denied.
- The main checkout's untracked `.claude/settings.local.json` hook ran inside the build worktree.
- The builder had the repository's `env` value and MCP tool, the CLAUDE.md instruction, and 199 of the operator's MCP tools. The judge-shaped run had no tool-search tool, so all of those definitions went into its prompt. One Haiku turn cost $0.115, and the run failed on its $0.05 budget.
- `--safe-mode` still applied the repository's `env` block. `disableAllHooks` and `--disallowedTools "mcp__*"` still started the repository's MCP server process. `--restricted` with an explicit `--tools` list loaded nothing from the repository or the operator. Login and structured output still worked.
- With an operator rule allowing `git push`, the Codex builder pushed to a remote outside its workspace and outside `/tmp`. With `--ignore-user-config --ignore-rules`, its sandbox refused the same push.

## Spec

- [Agent harnesses: harness contract](../../../factory-harnesses/spec.md#harness-contract)
- [Isolate factory execution: where each process runs](../../../factory-isolation/spec.md#where-each-process-runs)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)
- [Factory platform guide](../../../../docs/factory-platform.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Every Claude Code argument array that Waymark builds includes `--restricted` and an explicit `--tools` list. The builder and expectation author get `Bash Read Edit Write Glob Grep`. The refiner and judge get `Read Grep Glob`. Their permission mode, `--allowedTools`, `--json-schema`, `--model`, `--max-budget-usd` and prompts are unchanged.
- Every `codex exec` argument array that Waymark builds includes `--ignore-user-config` and `--ignore-rules`. No adapter passes `--dangerously-bypass-hook-trust`.
- Waymark reserves `.claude/`, `.mcp.json`, `.codex/` and `.github/hooks/` from every agent role, whatever the project's protected paths say. A builder candidate that adds, edits, renames or deletes a reserved path fails before verification queues. The job records an event that names the files as agent configuration, the attempt keeps the rejected candidate, and a repair restarts from the admitted base. An expectation proposal that touches a reserved path is refused, even when the project protects that path. Admission records the reserved list with the rest of the policy.
- A candidate may still change `CLAUDE.md` or `AGENTS.md`. They are project guidance, not configuration that runs.
- Deterministic tests cover:
  - each role's argument array for each harness;
  - a builder candidate that changes `.claude/settings.json`, which fails before verification;
  - an expectation proposal that adds `.mcp.json`, which is refused;
  - a project that protects `.claude/`, which still can't have the test author write there;
  - a candidate that edits `CLAUDE.md`, which is accepted.
- An opt-in live test reproduces the probe. It runs only when the live adapter variable is set (`WAYMARK_LIVE_HARNESSES`), and CI never sets it. It builds a disposable repository with marker hooks in `.claude/settings.json`, an untracked `.claude/settings.local.json` in the main checkout, an `env` value, a `.mcp.json` server and a CLAUDE.md canary word. It then runs each Claude role through Waymark's arguments on a cheap model with a small spending cap and checks four things:
  - no marker appears;
  - the builder can't see the `env` value;
  - no `mcp__` tool is offered;
  - no reply contains the canary, and a command that a repository hook would allow stays denied.

  For Codex, a `git push` to a bare remote outside the workspace and outside the temporary directory is refused.
- `docs/factory-platform.md` says that agent runs load no hooks, MCP servers, `env`, CLAUDE.md or auto memory from the repository or the operator. Project guidance reaches agents only through Waymark's instructions. It also lists the reserved paths. It advises operators who configure their own refiner, judge or expectation-author command to pass the same flags.
- The harness spec's contract gains a configuration rule matching the first three criteria. The isolation spec's paragraph on the refiner and judge rests their unsandboxed status on `--restricted`, not on having no shell.

## Comments

- 2026-10-29: Joakim asked for this ticket after the isolation probe in `docs/research/factory-evals-hooks-2026-10-29.md` (branch `claude/research-factory-evals-hooks`, commit `6885043`, not yet merged). He chose `--restricted` with explicit tools for Claude Code roles and `--ignore-user-config --ignore-rules` for Codex.
- The reserved paths are deliberately not protected paths. In Waymark, the expectation author may change only protected paths, so protecting `.claude/` would let the test author plant a hook. Work that should change agent configuration is for a person, outside the factory.
- Consequences to expect:
  - CLAUDE.md and auto memory stop reaching agents automatically. A project that keeps build commands only in CLAUDE.md needs them in its Waymark agent instructions or tickets. An agent can still read the file when told to.
  - `--restricted` loads only managed settings and `--settings`. An operator who authenticates through `apiKeyHelper` in user settings would need to pass it through `--settings`; this is untested. Subscription login worked in the probe.
  - `--ignore-user-config` also drops the operator's default Codex model, so a Codex profile without `--model` uses Codex's own default.
  - Managed settings still apply, by design.
- Related tickets. The live adapter check ([03](03-live-adapter-check.md)) can host the live test. Role profiles ([07](07-role-profiles.md)) and Codex roles ([08](08-codex-roles.md)) must keep these flags in their built-in argument arrays. Copilot's equivalent belongs to [06](06-copilot-builder.md) and [09](09-copilot-roles.md), which already strip `COPILOT_ALLOW_ALL`. Those tickets must never pass `--trust-folder`.
