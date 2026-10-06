---
type: WorkItem
id: 9c9dd2cc-6bc4-4bcc-acc0-dc976f01c08a
title: Write checkpoint tests in the implementer's loop
triage: ready-for-agent
execution: completed
---

# Write checkpoint tests in the implementer's loop

Proposed on 2026-10-04 in the workflow adaptation specification. In the Axpilot runs, separate expectation authors produced 4 to 9 times more test lines than production lines. [PR #32](https://github.com/Zokiio/context/pull/32) merged after human review on 2026-10-04.

## Scope

Replace orchestrate's separate expectation author with test-first vertical slices by the implementing agent, at seams named in the checkpoint file. Move the independence check to the Specification review. Runner, Standards review, combined-result verification, and human gates are unchanged.

## Acceptance criteria

- T1: The orchestrate plan names each checkpoint's test seams, and the user reviews them with the plan under the project's review policy.
- T2: The implementing agent works test-first in vertical slices at those seams: one failing test, the least code that passes it, then the next. It extends existing test files and helpers and gives a reason for any new harness.
- T3: The Specification review checks that every requirement has a test at an agreed seam and that expected values come from the specification, known literals, or worked examples rather than restating the implementation.
- T4: The separate expectation-author step is removed from the skill and the verification brief. The orchestration design records that the linked decision replaces choice 16.
- T5: The canonical templates and generated development copies agree, and `go test ./internal/initialization ./internal/cli` passes.

## Spec

- [Workflow adaptation](../../../workflow-adaptation/spec.md)

## Blocked by

None

## Blocked by decisions

- [Choose who writes checkpoint tests](../decisions/01-checkpoint-test-authorship.md)

## Context

- [Choose who writes checkpoint tests](../decisions/01-checkpoint-test-authorship.md)
- [Orchestration design](../../../workflow-improvements/orchestration-design.md)
- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-10-04: Completed after human review and merged publication in [PR #32](https://github.com/Zokiio/context/pull/32), covering T1 through T5. Independent Standards and Specification reviews, their follow-up renewals, and CI passed before merge. The closeout acceptance uses retained verification at the merged head. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

2026-10-06: Later Axpilot runs supply the measures the specification named. #272, #274, #275, and #266 are based on Axpilot PR #276, which committed Waymark 541e468. Their test-to-production ratios for added lines were 5.6:1, 5.6:1, 2.6:1, and 2.7:1, against 9:1 for #254 and 4.4:1 for #264. In the #266 and #275 batch, the native demo was the only gate that caught two defects. In #266, tests at the component seam passed an unavailable provider straight to a "not available" card that the app's own provider selection could never show. The #275 demo exposed a stale generated binding that the #274 pull request had merged after its runner, both review axes, the combined verifier, and CI passed. Axpilot answered with its own guidance on reaching user-visible states through the real selection path, and with a CI follow-up for generated bindings. Observation only; no Waymark change is recorded.

## Acceptance

[Closeout acceptance](../acceptances/04-tests-in-implementer-loop-closeout-20261004.md)
