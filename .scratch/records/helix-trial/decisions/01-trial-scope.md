---
type: Decision
id: 85d62f04-b7cb-41b2-931a-96443a926b39
title: Select the first Helix trial task and review its initial plan
status: draft
decisionState: open
---

# Select the first Helix trial task and review its initial plan

This record retains the unresolved Q21 from the [confirmed workflow design](../../../workflow-improvements/helix-design.md#current-questions). It blocks only the [proposed fixture trial](../issues/01-explicit-orchestrator-trial.md). The choice remains open after the new-chat handoff.

## Existing authorization

The handoff authorizes preparation of a concrete trial plan and orchestrator design, followed by implementation once scope is settled. The 19 workflow choices are already confirmed. Choice 8 requires user review of the initial plan and each verified checkpoint. Q21 expressly leaves the fixture versus an approved real ticket unresolved.

No existing human response selects a real ticket or approves the fixture plan. The instruction to continue does not resolve the explicitly retained choice. Routine filenames, fixture layout, and orchestration details can be prepared without another workflow interview.

## Proposal

Approve the [isolated two-checkpoint plan](../../../helix-trial/implementation-plan.md). This selects the fixture and reviews its proposed executable test seam. It authorizes creating the project-local skill and running checkpoint 1 through its gates, then presenting the checkpoint for the agreed human review.

The alternative is to name an approved real work item. That requires a new task-specific plan and checkpoint gates before its initial-plan review. The confirmed workflow choices remain applicable.

## Impact

The fixture isolates context delivery, correction evidence, and review summaries from production behavior. It cannot establish real-ticket usefulness or a reduction in review time without comparative observations. Its reusable code and tests belong in Git; generated evidence and Acceptance records remain ignored.

## Resolution

Awaiting the user's choice and initial-plan review. This text is not an approval. Record the actual response, source, date, approved document identities, and limits here before setting `decisionState: resolved`.
