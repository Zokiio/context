---
type: WorkItem
id: 2cc93a96-a641-4259-910f-d8ed0fd679b4
title: Exercise explicit checkpoint orchestration on an isolated fixture
status: stable
triage: ready-for-agent
execution: completed
---

# Exercise explicit checkpoint orchestration on an isolated fixture

The user selected the fixture and authorized starting the initial plan on 2026-10-01. Its independent outcome is a reviewable evaluation of the agreed loop through two small fixture behaviors. Human checkpoint reviews remain required.

## Scope

Create the smallest explicitly invoked project-local orchestrator skill and exercise it on the observed-criterion and missing-verification fixture. Reuse existing context, implementation, test preparation, two-axis review, recovery, and acceptance responsibilities. No CLI feature or broader automatic workflow adoption is included.

## Acceptance criteria

- A1: The project-local skill has explicit-only invocation, valid packaging, and an observed run that preserves the confirmed orchestration responsibilities and human review policy.
- A2: The fixture satisfies specification R1 through R4. Both outcomes have separate checkpoint files, revision-specific behavior evidence, independent standards and specification reviews, and actual human checkpoint reviews.
- A3: Retained dispatch records establish fresh scoped implementing contexts, independent expectations prepared before implementation, and sequential implementation. Feedback retains its source and scope; broader mandatory guidance appears only with explicit adoption.
- A4: A real failed behavior check or grounded review finding has retained evidence, a correction revision, and renewed affected checks and reviews. No failed revision advances as verified.
- A5: A fresh independent verifier checks the combined result against the full requirements before whole-ticket acceptance. Its result identifies the actual final commit and unmet criteria, if any.
- A6: The trial assessment reports review effort, context and authority, correction integrity, and combined verification with supporting evidence and limits. It records unavailable measurements and makes no comparative review-time claim without a baseline.

## Blocked by

None

## Blocked by decisions

- [Select the trial task and review its initial plan](../decisions/01-trial-scope.md)

## Spec

- [Fixture and workflow trial specification](../../../orchestration-trial/spec.md)

## Context

- [Confirmed workflow choices](../../../workflow-improvements/orchestration-design.md)
- [Original trial proposal and success measures](../../../workflow-improvements/orchestrator-trial.md)
- [Implementation plan](../../../orchestration-trial/implementation-plan.md)
- [Orchestrator design](../../../orchestration-trial/orchestrator-design.md)
- [Trial scope decision](../decisions/01-trial-scope.md)
- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Tracker conventions](../../../../docs/agents/issue-tracker.md)

## Acceptance

- [Accept the isolated orchestration fixture trial](../acceptances/01-fixture-trial.md)

## Comments

2026-10-01: Prepared a concrete proposal from the handoff and the 19 confirmed choices. Q21 remains open. No trial code, installed skill, automatic routing, hook, or acceptance decision has been created.

Preparation returned complete task context. Project-wide resumption remained partial because the local `ctx-init` Acceptance record was unavailable. The trial's criteria, required context, and dependency declaration checks passed; its scope decision remained open. No trial recovery notes existed.

2026-10-01: The user instructed starting the isolated fixture, resolving Q21 and initial-plan review. Begin skill preparation and checkpoint 1. A later trial on an Axpilot ticket is intended; that ticket remains unselected.

2026-10-01: The user requested a different project name while deciding on a final name. Use "Orchestration trial" temporarily. The trial and record directories, workflow design, commands, and links now use that name. WorkItem and Decision IDs remain unchanged. Earlier evidence, source snapshots, approvals, and published branch references retain their original paths and revisions; the original ignored evidence directory remains in place. The rename changes naming and paths, with the fixture requirements and implementation unchanged. Human checkpoint 2 review remains pending.

2026-10-01: The user authorized continuation past checkpoint 2. A fresh independent verifier checked four derived executable cases, the seven fixture regressions, and retained workflow provenance at 19de8f9694bab33bda00278ecb9890c316e8a4d0. Standards and Specification independently pass the final comparison. Repository tests, race checks, vet and build pass. The independent assessment audit supports A1-A6 with no remaining gaps. Human timing and comparative review effort remain unmeasured. Axpilot #241 is proposed as a separate next trial.
