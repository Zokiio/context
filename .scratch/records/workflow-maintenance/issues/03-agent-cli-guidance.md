---
type: WorkItem
id: 7e2d53b2-2c31-461e-a18c-ae628f8fcf3e
title: Give agents a discoverable ctx CLI entry point
triage: ready-for-agent
execution: completed
---

# Give agents a discoverable ctx CLI entry point

The user selected CLI guidance on 2026-10-02 after asking how agents know and are instructed to use the CLI. MCP is deferred.

## Scope

Give this repository's agents an instruction pointer to a concise CLI guide. Distribute the same guide through ctx init in the configured docs directory and include its pointer in init's reviewable instruction block. Explain when to run orient, context, and resume, how to identify the executable and explicit project scope, and how to interpret partial results. Keep specialized task-context and recovery procedures linked, and preserve explicit invocation of orchestrate.

The development guide must work in a fresh repository checkout without a saved project binding. Installed examples use the supplied binary and target binding from another working directory. Preserve existing init file handling and existing agent instruction files. Record closure, a needs-attention view, MCP, routing, supervision, and acceptance renewal are separate outcomes.

## Acceptance criteria

- G1: AGENTS.md directs agents to the CLI guide when selecting work, reporting project status, implementing a known ticket, or continuing work. The guide identifies the appropriate reader and links its specialized procedures.
- G2: ctx init installs the guide under the selected docs directory and prints its reachable instruction pointer. The existing preservation behavior applies; init does not edit AGENTS.md or CLAUDE.md.
- G3: Installed orient, context, and resume examples use the supplied binary and explicit target binding, including custom paths with spaces and quotes. Executed examples from an unrelated caller select the target project and actual task source text. Development examples work without a saved binding.
- G4: Canonical and generated guidance agree, installation is documented, and instructions distinguish reader completeness, readiness, human authorization, and acceptance. Orchestration remains explicitly requested; no MCP server or new automatic workflow is added.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Tracker conventions](../../../../docs/agents/issue-tracker.md)

## Comments

2026-10-02: One independently testable installation and discovery outcome, initially pending human review and merge.

2026-10-02: Completed after human review and merged publication in [PR #27](https://github.com/Zokiio/context/pull/27). The closeout acceptance covers G1 through G4, using retained verification at its original tested revision. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

## Acceptance

[Closeout acceptance](../acceptances/03-cli-closeout-20261002.md)
