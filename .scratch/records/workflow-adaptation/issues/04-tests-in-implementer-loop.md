---
type: WorkItem
id: 9c9dd2cc-6bc4-4bcc-acc0-dc976f01c08a
title: Write checkpoint tests in the implementer's loop
triage: ready-for-agent
execution: in-progress
---

# Write checkpoint tests in the implementer's loop

Proposed on 2026-10-04 in the workflow adaptation specification. In the Axpilot runs, separate expectation authors produced 4 to 9 times more test lines than production lines.

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
