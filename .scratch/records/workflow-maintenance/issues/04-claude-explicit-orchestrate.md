---
type: WorkItem
id: b7f7c310-a6fa-43f9-91ba-1801a8f74431
title: Keep orchestrate out of Claude's automatic invocation
triage: ready-for-agent
execution: in-progress
---

# Keep orchestrate out of Claude's automatic invocation

The user asked on 2026-10-04 for this fix as a separate small PR, after the review of [Zokiio/context#33](https://github.com/Zokiio/context/pull/33) gave the retrospective skill Claude Code's explicit-invocation field and left orchestrate's identical gap outside that ticket's scope.

## Scope

The installed orchestrate skill is explicit-only, but only its Codex `agents/openai.yaml` says so. Claude Code reads `disable-model-invocation` from the skill's frontmatter, so a project that links installed skills into `.claude/skills` lets Claude select orchestrate automatically. Add the field to the canonical template and assert it for every explicit-only installed skill. Other skills and invocation routing are outside scope.

## Acceptance criteria

- C1: The installed orchestrate skill's frontmatter sets `disable-model-invocation: true`, alongside its existing `agents/openai.yaml` policy.
- C2: The portable install test asserts the field for every explicit-only installed skill and fails when orchestrate lacks it.
- C3: The initialization guide describes both explicit-only settings for orchestrate. The canonical template and generated copy agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Install portable explicitly invoked checkpoint orchestration](02-portable-checkpoint-orchestration.md)
- [Domain glossary](../../../../CONTEXT.md)
