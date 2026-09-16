---
type: WorkItem
id: 591c566c-1d96-4824-8cf0-aa775a69a35f
title: "Read discovery configuration"
triage: ready-for-agent
execution: in-progress
---

# Read discovery configuration

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [x] Parse local and personal ContextConfig v1 documents, including project bindings, aliases, workspaces, and members, with field-level diagnostics.
- [ ] Enforce known field types, required identities, duplicate keys/aliases, profile separation, and unsupported-version errors; retain unknown metadata and Markdown body for later setup.
- [ ] Resolve filesystem fields from the declaring file location and canonicalize target identities without requiring unrelated registered targets to be available.
- [x] Tests cover malformed YAML, relative paths, personal/local profiles, duplicate declarations, and separate record locations.

## Blocked by

None

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

- [Configuration acceptance](../acceptances/01-configuration-20260917.md)

## Comments

2026-09-17: Codex started implementation on `feat/discovery-configuration` after complete task context and passing orientation checks. Current commitments now select the discovery tickets.

2026-09-17: All four criteria passed local verification. The Go suite, race suite, vet, and build passed. [Configuration design](../../../project-discovery/configuration-design.md) records the interface choice; [verification evidence](../../../project-discovery/evidence/01-configuration.md) records the tested source manifest and criterion results.

2026-09-17: PR review reopened validation for nested alias-key duplicates and path identity for symlink parent traversal. The shared configuration/path implementation from `feat/project-discovery-implementation` is being adopted and verified before dependent work starts. Earlier evidence remains unchanged.
