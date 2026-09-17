---
type: WorkItem
id: 18a3d758-2b28-48d7-a05c-e4ffc225fb8e
title: "Connect existing records through setup"
triage: ready-for-agent
execution: completed
---

# Connect existing records through setup

## What to build

Implement this slice of the approved project-discovery specification. Keep changes within the acceptance criteria below and preserve existing reader contracts except for the explicitly specified selector migration.

## Acceptance criteria

- [x] Implement the specified setup flags and guided terminal flow, with deterministic noninteractive operation and no prompts from reader commands.
- [x] Validate records, binding directory, and authorized roots; resolve cwd inputs before generating shared relative or personal absolute paths.
- [x] Support dry-run, idempotent setup, explicit replacement, and preservation of unrelated entries, frontmatter metadata, Markdown body, and file permissions.
- [x] Use atomic writes with concurrent-change detection; failure leaves the previous usable file intact. Never create records or silently repair malformed configuration.
- [x] Tests cover shared and personal registration, aliases, path rebasing from a nested cwd, conflicts, dry-run, replacement, concurrent edits, and write failures.

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

## Acceptance

- [Reassessed setup acceptance](../acceptances/05-setup-write-reassessed-20260917.md)

## Comments

2026-09-17: Implementation claimed by Codex subagent `/root/configuration_pr_review`, coordinated by `/root`, on `feat/discovery-setup` after complete task context and passing readiness checks at `39483dd`. The setup core reuses the reviewed implementation from `baa1a3e` and corrects preservation of tagged metadata and unrelated YAML alias values before CLI integration.

2026-09-17: All five criteria passed at `5aee51f`. The full Go tests, race checks, vet, and build passed. [Setup verification](../../../project-discovery/evidence/05-setup-verification.md) retains criterion results, the tested revision, and task-context source digests.

2026-09-17: Independent review reopened setup for concurrent shared/personal writes and owner, group, and ACL preservation. Codex `/root/configuration_pr_review` holds the correction claim under `/root` coordination; the permissions helper works under the same claim. The [previous setup acceptance](../acceptances/05-setup-20260917.md) and its evidence remain unchanged. Fresh context was complete and prerequisite checks passed at `933d9c7` before correction.

2026-09-17: The correction passed all five criteria at `c59d12e`, including uncached full tests, race checks, vet, build, and clean independent Standards and Spec rechecks on macOS and Linux. [Final setup write verification](../../../project-discovery/evidence/05-setup-write-final-verification.md) retains the results and platform limits. The subsequent README-only merge at `679e073` left all 110 program-source digests unchanged. The earlier `a0ecc4c` observation remains historical evidence; case-alias ordering and equal-timestamp ACL races are covered by the final correction.
