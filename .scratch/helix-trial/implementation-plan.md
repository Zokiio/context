# Run the first orchestrator trial

Status: proposed, awaiting the [scope and initial-plan decision](../records/helix-trial/decisions/01-trial-scope.md). This is the implementation plan for [one trial work item](../records/helix-trial/issues/01-explicit-orchestrator-trial.md). The [specification](spec.md) owns behavior. The [orchestrator design](orchestrator-design.md) owns the proposed delegation procedure.

## Review this proposal

The recommended task is the isolated fixture. No existing ticket has been selected as its replacement. Approving this proposal settles Q21, the fixture's executable test seam, and the initial plan review. It preserves the agreed human review after each checkpoint.

Expected human reviews are the initial plan, checkpoint 1, and checkpoint 2. The final independent result and trial assessment follow the second checkpoint review. Scope changes or unavailable verification may require an earlier decision.

## Prepare the run after approval

1. Record the user's scope answer and initial-plan review with source and date. Resolve the blocking decision and retain any limits. Add the trial ticket to Current commitments only once chosen.
2. Create `feat/helix-orchestrator-trial` from the inspected checkout. Preserve unrelated files. Commit only the approved trial records, supplied research documents if they belong in this change, and later scoped trial changes.
3. Build the current `ctx` in an ignored run directory. Read the full ticket using task-context, retain the exact returned JSON, and inspect `ctx resume` and orientation. Keep progress, feedback, context packets, and evidence under `.scratch/helix-trial/evidence/<run-id>/`. Use recovery notes when they retain information that authoritative records and current code cannot supply.
4. Commission an expectation author with the agreed specification, orchestrator design, and checkpoint files. Retain its requirements-derived skill evaluation cases and fixture gate expectations before writing skill or fixture code.
5. Create `.agents/skills/orchestrate/SKILL.md` and its explicit-invocation metadata. Use skill-creator and writing-for-agents. Validate the skill and commission an isolated behavioral check of its context delivery and waiting behavior. Correct demonstrated failures before using it on the fixture. General implementation and review skills remain unchanged.
6. Invoke `$orchestrate` explicitly for this ticket with the prepared expectations. The expectation author prepares executable fixture tests as each checkpoint begins.

Skill construction is preparation for the two behavior checkpoints. Review its identified commit independently on standards and the agreed design. Skill metadata validation checks packaging; it does not establish that orchestration works.

## Implement two checkpoints in order

| Checkpoint | Outcome | Dependency | Required before advancement |
| --- | --- | --- | --- |
| [01: Report an authored observation](checkpoints/01-observed-criterion.md) | One row preserves criterion, result, evidence source, and tested revision | Approved plan, skill preparation, independent expectations | Behavior checks, standards review, specification review, correction exercise, and human checkpoint review |
| [02: Report missing verification](checkpoints/02-missing-verification.md) | A criterion without an observation remains visible as missing | Approved checkpoint 1 revision and its contract | Missing-case and checkpoint 1 regression checks, both review axes, and human checkpoint review |

For each checkpoint, the expectation author prepares executable tests at the agreed seam. A fresh implementer receives that checkpoint's packet and implements only its outcome. The independent runner executes the gate commands on a committed candidate revision. Standards and specification reviewers examine that revision independently. Repairs use fresh scoped implementing context and renew the affected checks.

Checkpoint 1 includes the controlled R1 defect described in the specification. Introduce it only after an executable observed row exists. The gate runner receives the requirements, candidate revision, and prepared tests, without the intended repair. Its actual failed result supplies the correction exercise. Preserve every round.

Checkpoint 2 preserves checkpoint 1's executable contract. Its implementer receives the accepted dependency contract and relevant code, without checkpoint 1's implementation conversation or the full plan.

## Verify and assess the combined result

Commission a verifier that did not implement or prepare the fixture tests. Give it the full ticket, specification, final commit, and required integration checks. It derives a combined input independently and checks observed and missing rows together. It also checks null observations, an authored `fail` result, and malformed JSON at the same executable seam. These exercise the existing requirements without creating another outcome.

Run the fixture suite and the repository's normal required checks on the final result. The fixture suite needs an explicit package path because `go test ./...` omits `.scratch/`. Report check coverage accurately. A production-suite pass does not establish fixture behavior.

Write an evidence-backed trial assessment for the four measures in the specification. Retain human review attribution and report unavailable timing or comparative data explicitly. Resolve failed required measures before acceptance. Then apply the repository's acceptance procedure and verify this ticket's valid, fresh acceptance in orientation. Snapshot both checkpoint files as additional accepted requirements because the reader does not expand links inside the plan. Report unrelated partial diagnostics separately. Keep any request for real-ticket adoption or broader guidance separate from this fixture's acceptance.

## Review summary for each checkpoint

Present the following in one compact summary:

- Intended outcome and the checkpoint file.
- Starting and reviewed commit, changed files, and a runnable example with observed output.
- Behavior result and separate standards and specification verdicts, each linked to evidence.
- Corrections, renewed checks, unavailable checks, and unresolved concerns.
- The requested human review of that exact revision.

Record review time as human-reported time or as elapsed request-to-response time with that limitation. Count substantive clarification questions. A human response may authorize an in-scope correction; it does not automatically adopt broader mandatory guidance.

## Changes and interruptions

The orchestrator may split or reorder work within approved outcomes and dependencies, recording the reason separately from progress. Apply confirmed task-local corrections. For changed commitments, assess impact and existing authorization before deciding whether another human choice is required.

Unavailable required checks leave the affected checkpoint unverified. Continue only independent preparation. Escalate repairs when attempts stop producing new evidence or reducing known failures. Waiting for a human review is a valid pause.

On resumption, inspect authoritative sources, current Git revisions, recovery candidates, and live agent status before using retained results. Renew affected evidence when code or requirements differ. A status label never establishes that a gate passed or a previous writer stopped.
