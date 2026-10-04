---
type: Decision
id: acebfefb-2f50-4a74-9c0d-546233e4497f
title: Choose who writes checkpoint tests
decisionState: resolved
---

# Choose who writes checkpoint tests

## Resolution

2026-10-04: The user chose to move checkpoint tests into the implementer's loop. [Ticket 04](../issues/04-tests-in-implementer-loop.md) implements it and records in the orchestration design that this decision replaces choice 16.

## Basis

Orchestration design choice 16 gives test preparation to an expectation author separate from implementation. In the 2026-10-04 Axpilot runs, that produced 4 to 9 times more test lines than production lines, as the [specification](../../../workflow-adaptation/spec.md) records. Both orchestrators also told their authors to write new files only, which multiplied test harnesses.

## Options

- **Keep the separate author and limit its output.** The verification brief asks for the fewest cases that distinguish each requirement and its failure cases, directs authors to extend existing test files and helpers, and requires a reason for any new file or harness. The Standards baseline treats duplicated test setup as Duplicated Code. This keeps choice 16 and is a two-paragraph template change.
- **Move tests into the implementer's loop.** The implementing agent works test-first in vertical slices at the seams the checkpoint file names. The Specification reviewer checks test coverage per requirement and the independence of expected values. This replaces choice 16. [Ticket 04](../issues/04-tests-in-implementer-loop.md) implements it.

## Recommendation

Move tests into the implementer's loop. The defects found in both runs came from reviewers, the combined-result verifier, and the user's demo, while the separate authors mainly added volume. Upstream `tdd` names the separate-author pattern, horizontal slicing, as an anti-pattern because bulk tests verify imagined behavior.

If the user keeps the separate author, cancel ticket 04 and record the first option as a new ticket.
