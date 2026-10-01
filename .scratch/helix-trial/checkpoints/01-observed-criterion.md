# Checkpoint 1: Report an authored observation

Status: scope approved with the initial plan on 2026-10-01. Verification and human checkpoint review are pending. This file is the implementing agent's checkpoint, not the whole work-item plan.

## Outcome

Run a standalone Go fixture with one criterion and a present observation. Emit the criterion ID, description, result, evidence source, and tested revision as a tab-separated row. Preserve each authored string.

## Relevant requirements

The authoritative source is [fixture requirements R1, R3, and R4](../spec.md#fixture-interface-and-requirements). The orchestrator supplies their text as scoped excerpts with the source path and whole-file digest. R2 is outside this checkpoint's outcome.

The executable reads one JSON document from standard input and writes rows to standard output. Successful valid input exits zero. JSON decoding errors produce a diagnostic on standard error and a nonzero status. A supplied `pass` or `fail` observation is authored data. Reading an evidence file or deciding acceptance is outside this program's responsibility.

## Supplied context and dependency contract

The dispatch contains this checkpoint, relevant requirement excerpts, applicable repository guidance, prepared gate expectations, permitted code paths, and a full starting commit SHA. It contains no orchestrator conversation, whole plan, sibling checkpoint, or unrelated work item.

There is no fixture dependency. The expected edit scope is `.scratch/helix-trial/fixture/`. The expectation author owns the executable test assertions and reusable input examples. The implementer changes fixture implementation and reports proposed expectation changes to the orchestrator before editing them.

## Required gates

- An independent runner checks the observed input through the executable interface, including exact authored source and revision preservation. Test literals come from the specification, not the renderer.
- A behavior assertion detects the controlled R1 defect described in the scoped correction assignment. Retain the actual failed result and defect revision. The corrected commit passes the renewed suite and reviews.
- Standards review and specification review both examine the identified checkpoint comparison. Both axes retain cited findings and their own tested revision.
- The user reviews the passing checkpoint's outcome, revision, changed files, results, and unresolved concerns before checkpoint 2 starts.

The expected suite command is `go test ./.scratch/helix-trial/fixture`. The example command is `go run ./.scratch/helix-trial/fixture < .scratch/helix-trial/fixture/testdata/observed.json`. The test author confirms working commands before dispatch. No visual or remote-environment gate applies to this text-output fixture.

## Evidence and completion

Store each runner result and review round under the ignored trial evidence directory, with actual actor, command, exit status, time, environment, source identity, and candidate commit. A correction renews affected checks for its new commit.

Return the committed change, example output, checks run, and any unresolved information. Report discoveries that require changing the checkpoint to the orchestrator. Passing this checkpoint supplies a preserved observed-row contract for checkpoint 2. It does not accept the whole ticket.
