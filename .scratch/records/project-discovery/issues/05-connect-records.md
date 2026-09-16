---
type: WorkItem
id: 18a3d758-2b28-48d7-a05c-e4ffc225fb8e
title: "Connect existing records through setup"
triage: ready-for-agent
execution: unstarted
---

# Connect existing records through setup

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [ ] Implement the specified setup flags and guided terminal flow, with deterministic noninteractive operation and no prompts from reader commands.
- [ ] Validate records, binding directory, and authorized roots; resolve cwd inputs before generating shared relative or personal absolute paths.
- [ ] Support dry-run, idempotent setup, explicit replacement, and preservation of unrelated entries, frontmatter metadata, Markdown body, and file permissions.
- [ ] Use atomic writes with concurrent-change detection; failure leaves the previous usable file intact. Never create records or silently repair malformed configuration.
- [ ] Tests cover shared and personal registration, aliases, path rebasing from a nested cwd, conflicts, dry-run, replacement, concurrent edits, and write failures.

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
