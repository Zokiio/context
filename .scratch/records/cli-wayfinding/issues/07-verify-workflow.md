---
type: WorkItem
id: a9d219cd-935b-4f9a-83db-9473e2d4ea6a
title: "Verify the complete interrupted-task workflow"
triage: ready-for-agent
execution: completed
---

# Verify the complete interrupted-task workflow

## What to build

Complete the user-facing reference and prove the assembled CLI/skills workflow through the agreed interrupted-real-task acceptance scenario and independent project arrangements.

## Slice boundary

This is combined behavior verification and documentation. Each earlier ticket still owns its own focused tests and acceptance; this ticket is not a substitute for them.

## Acceptance criteria

- [x] User documentation explains compact/detail/operator and JSON/agent views, resume selection and source authorization, checkpoint provenance, complete versus partial reports, exits, conflicting notes, and deliberate recovery/reset procedures using working command examples.
- [x] Run a real task until meaningful unfinished work and verification gaps exist, interrupt the session, and resume in a fresh session without inherited chat, given only the repository path, ticket reference, and instruction to continue. Retain evidence that the successor finds context/notes, explains completed/remaining/changed work, states the next action, and continues within agreed scope without the user reconstructing the prior session.
- [x] Exercise an uncommitted requirement change, a recorded blocking decision, unsupported test claims, retained historical local results, and missing device/runtime evidence. Missing verification stays visible; backend work does not silently complete a parent outcome.
- [x] Run the workflow in separate checkouts, with a separate record store, and in an independent non-Git project. Demonstrate missing notes, conflicting notes, unavailable sources, interruption recovery, and bounded/incomplete inventory behavior without mixing scopes.
- [x] Confirm cache deletion preserves authoritative requirements, blockers, decisions, and accepted evidence. Snapshot files and Git state around read operations to verify that context, orient, and resume do not write.
- [x] Run the repository Go test, race, vet, and build checks against the final source identity. Retain criterion-level evidence, actual tested revisions or working-tree source identity, remaining limitations, and acceptance records using the repository acceptance procedure.
- [x] Reassess any prerequisite acceptance affected by shared source, contract, or instruction changes before claiming the final dependency graph satisfied. Do not represent illustrative prototypes or JSON examples as executed acceptance evidence.

## Blocked by

- [Show compact project orientation](02-compact-orientation.md)
- [Teach skills to maintain and recover working notes](06-recovery-skills.md)

## Blocked by decisions

None

## Spec

- [Local orientation and resumption](../../../cli-wayfinding/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Reader contracts](../../../../docs/readers.md)
- [Discovery contracts](../../../../docs/discovery.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)

## Comments

Created from the user-approved seven-ticket breakdown. Acceptance requires retained verification and an authored Acceptance record; unstarted execution and triage do not establish implementation eligibility.

2026-09-19: For the deliberate fresh-session trial, the documentation session owns user documentation. The root coordinator owns controlled verification fixtures, final repository checks, and acceptance records. The initial documentation session stops after useful unfinished work and a checkpoint. Its successor continues from this ticket and repository guidance.

## Acceptance

- [Acceptance](../acceptances/07-verify-workflow-final-20260919.md)
