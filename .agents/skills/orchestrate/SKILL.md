---
name: orchestrate
description: Coordinate an explicitly requested checkpoint implementation and correction loop for a named work item.
---

# Orchestrate a work item's checkpoints

Use this skill when the user explicitly requests checkpoint orchestration for a named ticket or approves a plan that invokes it. Keep the full ticket context here. Delegate implementation through fresh checkpoint contexts.

## Establish the current basis

Read the project's `AGENTS.md`, tracker guidance, specification, and one implementation plan linking separate checkpoint files. Use [task-context](../task-context/SKILL.md) to collect the full ticket requirements and [recovery-notes](../recovery-notes/SKILL.md) to inspect continuation state. Preserve explicit project scope and the exact context JSON used. Read the returned source text, not just its inventory.

Check the assigned ticket's readiness and the user's existing authorization separately. Report unrelated partial diagnostics without treating them as a new dependency. Resolve missing required context before dependent work. Inspect current source digests, code revisions, recovery candidates, and live assignments before relying on earlier observations. A progress label cannot establish a passed gate or a stopped writer.

The plan owns sequence and dependencies. Each checkpoint owns its outcome and required gates. Keep mutable progress and source-scoped feedback separate from those requirements. Keep generated evidence and Acceptance records in the project's ignored locations.

For this trial, apply the [confirmed workflow choices](../../../.scratch/workflow-improvements/helix-design.md). Reuse an existing initial-plan approval when it covers the current scope and test seam. If a material choice remains, prepare a concrete reviewable proposal and wait for the decision before dependent implementation. Waiting for human review is a valid pause.

## Prepare independent expectations and scoped assignments

Commission an expectation author separate from implementation. Give it the agreed requirements, checkpoint outcomes, approved test seam, and relevant constraints. Retain its cases and source identities before implementation. Prepare executable tests one checkpoint at a time through [TDD](../tdd/SKILL.md). Keep valid regression tests that already pass. A missing executable or compile failure is baseline evidence; a behavior assertion must detect the intended missing behavior or defect.

Use the agent environment's delegation tools. Create implementing and repairing agents with `fork_turns: "none"`; otherwise use an equivalent context boundary that supplies no parent conversation. Retain the exact dispatch request and material supplied, its source paths and digests, worker identity, and starting commit.

An implementation packet contains only:

- The assigned checkpoint file and relevant requirement excerpts, with authoritative source references.
- Applicable guidance, permitted code paths, dependency contracts, and the starting commit.
- Independently prepared gate demands and their evidence expectations.
- Applicable adopted guidance and task-local findings for a repair.

Supply excerpts as actual text. A pointer list is insufficient. Keep the whole plan, sibling checkpoint files, other workers' conversations, and full ticket-context JSON out of the implementer's packet. This deliberately adapts the full-ticket implementation workflow. If information is missing, return focused context through the orchestrator rather than asking the worker to reconstruct the plan. Retaining a scoped packet proves supplied context, not restricted filesystem access.

Keep one implementing or repairing agent active at a time. Supporting expectation authors, runners, and reviewers retain their own roles and observation identities. Inspect existing assignments before starting another writer.

## Check and correct an identified revision

Commit the scoped candidate code and reusable tests before review. Pin a checkpoint base and full target SHA. Verify the base is an ancestor and that the reviewed paths match the target. Include prepared tests in the comparison even if their author wrote them earlier. Archive the exact diff command and commit list.

Commission an independent runner on that revision. Retain commands, output, exit statuses, actual observer, time, environment, tested commit, and source digests. An unavailable required check leaves the checkpoint unverified. Continue only work independent of that result and escalate what is needed to run the check.

Reuse [code-review](../code-review/SKILL.md)'s independent standards and specification axes. Give both reviewers the exact `git diff <base>...<target>` comparison and commit list. This adapts its usual HEAD endpoint to the pinned target. Give standards reviewers applicable rules and its smell baseline. Give specification reviewers the checkpoint requirements and dependency contract. Reviewers receive neither implementation reasoning nor the other axis's report. Retain both verdicts separately.

Unmet requirements, violated documented constraints, and demonstrated defects block advancement. Investigate uncertain correctness concerns. Retain preferences as proposals until adopted. A broader mandatory rule needs the user's approval with its source and scope.

Dispatch an in-scope repair to a fresh implementing context. Preserve the original failure and reviewed revision. Commit the repair, rerun affected checks, and renew the affected review axes for the changed code. Include visual or environment checks when the checkpoint requires them. Escalate unavailable verification or repairs that stop producing new evidence or reducing known failures.

When requirements or dependency contracts change, assess affected checkpoints, prior evidence, uncertainty, alternatives, and existing authorization. Reuse authorized decisions. Resolve a remaining material choice before affected implementation. Preserve earlier evidence and refresh context and expectations. Unrelated work may continue. Split or reorder checkpoints only within agreed outcomes, dependencies, and constraints, recording the reason.

## Present the checkpoint and wait for review

After required checks and both review axes pass for the current target, present its outcome, starting and reviewed commits, changed files, runnable example, gate evidence, corrections, and unresolved concerns. Ask the user to review that identified revision.

For this first trial, wait for the actual human checkpoint review before dispatching the next checkpoint. Retain the response, source, scope, actor, and reviewed revision. A workflow verdict is not human approval. Apply authorized task-local corrections and renew affected checks; broader guidance still requires adoption. Keep review timing and clarification observations outside the plan.

## Verify the whole result

After all human checkpoint reviews, commission a fresh combined-result verifier that did not implement or prepare the tests. Supply the full ticket, specification, final revision, and integration obligations. It independently evaluates the combined behavior and identifies unmet criteria.

Checkpoint completion is progress. Whole-ticket acceptance follows [Record acceptance](../../../docs/agents/acceptance.md) only after every criterion has evidence. Snapshot selected sources and explicitly loaded checkpoint requirements. Retain original observations and actual human approval attribution. Verify this ticket's valid, fresh acceptance and report unrelated diagnostics separately.

Assess the approved trial measures from actual observations. Missing timing or a comparison baseline remains unmeasured. Propose broader adoption separately from accepting this work item.
