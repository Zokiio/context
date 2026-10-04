---
name: orchestrate
description: Coordinate an explicitly requested checkpoint implementation and correction loop for a named work item.
---

{{if .Development}}<!-- Generated from internal/initialization/templates/orchestrate.md. Run go generate ./internal/initialization after editing the template. -->

{{end -}}
# Orchestrate a work item's checkpoints

Use this skill when the user explicitly requests checkpoint orchestration for a named work item or approves a plan that invokes it. The orchestrator retains the whole task. Implementing and repairing agents receive fresh context for one checkpoint.

This is an opt-in workflow in the existing agent environment. The ctx readers supply records and evaluations. They do not route tasks, supervise execution, or manage implementation claims.

## Establish the current basis

Read the project's instructions and [tracker guidance]({{.TrackerLink}}). Use [task-context]({{.TaskContextLink}}) to collect and read the full ticket requirements with explicit project scope. Keep the exact context JSON and source identities used. Use [recovery-notes]({{.RecoveryNotesLink}}) to inspect continuation state.

Read the specification and implementation plan, then explicitly load its separate checkpoint files, progress, and feedback. Reader selection includes explicitly linked whole documents. Links inside a selected plan do not automatically load checkpoint files. The plan owns sequence and dependencies. Each checkpoint owns its outcome and gates. Keep mutable progress, feedback, and evidence separate from requirements. Assess feedback's source, scope, adoption status, and affected checkpoints before applying it.

Inspect current source digests, code revisions, recovery candidates, and live assignments before relying on earlier observations. A progress label cannot establish a passed gate or a stopped writer. Check readiness and the user's authorization separately. Report unrelated partial diagnostics without making them a new dependency. Resolve missing required context before dependent work.

Identify the applicable project policy for human review, publication, and acceptance. Reuse existing authorization when it covers the current scope and test seam. If a material choice remains, prepare a concrete proposal and wait for the needed decision before dependent implementation.

## Prepare independent expectations and scoped assignments

Each checkpoint file is its implementing agent's packet. Before dispatch, confirm that it holds:

- The outcome, requirements, relevant acceptance criteria, and authoritative source identities.
- Approved seams, gates, and required evidence.
- Permitted code paths and the dependency contract.

Complete a missing item from the agreed plan and requirements, or seek the needed decision, before dispatch.

Before production edits, commission an expectation author separate from implementation. Give that author the checkpoint path and digest, starting revision, and relevant constraints. Read the "Prepare independent expectations" section of [Verify a checkpoint independently](VERIFICATION.md) before commissioning checks. Retain the resulting cases, original baseline observations, and source identities.

Use the agent environment's delegation tools. Create implementing and repairing agents with `fork_turns: "none"`, or use an equivalent fresh context boundary without parent conversation. Retain each exact dispatch with its paths and digests, worker identity, and starting or evaluated revision.

Dispatch implementation, repair, runner, and review work by pointer. Each dispatch names the role, the checkpoint path and digest, the starting or evaluated revision, and applicable shared guidance by path and digest. An implementation or repair dispatch also names the commit with the prepared expectations, and a repair dispatch names its findings file. The worker reads the checkpoint file itself, so a dispatch does not repeat its text.

Exclude the whole plan, sibling checkpoint files, full task-context JSON, and other workers' conversations. Return focused missing context through the orchestrator instead of asking the worker to reconstruct the plan. A scoped packet proves supplied context, not filesystem isolation.

Keep one implementation or repair writer active at a time. Confirm existing assignments before starting another writer. Expectation authors, runners, and reviewers retain separate roles and observation identities.

## Check and correct an identified revision

Commit the scoped candidate and reusable tests. Pin the full target SHA and the agreed review base. The complete base-to-target comparison must include independently prepared tests, even when written before implementation. Choose and record a base that includes those changes. Verify that the base is an ancestor and retain the exact diff command and commit list.

Commission an independent runner and separate Standards and Specification reviewers on that identified revision. Read the "Run checks and both review axes" section of [Verify a checkpoint independently](VERIFICATION.md) for their complete briefs. Reviewers receive neither implementation reasoning nor the other axis's report. Retain commands, output, exit statuses, observer, time, environment, source identities, and tested SHA in each full report's evidence file. Keep the Standards and Specification reports in separate files. Pass full reports by path to repair dispatches and acceptance. A different HEAD or an uncommitted tree cannot establish a gate on the candidate.

Unmet requirements, documented constraint violations, and demonstrated defects block advancement. Investigate uncertain correctness concerns. Keep preferences as proposals until adopted. An unavailable required check remains unverified. Continue independent work and escalate what is needed to resolve the gap.

For a grounded finding, retain the original failed evidence and dispatch an in-scope repair in a fresh scoped context. Commit the correction, rerun affected checks, and renew affected review axes on the new revision. Include required visual and environment checks. Diagnose or escalate a repair loop that stops producing new evidence or reducing known failures.

If requirements or dependency contracts change, assess affected checkpoints and existing authorization. Preserve earlier evidence. Resolve needed choices, refresh scoped context, and renew affected expectations and verification. Split or reorder checkpoints only within agreed outcomes and dependencies, and record the reason.

## Apply the project's human gates

After required checks and both review axes pass on the current revision, present the outcome, base and target commits, changed files, runnable example, evidence, corrections, and unresolved concerns.

Where project policy requires human review before advancement, wait for the actual response to that identified candidate before dependent implementation. Retain the response, source, actor, scope, and reviewed revision. An agent verdict cannot supply human approval. Reuse prior authorization within its actual scope. Apply authorized corrections and renew affected verification.

## Verify the whole result

Commission a fresh combined-result verifier that neither implemented nor prepared the reusable tests. Supply the full ticket, specification, final revision, and integration obligations. The verifier derives independent combined observations and identifies remaining criteria. Read the "Run checks and both review axes" section of [Verify a checkpoint independently](VERIFICATION.md) for the full report and summary it returns. Prepare these observations before final human review when that work is independent of the review.

Checkpoint completion records progress. Whole-work acceptance requires evidence for every criterion under [Record acceptance]({{.AcceptanceLink}}). Snapshot selected sources and explicitly loaded checkpoint requirements. Keep acceptance, human approval, publication, and tracker updates separately attributed under project policy. A successful merge is not acceptance. Report remaining uncertainty and unrelated diagnostics separately.
