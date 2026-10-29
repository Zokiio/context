---
type: WorkItem
id: 216f381f-fc27-4bd8-a3cb-3424e1dbd5f0
title: Review each verified candidate with an independent agent
triage: ready-for-agent
execution: unstarted
---

The controller stage, its records and its failure rules. A configured reviewer command reads each candidate that passed its checks, and its result gates the autopilot's delegated approval. Until ticket 04, a blocking review goes to the person.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)
- [Software factory research](../../../../docs/research/software-factory-2026-10-22.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Admission records whether a reviewer was configured, and the job keeps that for its lifetime. Jobs admitted without one behave as today.
- When the last required check passes, the controller records one agent review for that candidate and requirement fingerprint in the same transaction. It then runs the configured command once per axis, `standards` and `spec`, in parallel. Each run gets the input the spec lists, in a clean worktree at the candidate, with a 10-minute limit that never passes the job deadline. The builder's transcript, final message and progress notes, the other axis's result and earlier candidates' reviews are excluded.
- The controller validates each axis's output against the spec's contract and records the outcome as `clear`, `blocking` or `failed`. Every failure the spec lists records `failed` with its reason, log and reported cost, and never counts as a pass. A restart marks running reviews failed and starts nothing. Stopping the job cancels its reviewers. A late result is kept and starts nothing.
- Agent reviews live in their own records, attributed to harness, model and session, never to a user. They never enter `reviews`, the five human finding kinds or the human finding metrics. The service never refuses a person's approval because of an agent review. Each review event names the agent review current at that moment.
- Under `when_untested` or `never`, delegated approval requires a `clear` agent review of the exact candidate whenever the job was admitted with a reviewer. A running, failed, blocking or missing review asks the person.
- The board shows "Reviewing the change" in Build while the review runs. It shows the clear, blocking and failed states on the Approve card as the spec describes, and "Approving for you" only on a clear review. Run adds an Agent review step with each axis's verdict, findings, model, turns, time and cost. History keeps every candidate's review. Card and job cost include the reviewer's reported cost.
- `serve` and `solo` accept `--review-command-file`. `jobs show` and `GET /v1/jobs/{id}` include the job's agent reviews.
- `CONTEXT.md` defines agent review and agent finding as distinct from a person's review. The operator guide, the architecture document and the HTTP document describe the stage.
- Deterministic subprocess reviewers exercise clear, blocking and advisory-only outcomes and every failure case the spec lists, the delegated-approval gate for each outcome, a person approving over blocking findings, and the board and Run states.
