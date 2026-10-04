---
type: Project
id: ca6a73e0-ae93-49a3-b287-12071f7446fd
title: Waymark
---

# Waymark

Authoritative work-item records for Waymark, the local-first project and context management system.

## Goals

Implement the preparation repeated in the Opsbase and Mukabi trials through the [authorized init scope](ctx-init/decisions/01-initialization-scope.md). Keep recovery work limited to demonstrated needs.

Make project understanding durable through local, Git-native records and a reusable Go core, delivered first through a read-only CLI. The [product vision](../../docs/vision.md) defines the direction and staged adoption.

Deliver [session orientation](../session-orientation/spec.md) so a fresh agent can inspect goals, current commitments, ready work, and unresolved decisions before requesting detailed task context. Show eligible work with reasons and retain useful known information with explicit gaps.

Use the [trial conclusions](../workflow-improvements/orchestration-conclusions.md) as the basis for the workflow-maintenance commitments. They record delivered behavior and verification limits separately from historical acceptance.

## Current commitments

- [Dispatch checkpoint work by pointer](workflow-adaptation/issues/01-dispatch-by-pointer.md)
- [Return short check and review summaries](workflow-adaptation/issues/02-short-summaries.md)
- [Review a work item's sessions through a retrospective](workflow-adaptation/issues/03-retrospective.md)
- [Write checkpoint tests in the implementer's loop](workflow-adaptation/issues/04-tests-in-implementer-loop.md)
- [Keep orchestrate out of Claude's automatic invocation](workflow-maintenance/issues/04-claude-explicit-orchestrate.md)

## Commitment notes

2026-10-04: The user selected a separate fix for orchestrate's missing Claude Code explicit-invocation field, found while reviewing the retrospective skill.

2026-10-04: The user selected the four [workflow adaptation](../workflow-adaptation/spec.md) WorkItems and resolved both of its decisions: tests move into the implementer's loop, and `ctx init` installs the retrospective skill.

2026-10-02: The two maintenance commitments completed after human review and merged through PRs #25 and #26. The additional CLI-guidance outcome merged through PR #27. No new implementation commitment is selected here.

## Delivered work and acceptance

2026-10-02: The maintenance work below is delivered. Each WorkItem links its current local Acceptance decision, based on retained verification at the original tested revision and reviewed publication. Restore those decisions with their evidence when evaluating acceptance in another checkout.

- [Publish current project and orchestration trial conclusions](workflow-maintenance/issues/01-project-status-conclusions.md), [PR #25](https://github.com/Zokiio/context/pull/25).
- [Install portable explicitly invoked checkpoint orchestration](workflow-maintenance/issues/02-portable-checkpoint-orchestration.md), [PR #26](https://github.com/Zokiio/context/pull/26).
- [Give agents a discoverable ctx CLI entry point](workflow-maintenance/issues/03-agent-cli-guidance.md), [PR #27](https://github.com/Zokiio/context/pull/27).

2026-10-02: The previous commitments below record completed execution. The [trial conclusions](../workflow-improvements/orchestration-conclusions.md) identify the fixture and Axpilot #241 outcomes, tested revisions, merged publication, and verification limits.

Completed execution and merged publication do not establish current acceptance. In the original checkout, changed linked sources made historical acceptance stale, and the local `ctx-init` Acceptance record was unavailable. Fresh checkouts have additional unknowns when ignored decisions or evidence are absent. Acceptance reassessment is separate from implementation delivery and is outside these commitments.

- [Exercise explicit checkpoint orchestration on an isolated fixture](orchestration-trial/issues/01-explicit-orchestrator-trial.md)

- [Prepare a project through ctx init](ctx-init/issues/01-agent-guided-project-initialization.md)

- [Share captured sources between readers](cli-wayfinding/issues/01-share-captured-sources.md)
- [Show compact project orientation](cli-wayfinding/issues/02-compact-orientation.md)
- [Resume a task without recovery notes](cli-wayfinding/issues/03-resume-without-notes.md)
- [Resume from a retained checkpoint](cli-wayfinding/issues/04-retained-checkpoint.md)
- [Handle competing and interrupted checkpoints](cli-wayfinding/issues/05-competing-checkpoints.md)
- [Teach skills to maintain and recover working notes](cli-wayfinding/issues/06-recovery-skills.md)
- [Verify the complete interrupted-task workflow](cli-wayfinding/issues/07-verify-workflow.md)
- [Keep compact orientation focused on work](cli-wayfinding/issues/08-compact-paths.md)

## Open decisions

None

## Context

- [Product vision](../../docs/vision.md)
- [Domain glossary](../../CONTEXT.md)
- [Tracker conventions](../../docs/agents/issue-tracker.md)
