# First orchestrator trial

Status: stable

Notes: isolated fixture selected and initial plan approved on 2026-10-01. This document made Q21 in the [workflow design](orchestration-design.md) concrete. The [scope decision](../records/orchestration-trial/decisions/01-trial-scope.md) records the user's authorization to start the fixture trial. It adds no product feature.

2026-10-02: This proposal is retained as trial history. The [trial conclusions](orchestration-conclusions.md) record the completed and merged fixture and Axpilot #241 trials. The intended follow-up below has been delivered.

The [implementation plan](../orchestration-trial/implementation-plan.md), [fixture specification](../orchestration-trial/spec.md), and [orchestrator design](../orchestration-trial/orchestrator-design.md) define the approved run. An Axpilot ticket is the intended follow-up after this fixture trial.

## Work to exercise

Create an orchestrator skill for the agreed workflow and exercise it on one small local fixture outside production packages. The fixture is a standalone program that reports explicitly authored criterion-to-evidence mappings. It makes no acceptance decisions and requires no remote service, account, deployment, or device.

Use two checkpoints:

1. Display one authored criterion, its observation, evidence source, and tested revision.
2. Handle a criterion with no observation. Show that verification is missing, preserve the criterion, and avoid presenting it as satisfied.

An independent final check covers a combined fixture containing both cases. The fixture is temporary evaluation work, not a commitment to implement the proposed verification view in `ctx`.

## Workflow to exercise

Retain a spec, a plan linking separate checkpoint files, progress observations, and verification evidence in project files. Each document has one authoritative home. Use the existing document conventions for the trial; a new product record schema is outside its scope.

The orchestrator receives the complete task context. A separate agent derives the gate expectations and tests from the agreed requirements before implementation. Fresh implementing agents receive only their checkpoint and its relevant source context, starting revision, and gate demands. Retain the supplied source identities so the trial can inspect what each agent received.

Implement one checkpoint at a time. Review explicit local code revisions, retain independent standards and specification reviews, and rerun affected checks after corrections. The user reviews the plan and each checkpoint after its gates pass. The orchestrator then arranges independent verification of the combined result before acceptance of the fixture task.

Existing skills may need focused changes for checkpoint delegation and exact review targets. The [workflow design](orchestration-design.md#existing-constraints-and-observed-gaps) records those gaps. CLI behavior, managed claims, concurrent implementation, and enforcement hooks are outside this first trial.

## Observations that determine success

- Review effort: each checkpoint's review summary lets the user identify its intended outcome, reviewed revision, gate results, and unresolved concerns. Record review time and clarification questions. Do not claim an improvement without a comparison baseline.
- Context and authority: inspect the supplied context to confirm that an implementer did not inherit the full plan or orchestrator conversation. Check that gate expectations came from a separate agent and that feedback did not acquire broader authority without adoption. This checks supplied context, not filesystem access restrictions.
- Correction integrity: exercise a failed check or grounded review finding, deliberately if needed in this controlled fixture. Retain the failed result, the correction revision, and the renewed affected checks. Failure must not be recorded as a pass to advance the checkpoint.
- Combined verification: an independent reviewer evaluates both fixture cases against the complete requirements. Passing checkpoint gates alone must not produce ticket acceptance.

The trial report should retain both useful results and unresolved limitations. Its outcome informs whether to use the skill on an approved real ticket and whether further product support is justified.
