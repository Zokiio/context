# Install portable explicitly invoked checkpoint orchestration

Status: stable

The user authorized this scope on 2026-10-02. [The WorkItem](../../records/workflow-maintenance/issues/02-portable-checkpoint-orchestration.md) owns scope and acceptance criteria.

## Sequence

1. Prepare independent gate expectations before implementation.
2. Implement [checkpoint 1](checkpoint-1.md) in a fresh scoped context.
3. Commit the candidate, run an independent check, and review Standards and Specification at the full target revision. Correct grounded findings and renew affected checks.
4. Present the identified revision for human review. This outcome has no dependent implementation checkpoint. Independent work may proceed while review remains pending.
5. A fresh independent verifier assesses the complete outcome. Any recorded whole-ticket acceptance must retain actual human approval attribution and local evidence limitations.

Keep dispatches, progress, reviews, command output, and generated Acceptance records ignored. Publication ends with a separate reviewable PR, leaving merge to the user.
