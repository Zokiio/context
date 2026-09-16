---
type: WorkItem
id: 591c566c-1d96-4824-8cf0-aa775a69a35f
title: "Read discovery configuration"
triage: ready-for-agent
execution: unstarted
---

# Read discovery configuration

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [ ] Parse local and personal ContextConfig v1 documents, including project bindings, aliases, workspaces, and members, with field-level diagnostics.
- [ ] Enforce known field types, required identities, duplicate keys/aliases, profile separation, and unsupported-version errors; retain unknown metadata and Markdown body for later setup.
- [ ] Resolve filesystem fields from the declaring file location and canonicalize target identities without requiring unrelated registered targets to be available.
- [ ] Tests cover malformed YAML, relative paths, personal/local profiles, duplicate declarations, and separate record locations.

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
