---
type: WorkItem
id: 411b505a-db97-4a45-94df-71289bff5a21
title: "Show workspace member navigation"
triage: ready-for-agent
execution: unstarted
---

# Show workspace member navigation

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [ ] Render workspace orient in compact text and the distinct version-1 workspace-navigation JSON contract with work status unevaluated.
- [ ] List authored members deterministically without readiness evaluation, preserving distinct checkout entries with shared project IDs.
- [ ] Provide shallow availability results and safely quoted human selection commands plus structured argument arrays carrying member-specific roots.
- [ ] Return success for complete membership enumeration with unavailable members; fail invalid selected workspaces and project-only operations without choosing a member.
- [ ] Tests cover empty workspaces, missing records, absent checkout with available records, conflicting workspace identities, escaping, and stable output.

## Blocked by

- [Use discovery in reader commands](03-integrate-cli.md)

## Blocked by decisions

None

## Spec

- [Project discovery specification](../../../project-discovery/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)
