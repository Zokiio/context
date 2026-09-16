---
type: WorkItem
id: baf829dd-5016-47a9-aa76-368e50c8f327
title: "Evaluate prerequisite chains and cycles"
triage: ready-for-agent
execution: completed
---

# 06: Evaluate prerequisite chains and cycles

## What to build

Orientation evaluates the complete prerequisite graph, explains indirect obstacles, and handles cycles without preventing unrelated work from being considered. This completes the milestone's readiness behavior.

## Scope

Extend accepted direct prerequisites through their own prerequisites. Integrate recursive evaluation with the existing source budget, partial results, ordering, decisions, and acceptance checks. Preserve the distinction between work readiness and task-context completeness.

## Acceptance criteria

- [x] A prerequisite edge passes only when the target is completed, its current acceptance is valid and fresh, its blocking decisions are resolved, and all of its own prerequisite edges pass. A completed intermediate record cannot hide an unmet deeper prerequisite.
- [x] Evaluate shared prerequisites consistently from captured sources. Preserve the direct and transitive edges needed to explain each result without rereading sources or inflating source limits for repeated references.
- [x] Unfinished or cancelled prerequisites remain known blockers. Missing sources, ambiguous identities, wrong-type targets, malformed declarations, and unavailable acceptance remain unknown. Preserve each unknown condition even when another known failure determines blocked readiness.
- [x] Detect self-dependencies and longer cycles without unbounded recursion. Report the participating identities and edges. Mark cycle members and work depending on those cyclic edges blocked while continuing to evaluate unrelated records.
- [x] A fully observed cycle or known stale acceptance does not by itself make the report partial. Complete evaluations can return exit 0 with every ticket blocked. The existing task-context operation can still return complete source context for the same cycle.
- [x] Honor collection limits across graph, decision, requirement, and evidence reads. After the first breach, retain established facts, identify known omissions, and leave affected checks unknown. Incomplete inventory suppresses the shortlist even when some individual checks already pass.
- [x] A partial report caused by unrelated unavailable project information can still show independently established eligible work when scope, identity, inventory, membership, and that work's checks are known. Do not silently turn missing data into a known blocker or a passing condition.
- [x] Remove temporary unsupported results for all behavior required by the milestone. Text and JSON show the complete four-check model, acceptance details, reasons, and deterministic work groups without ranking a preferred task.
- [x] Public-operation tests cover direct and transitive chains, shared prerequisites, cancelled work, open decisions in prerequisites, stale and unavailable acceptance, self-cycles, longer cycles, dependents of cycles, and unrelated ready work. Include combined known failures and unknown information.
- [x] Verify graph behavior through partial and exact-fit collections, deduplicated external sources, and an independent project. Small CLI checks distinguish complete blocked reports, partial reports, and operation failures. Existing reader regression checks still pass.
- [x] Record this slice's criteria evidence, actual tested revision, and current acceptance using the established completion procedure. Preserve earlier acceptance records and reassess any requirements changed during implementation.

## Blocked by

- [05: Satisfy direct prerequisites using recorded acceptance](05-verify-prerequisite-acceptance.md)

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Domain glossary](../../../../CONTEXT.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
- [Acceptance authoring and fingerprint reference](../../../../docs/agents/acceptance.md)
- [Existing reader contract](../../../context-reader/spec.md)

## Comments

- Verified all 11 criteria. [Immutable verification evidence](../../../session-orientation/evidence/06-dependencies.md) retains the actual tested source identity and results.

## Acceptance

- [Current acceptance](../acceptances/06-dependencies-20260916.md)
