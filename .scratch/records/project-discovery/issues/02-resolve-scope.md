---
type: WorkItem
id: 9cb1ab9e-8d7c-4358-948f-1b928ab2472a
title: "Resolve scope from directories and aliases"
triage: ready-for-agent
execution: completed
---

# Resolve scope from directories and aliases

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [x] Implement one testable discovery operation with injected cwd, home, and filesystem dependencies or equivalent test seams, returning resolved scope and provenance without printing or writing.
- [x] Apply the specified depth, kind, conflict, malformed-candidate, and stopping rules across local configuration, personal bindings, and bundle markers.
- [x] Use physical ancestry and canonical comparisons; preserve original spellings for diagnostics and enforce no fallback from broken selected scope.
- [x] Resolve personal aliases independently of cwd and unavailable checkout paths; do not evaluate unrelated record targets.
- [x] Tests cover nested scopes, explicit kind filtering, same-depth roots conflicts, symlinks, root boundaries, malformed registry/local files, and agreeing marker-plus-mapping roots.

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

## Acceptance

- [Scope acceptance](../acceptances/02-scope-20260917.md)

## Comments

2026-09-17: Codex started ticket 02 after complete task context and passing readiness checks on `18728ee`. This slice reuses the resolver from `baa1a3e` with the shared configuration core and adds focused boundary tests.

2026-09-17: All five criteria passed at `fbfcacd`. Repository tests, race checks, vet, and build passed. [Scope verification](../../../project-discovery/evidence/02-scope-verification.md) retains criterion results and source identity.
