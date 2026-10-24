# Resolve disputes through an independent judge

The implementer must be able to challenge requirements or protected tests without changing either. The factory currently tells it to stop and explain, but ordinary repair reuses the same frozen admission. A dispute needs a retained decision and a concrete handoff.

## Approved behavior

The user approved an independent judge and requested that the Run view show its decision and the next responsible agent. Deliver this work in the existing combined PR #55.

An implementer submits a structured report identifying the affected criterion or test, observed failure, evidence, and proposed correction. It stops without committing a candidate. An authorized coordinator can also report a dispute on a stopped failed or reviewable run. Ordinary build and execution failures continue to use the existing repair and reconciliation behavior.

A separate read-only judge receives the frozen requirements, protected test snapshot, relevant candidate and execution evidence, and the report. Its recorded decision routes implementation defects to the builder, incorrect or missing tests to the test author, and unclear or changed intent to the person. The judge decides; the test author edits; the independent verifier checks the result. The judge cannot directly edit requirements, tests, or implementation.

Test changes preserve the original run and admitted tests. The test author prepares a new protected patch from the applicable snapshot, with the judge's directive. The judge then reviews the exact proposed revision, its cumulative diff, coverage, and author concerns against the original requirements and tests. It can accept the proposal and pass work to the builder, return it to the test author with specific improvements, or ask a person when the correct result is uncertain. The user delegated this proposal review to the judge on 24 October 2026. A correction that preserves intent and acceptance strength can proceed without another manual test approval. Changed behavior, reduced acceptance, ambiguous judgment, or a repeated implementation dispute requires a person.

Each review retains the proposed revision, decision, rationale, evidence, model run, and next owner. Improvements start from the previous proposal and preserve valid coverage. The judge reviews at most three proposals per dispute, allowing two returns for improvement before human escalation. A replacement run links to the original and uses the judge-accepted test revision with fresh verification and the normal final review policy. Returning a task to planning must release its old claim only after execution is known to have stopped and keep automatic dispatch held until the person resumes it.

One automatic implementation-dispute cycle is allowed per run lineage. Its bounded test-proposal review loop does not create another implementation dispute. Budget, deadline, cancellation, authorization, expected-version checks, idempotent operations, restart recovery, and uncertain execution retain their existing meaning. Stop prevents late judge or test-author results from dispatching more work.

## Run and History

Run shows a conditional Judge step and a decision panel with the report, decision, rationale, evidence references, routing status, and next responsible agent. It distinguishes initial judgment, proposal review, a return to the test author, and acceptance that hands work to the builder. Links connect original and replacement runs. History retains each proposal review and prior failures. Previously pending human test proposals can be sent to the judge for review. Read-only users can inspect but cannot change decisions. Forms preserve writing during polling and stale submissions require a fresh decision.

## Verification

Use deterministic subprocess agents to exercise report-to-judge-to-builder, test-author revision and replacement-run verification, human escalation, repeated disputes, stale requests, permissions, cancellation, interrupted judgment, and retained evidence. Verify the actual browser views and routing controls. Keep generated trial data and acceptance evidence out of Git. Live-provider compatibility must be reported separately from fixture checks.
