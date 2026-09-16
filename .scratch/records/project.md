---
type: Project
id: ca6a73e0-ae93-49a3-b287-12071f7446fd
title: Project and context management
---

# Project and context management

Authoritative work-item records for the local-first project and context management system.

## Goals

Make project understanding durable through local, Git-native records and a reusable Go core, delivered first through a read-only CLI. The [product vision](../../docs/vision.md) defines the direction and staged adoption.

Deliver [session orientation](../session-orientation/spec.md) so a fresh agent can inspect goals, current commitments, ready work, and unresolved decisions before requesting detailed task context. Show eligible work with reasons and retain useful known information with explicit gaps.

Deliver [project discovery](../project-discovery/spec.md) so routine reader commands resolve project or workspace scope from the current directory. Connect existing records through explicit setup.

## Current commitments

- [01: Show project orientation and work inventory](session-orientation/issues/01-show-project-orientation.md)
- [02: List eligible work with reasons](session-orientation/issues/02-list-eligible-work.md)
- [03: Explain blocking decisions](session-orientation/issues/03-explain-blocking-decisions.md)
- [04: Expose reproducible requirement fingerprints](session-orientation/issues/04-expose-requirement-fingerprints.md)
- [05: Satisfy direct prerequisites using recorded acceptance](session-orientation/issues/05-verify-prerequisite-acceptance.md)
- [06: Evaluate prerequisite chains and cycles](session-orientation/issues/06-evaluate-prerequisite-chains.md)
- [07: Migrate project records and prepare the pickup trial](session-orientation/issues/07-migrate-records-and-prepare-trial.md)
- [08: Verify fresh-session pickup on the real project](session-orientation/issues/08-verify-fresh-session-pickup.md)
- [Read discovery configuration](project-discovery/issues/01-read-configuration.md)
- [Resolve scope from directories and aliases](project-discovery/issues/02-resolve-scope.md)
- [Use discovery in reader commands](project-discovery/issues/03-integrate-cli.md)
- [Show workspace member navigation](project-discovery/issues/04-workspace-navigation.md)
- [Connect existing records through setup](project-discovery/issues/05-connect-records.md)
- [Document migration and verify discovery end to end](project-discovery/issues/06-document-and-trial.md)

## Open decisions

None

## Context

- [Product vision](../../docs/vision.md)
- [Domain glossary](../../CONTEXT.md)
- [Tracker conventions](../../docs/agents/issue-tracker.md)
