# Resolve disputes through an independent judge

The implementer must be able to challenge requirements or protected tests without changing either. The factory currently tells it to stop and explain, but ordinary repair reuses the same frozen admission. A dispute needs a retained decision and a concrete handoff.

## Approved behavior

The user approved an independent judge and requested that the Run view show its decision and the next responsible agent. Deliver this work in the existing combined PR #55.

An implementer submits a structured report identifying the affected criterion or test, observed failure, evidence, and proposed correction. It stops without committing a candidate. An authorized coordinator can also report a dispute on a stopped failed or reviewable run. Ordinary build and execution failures continue to use the existing repair and reconciliation behavior.

A separate read-only judge receives the frozen requirements, protected test snapshot, relevant candidate and execution evidence, and the report. Its recorded decision routes implementation defects to the builder, incorrect or missing tests to the test author, and unclear or changed intent to the person. The judge decides; the test author edits; the independent verifier checks the result. The judge cannot directly edit requirements, tests, or implementation.

Test changes preserve the original run and admitted tests. The test author prepares a new protected patch from the applicable snapshot, with the judge's directive. Human approval is required under the default ask policy. Existing explicit delegation may approve a proposal only when it preserves intent, does not weaken acceptance, and has no test-author concerns. Changed behavior, reduced acceptance, ambiguous judgment, or a repeated dispute requires a person. A replacement run links to the original and uses the newly accepted test revision with fresh verification and approval. Returning a task to planning must release its old claim only after execution is known to have stopped and keep automatic dispatch held until the person resumes it.

One automatic dispute/revision cycle is allowed per run lineage; repeated disagreement goes to a person. Budget, deadline, cancellation, authorization, expected-version checks, idempotent operations, restart recovery, and uncertain execution retain their existing meaning. Stop prevents late judge or test-author results from dispatching more work.

## Run and History

Run shows a conditional Judge step and a decision panel with the report, decision, rationale, evidence references, routing status, and next responsible agent. It distinguishes a proposed destination from waiting for human approval. Links connect original and replacement runs. History retains prior decisions and failures. Read-only users can inspect but cannot change decisions. Forms preserve writing during polling and stale submissions require a fresh decision.

## Verification

Use deterministic subprocess agents to exercise report-to-judge-to-builder, test-author revision and replacement-run verification, human escalation, repeated disputes, stale requests, permissions, cancellation, interrupted judgment, and retained evidence. Verify the actual browser views and routing controls. Keep generated trial data and acceptance evidence out of Git. Live-provider compatibility must be reported separately from fixture checks.
