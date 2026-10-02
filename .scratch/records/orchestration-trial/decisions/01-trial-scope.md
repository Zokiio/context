---
type: Decision
id: 85d62f04-b7cb-41b2-931a-96443a926b39
title: Select the first orchestration trial task and review its initial plan
status: stable
decisionState: resolved
---

# Select the first orchestration trial task and review its initial plan

This record resolves Q21 from the [confirmed workflow design](../../../workflow-improvements/orchestration-design.md#trial-scope) for the [fixture trial](../issues/01-explicit-orchestrator-trial.md).

## Existing authorization

The handoff authorizes preparation of a concrete trial plan and orchestrator design, followed by implementation once scope is settled. The 19 workflow choices are already confirmed. Choice 8 requires user review of the initial plan and each verified checkpoint. Q21 expressly leaves the fixture versus an approved real ticket unresolved.

The initial handoff left the task choice open. The later human response recorded below selects the fixture and authorizes starting the presented plan. Routine filenames, fixture layout, and orchestration details remain implementation choices.

## Proposal

Approve the [isolated two-checkpoint plan](../../../orchestration-trial/implementation-plan.md). This selects the fixture and reviews its proposed executable test seam. It authorizes creating the project-local skill and running checkpoint 1 through its gates, then presenting the checkpoint for the agreed human review.

The alternative is to name an approved real work item. That requires a new task-specific plan and checkpoint gates before its initial-plan review. The confirmed workflow choices remain applicable.

## Impact

The fixture isolates context delivery, correction evidence, and review summaries from production behavior. It cannot establish real-ticket usefulness or a reduction in review time without comparative observations. Its reusable code and tests belong in Git; generated evidence and Acceptance records remain ignored.

## Resolution

On 2026-10-01, the human user in Codex chat `01a0f86c-52b2-7d13-a11c-83eb111c1912` instructed: "lets start with isolated fixture then we can test on a ticket in axpilot".

This selects the isolated fixture and authorizes starting the plan presented at proposal commit `a79407d3e294d77dd97981ac57205988fb9c8eaa`. It approves the initial plan and its standard-input, standard-output, and exit-status test seam. The existing human review after each passing checkpoint remains required. The workflow recorded the response and reviewed document identities in local trial evidence; it does not claim to authenticate the human actor.

After the fixture trial, assess a ticket in Axpilot for a real-work trial. No Axpilot ticket has been selected here.
