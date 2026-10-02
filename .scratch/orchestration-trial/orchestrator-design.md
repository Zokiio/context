# Project-local orchestrator design

Status: approved for the isolated fixture on 2026-10-01. This design implements the [confirmed workflow choices](../workflow-improvements/orchestration-design.md) for the [fixture trial](spec.md). It leaves the CLI's existing read-only responsibilities intact.

## Invocation and ownership

The proposed caller supplies one ticket:

```text
Use $orchestrate for orchestration-trial/issues/01-explicit-orchestrator-trial.md.
```

Create one project-local skill at `.agents/skills/orchestrate/`, with `SKILL.md` and `agents/openai.yaml`. The latter sets `policy.allow_implicit_invocation: false`. The entrypoint names the ticket, reads its authoritative plan, and coordinates the run. It needs no shell scheduler, new role skills, routing change, hook, or generated state schema.

The orchestrator owns the complete task context, checkpoint assignments, revision selection, correction loop, and evidence-backed review summaries. Existing responsibilities remain with task-context, implementation and TDD, two-axis code review, recovery notes, and the acceptance procedure. The orchestrator's checkpoint delegation is a deliberate scoped adaptation of the full-ticket implementation procedure.

## Artifact contracts

| Artifact | Owner and readers | Mutability and authority |
| --- | --- | --- |
| `spec.md` | Authored requirements, read by orchestrator, expectation author, and final verifier | Changes reflect authorized requirements |
| `implementation-plan.md` | Orchestrator's whole-ticket sequence, read by human and orchestrator | One plan per ticket; contains no live progress |
| `checkpoints/<id>.md` | Checkpoint outcome and gates, read by its workers and reviewers | Separate source for each checkpoint |
| Fixture code, test files, and JSON examples | Implementer and test author, read by runners and reviewers | Reusable source material committed in Git |
| `evidence/<run-id>/progress.md` | Orchestrator's observations and next action | Disposable continuity, explicitly loaded on resume |
| `evidence/<run-id>/feedback.md` | Feedback source, scope, adoption status, affected checkpoints | Retention alone does not create mandatory guidance |
| `evidence/<run-id>/dispatch/<assignment-id>/` | Exact prompt, supplied text, paths and digests, agent identity, starting revision | Immutable dispatch record; proves supplied context only |
| `evidence/<run-id>/<checkpoint>/<revision>/<round>/` | Runner results and separate review reports | Immutable evidence with actual observer, commands, environment, time, source digests, and tested commit |
| Ignored Acceptance record | Existing acceptance procedure | Whole-work-item decision, with human reviews attributed separately |

The ticket selects agreed requirement documents through Spec and Context. Plans link checkpoint files, but the reader does not expand those nested links. The orchestrator explicitly loads them and records their digests. Progress, review reports, and feedback stay outside requirement selection so routine observations do not stale acceptance.

## Delegation contracts

Use the agent environment's isolated subagents with `fork_turns: "none"`. Preserve the assigned scope when delivering any additional context. A worker reports missing required information to the orchestrator. It receives focused source material rather than the whole conversation. Keep one implementing or repairing agent active at a time. Independent reviews may run in parallel.

| Role | Receives | Returns |
| --- | --- | --- |
| Expectation author | Full agreed fixture specification, checkpoint scope, test seam, relevant standards | Requirements-derived cases and checkpoint-scoped executable tests, with source citations and authorship |
| Implementer or repairer | One checkpoint, relevant source excerpts, standards, dependency contract, code paths, start commit, and gate demands | Scoped code change, intended outcome, discoveries, commands it ran, and unresolved concerns |
| Independent runner | Candidate commit, checkpoint and test expectations, commands, environment | Actual output, exit statuses, behavior diagnosis, and tested revision |
| Standards reviewer | Identified comparison, applicable repository standards, two-axis review's smell baseline | Cited rule violations and separately labelled heuristic concerns |
| Specification reviewer | Identified comparison, checkpoint requirements, relevant spec excerpts and dependency contract | Missing behavior, incorrect behavior, or scope beyond the checkpoint |
| Combined verifier | Full ticket and specification, final commit, integration obligations | Independent combined-case observations and criterion-level gaps |

Implementers receive the prepared test expectations as gate demands. Reviewers receive neither the implementer's reasoning nor the other axis's findings. Standards and specification remain separate questions. Supporting agents retain their own observation identity; the orchestrator's summary does not replace it.

Use TDD at the approved executable seam. Expectations can cover both checkpoints up front, but executable test preparation and implementation proceed in checkpoint slices. A previously passing valid regression test stays useful.

## Revision protocol

Before review, stage only the checkpoint's code and reusable tests, then commit a local candidate. Include independently authored tests even if they were prepared earlier. For checkpoint 1, pin `base` to the completed skill-preparation commit. Later checkpoints use the preceding approved checkpoint commit. Pin `target` to the candidate's full SHA. Check that the working tree under the review paths matches `target`, and verify `base` is an ancestor.

Reuse code-review's two independent axes with the identified comparison. Its normal comparison ends at HEAD. For this trial, capture `git diff <base>...<target>` and `git log <base>..<target> --oneline`, scoped to the checkpoint's fixture files, and give both reviewers those exact commands and SHAs. This explicitly adapts the ending revision without changing the shared skill. Unrelated checkout edits remain outside the reviewed result.

Run checks against the same candidate tree before review. A repair creates another commit and renews affected checks and reviews against the new SHA. Keep the original failure and earlier reviews. The cumulative comparison from the checkpoint's base includes the final correction. Earlier evidence retains its original identity and cannot be relabelled as a check of the repaired commit.

Commit identity establishes which files were examined. It does not approve the checkpoint. The user's checkpoint review names `target`, and the next checkpoint starts only after that review and passing gates. Final verification identifies the combined result's revision separately.

## Progress and correction loop

Use descriptive progress observations such as waiting for plan review, preparing expectations, implementing, checking, repairing, waiting for checkpoint review, and verifying the combined result. Advancement depends on the evidence and decision for the evaluated revision. Persisted wording is never the gate result.

Classify feedback before acting:

- A violated requirement, documented constraint, or demonstrated defect blocks the checkpoint and authorizes an in-scope repair.
- An uncertain correctness concern requires investigation before treating it as a defect or clearing it.
- A preference remains a proposal until a decision adopts it. Broader mandatory guidance requires the user's approval with source and scope.

When a requirement or dependency contract changes, identify affected checkpoints and prior evidence. Assess existing authorization, uncertainty, alternatives, and consequences for completed work. Continue unrelated work. Resolve a remaining material choice before affected implementation, preserving earlier evidence and refreshing expectations.

An unavailable check keeps the checkpoint unverified. Repeated repair attempts need concrete progress in diagnosis or fewer failures. Escalate when they cease making progress instead of imposing an unlimited retry rule. Human-review waits end the active implementation turn normally.

## Recovery and completion

Use task-context and recovery-notes for the orchestrator's full ticket. Preserve the exact context JSON that governed the work. Inspect current source digests, candidate revisions, evidence, and live agents before continuing. A digest or revision difference requires impact assessment; a progress marker cannot prove verification freshness or writer termination.

After all checkpoint reviews, commission the independent combined verifier. The final acceptance decision follows the existing procedure, snapshots selected requirement sources and explicitly loaded checkpoint requirements, and links ignored immutable evidence. A workflow actor may author that decision; only actual user responses become human approval entries. Accepted checkpoint revisions and the final tested revision remain distinguishable.

## Borrowing from the pinned source

The [source walkthrough](../workflow-improvements/helix-loop-reference.md) supplies useful detail for the expectation-author, implementer, runner, and repair handoffs. The [upstream build roles](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/orchestrator/SKILL.md#L53-L64) and [review loop](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/adversarial-review/SKILL.md#L20-L29) inform those procedures. The [feedback template](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/templates/learnings-template.md#L1-L15) informs source retention.

The [assessment](../workflow-improvements/helix-loop-assessment.md) explains why this design uses separate checkpoint files, evidence-backed gates, explicit waits, scoped feedback adoption, and ignored verification records. Upstream installation and state-label enforcement remain outside the trial.
