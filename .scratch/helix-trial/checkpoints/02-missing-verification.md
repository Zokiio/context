# Checkpoint 2: Report missing verification

Status: scope approved with the initial plan on 2026-10-01. Dispatch follows passing gates and the user's review of checkpoint 1.

## Outcome

Keep a criterion visible when its observation is absent or JSON-null. Preserve its ID and description. Emit `missing` as the result and `-` for the evidence source and tested revision. Preserve observed rows in the same invocation.

## Relevant requirements

The authoritative source is [fixture requirements R1 through R4](../spec.md#fixture-interface-and-requirements). The orchestrator supplies only those requirements and the missing-case example, with source path and digest. The implementing agent does not need the workflow evaluation sections or the whole plan.

No observation means no verification. It cannot become a passing observation. The fixture reports authored data and emits no work-item acceptance verdict.

## Supplied context and dependency contract

The dispatch contains this checkpoint, relevant excerpts, applicable guidance, gate expectations, edit paths, and the preceding approved commit. The dependency contract is the existing executable interface and preservation of observed rows, including authored `pass` and `fail` results. Relevant fixture code and tests are available at that commit.

Use a fresh implementing context without the prior implementation conversation, full plan, or sibling checkpoint file. Implement only under `.scratch/helix-trial/fixture/`. The expectation author owns new executable assertions and reusable missing-case input. Proposed expectation changes go to the orchestrator with their requirements basis.

## Required gates

- The independent runner checks absent and JSON-null observations through the executable seam. Both preserve criterion identity and description and emit the required missing markers.
- Checkpoint 1's observed-row tests pass again. A valid existing regression test may already pass.
- Standards and specification reviewers examine the checkpoint's identified base and target revisions independently. Corrections renew affected checks and review.
- The user reviews the passing checkpoint at its exact revision before combined-result verification.

The expected suite command is `go test ./.scratch/helix-trial/fixture`. The example command is `go run ./.scratch/helix-trial/fixture < .scratch/helix-trial/fixture/testdata/missing.json`. The independent test author confirms working commands. No visual or remote-environment gate applies.

## Evidence and completion

Retain each result and review in ignored evidence, identifying its actual observer, source digests, environment, and tested commit. Present the missing-row example and checkpoint 1 regression result in the human summary.

Return the committed change, output, checks run, and unresolved concerns. Completing this checkpoint leaves the independent combined-result check and whole-ticket acceptance outstanding.
