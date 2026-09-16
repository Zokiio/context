---
type: WorkItem
id: cab6f066-8afe-43cb-be4d-8238740e183a
title: "Use discovery in reader commands"
triage: ready-for-agent
execution: completed
---

# Use discovery in reader commands

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [x] Make scope optional in context and orient, with cwd discovery and consistent --project/--workspace path or alias selection.
- [x] Add mutually exclusive --bundle direct access that bypasses configuration and preserves manifestless context reads and existing orientation validation.
- [x] Preserve ticket-relative semantics, existing project JSON, reader limits, source authorization, and exit statuses; add per-invocation source roots after scope resolution.
- [x] Provide actionable no-scope, ambiguity, and broken-selection errors plus --explain-scope on stderr; reject invalid/repeated selectors.
- [x] CLI tests demonstrate identical selection from cwd and explicit paths, no raw-directory fallback, manifestless migration, and unchanged files after reads.

## Blocked by

- [Resolve scope from directories and aliases](02-resolve-scope.md)

## Blocked by decisions

None

## Spec

- [Project discovery specification](../../../project-discovery/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)

## Comments

Implementation claimed by Codex subagent `/root/integration_design`, coordinated by `/root`, after ticket 02 had valid, fresh acceptance and passing prerequisite checks.

## Acceptance

- [Current acceptance](../acceptances/03-integrate-cli-final.md)

## Comments

Earlier workflow acceptance, retained with its original evidence:

- [Current acceptance](../acceptances/03-integrate-cli-implementation.md)
