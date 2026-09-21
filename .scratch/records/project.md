---
type: Project
id: ca6a73e0-ae93-49a3-b287-12071f7446fd
title: Project and context management
---

# Project and context management

Authoritative work-item records for the local-first project and context management system.

## Goals

Implement the preparation repeated in the Opsbase and Mukabi trials through the [authorized init scope](ctx-init/decisions/01-initialization-scope.md). Keep recovery work limited to demonstrated needs.

Make project understanding durable through local, Git-native records and a reusable Go core, delivered first through a read-only CLI. The [product vision](../../docs/vision.md) defines the direction and staged adoption.

Deliver [session orientation](../session-orientation/spec.md) so a fresh agent can inspect goals, current commitments, ready work, and unresolved decisions before requesting detailed task context. Show eligible work with reasons and retain useful known information with explicit gaps.

## Current commitments

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
