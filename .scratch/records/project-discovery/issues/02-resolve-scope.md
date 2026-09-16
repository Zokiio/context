---
type: WorkItem
id: 9cb1ab9e-8d7c-4358-948f-1b928ab2472a
title: "Resolve scope from directories and aliases"
triage: ready-for-agent
execution: unstarted
---

# Resolve scope from directories and aliases

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [ ] Implement one testable discovery operation with injected cwd, home, and filesystem dependencies or equivalent test seams, returning resolved scope and provenance without printing or writing.
- [ ] Apply the specified depth, kind, conflict, malformed-candidate, and stopping rules across local configuration, personal bindings, and bundle markers.
- [ ] Use physical ancestry and canonical comparisons; preserve original spellings for diagnostics and enforce no fallback from broken selected scope.
- [ ] Resolve personal aliases independently of cwd and unavailable checkout paths; do not evaluate unrelated record targets.
- [ ] Tests cover nested scopes, explicit kind filtering, same-depth roots conflicts, symlinks, root boundaries, malformed registry/local files, and agreeing marker-plus-mapping roots.

## Blocked by

- [Read discovery configuration](01-read-configuration.md)

## Blocked by decisions

None

## Spec

- [Project discovery specification](../../../project-discovery/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)
