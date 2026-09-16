---
type: WorkItem
id: 14cf4188-8cc0-451b-b5d4-758b32a0db52
title: "Document migration and verify discovery end to end"
triage: ready-for-agent
execution: in-progress
---

# Document migration and verify discovery end to end

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [ ] Document config profiles, selection precedence, setup, workspace authoring, source-root rules, diagnostics, and the manifestless --project to --bundle migration.
- [ ] Run a real trial with this repository plus an independent non-Git project and a workspace; demonstrate routine orient without --project, alias selection, and explicit overrides.
- [ ] Exercise missing scope, malformed configuration, broken selected mappings, moved shared checkout, direct recovery through --bundle, and workspace member selection.
- [ ] Retain criterion-level evidence and actual tested revision under the repository acceptance procedure; complete Go test, race, vet, and build checks.
- [ ] Verify readers do not write and setup changes only the intended registration; clearly separate member availability from work readiness in trial outputs.

## Blocked by

- [Use discovery in reader commands](03-integrate-cli.md)
- [Show workspace member navigation](04-workspace-navigation.md)
- [Connect existing records through setup](05-connect-records.md)

## Blocked by decisions

None

## Spec

- [Project discovery specification](../../../project-discovery/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)

## Acceptance

## Comments

2026-09-17: Codex /root claimed the final documentation and trial slice after complete task context and passing prerequisite checks at `933d9c7`. The docs and trial driver reuse `baa1a3e`; verification will run against this stack with a disposable personal registry and fixtures outside the repository.
