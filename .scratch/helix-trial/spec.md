# Explicit orchestrator fixture trial

Status: approved for the isolated fixture on 2026-10-01 through the [scope decision](../records/helix-trial/decisions/01-trial-scope.md). The [workflow design](../workflow-improvements/helix-design.md) owns the confirmed workflow policy. This specification defines the fixture and trial obligations.

## Outcome

Exercise the checkpoint-and-correction loop through one explicitly invoked, project-local orchestrator skill. Use a standalone Go fixture under `.scratch/helix-trial/fixture/`. It reports authored criterion-to-evidence mappings without deciding acceptance. The two outcomes are an observed criterion row and a row showing missing verification.

This is reusable evaluation material outside production packages. It adds no `ctx` command, record schema, dependency, automatic routing, hook, global skill, managed claim, deployment, or viewer.

## Fixture interface and requirements

The proposed test seam is the executable's standard input, standard output, and exit status. Run it from the repository root with `go run ./.scratch/helix-trial/fixture`. Tests exercise the same executable interface. They do not inspect private rendering helpers.

Input is one JSON document with a `criteria` array. Each entry has `id`, `description`, and an optional `observation`. A present observation has authored `result`, `source`, and `testedRevision` strings. The trial uses valid entries, single-line strings without tabs, and results `pass` or `fail`. General input validation is outside this fixture.

The fixture requirements are:

- R1: Emit one tab-separated row per input criterion, in input order, ending each row with a newline. Fields are criterion ID, description, observation result, evidence source, and tested revision. Preserve the authored strings.
- R2: For an absent or JSON-null observation, preserve the ID and description. Emit `missing` as the result and `-` for both evidence source and tested revision. Never infer a passing observation from absence.
- R3: Present a supplied observation as an authored result. Do not read its evidence source, run checks, compare its tested revision with Git HEAD, or emit a work-item acceptance verdict.
- R4: Both row kinds work in the same invocation. Successful valid input returns exit status zero. JSON decoding errors return a nonzero status and a diagnostic on standard error.

The observed-case example is:

```json
{"criteria":[{"id":"C1","description":"Preserve authored mapping","observation":{"result":"pass","source":"checks/observed.txt","testedRevision":"fixture-rev-1"}}]}
```

Its output is the following row. Separators shown as `\t` and the ending shown as `\n` represent actual tab and newline bytes.

```text
C1\tPreserve authored mapping\tpass\tchecks/observed.txt\tfixture-rev-1\n
```

The missing-case example is:

```json
{"criteria":[{"id":"C2","description":"Verification remains missing"}]}
```

Its output is:

```text
C2\tVerification remains missing\tmissing\t-\t-\n
```

`fixture-rev-1` is synthetic fixture data. Trial evidence separately identifies the actual Git commit checked.

## Required workflow observations

The orchestrator retains the complete ticket context and coordinates the [implementation plan](implementation-plan.md). It supplies a fresh implementing agent with only its checkpoint, applicable source excerpts, dependency contract, starting commit, and independently prepared gate demands. Each excerpt retains its source path and digest. Supplied context can be inspected, but it does not restrict filesystem access.

A separate agent derives gate expectations from agreed requirements before implementation. Prepare executable tests one checkpoint at a time. Keep valid existing passing tests; the missing behavior must fail for a behavior-related reason before its implementation passes. A missing executable or compile error is useful baseline evidence but cannot alone establish that a behavior assertion detected a defect.

Implement sequentially. Pin local commits before standards and specification review. Retain both review axes separately. Every correction creates a new revision and renews affected checks and reviews. The user reviews the initial plan and each checkpoint after the required gates pass.

Exercise one grounded correction deliberately if ordinary work does not produce one. On checkpoint 1's disposable local trial branch, change only the displayed tested revision to `HEAD` instead of preserving the input. The independent runner must detect R1's violation. Retain the failed commit, result, correction commit, and renewed checks and reviews. The defect never enters production packages or receives checkpoint approval.

After both human checkpoint reviews, commission a fresh verifier for the combined result against the full ticket and specification. Use the existing [acceptance procedure](../../docs/agents/acceptance.md) only after all work-item criteria have evidence. Human reviews and workflow acceptance retain their actual actors and revisions.

## Evaluation

The trial report assesses these measures from the [original proposal](../workflow-improvements/orchestrator-trial.md#observations-that-determine-success):

- Review effort: each human review shows outcome, reviewed revision, changed files, gate results, and open concerns. Record clarification questions and human-reported review time when available. Request a usability assessment after the last checkpoint. Report time as unknown when unavailable. This trial has no comparison baseline, so it cannot establish reduced review time.
- Context and authority: retain exact dispatch prompts and supplied files or excerpts with digests. Confirm fresh implementation contexts, separate expectation authorship, sequential implementation, and source-scoped feedback. Broader guidance requires explicit adoption.
- Correction integrity: preserve the failure, diagnosis, repair, changed revision, and renewed checks. Advancement requires passing evidence for the corrected revision.
- Combined verification: retain an independent result covering both examples together, preserved order, authored revision strings, and the lack of an acceptance verdict. Checkpoint completion alone leaves whole-ticket acceptance outstanding.

Report each measure as supported, failed, or unmeasured, with its evidence and practical limits. Unknown human timing does not become an invented success claim. Failed context separation, correction integrity, or combined verification leaves the trial's corresponding criterion unmet.
