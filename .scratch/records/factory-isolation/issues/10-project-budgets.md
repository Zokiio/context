---
type: WorkItem
id: 93aea191-3a03-42f6-b4ea-e2c32a913114
title: Cap a project's monthly agent spend
triage: ready-for-agent
execution: unstarted
---

All agent runs in a project, including refinements and expectation suites that run before any job exists, reserve against a monthly project budget.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Cap a job's agent spend by reserving each run's cap](09-job-budgets.md)

## Blocked by decisions

None.

## Acceptance criteria

- Project policy gains `monthlyBudgetUSD` per calendar month in UTC, where zero means no limit. A human project administrator sets it through `PUT /v1/projects/{id}/budget` and the CLI, and the change is recorded.
- Every agent run in the project reserves its cap against the month in the same transaction as any job reservation. That covers build attempts, judge runs, test-author proposals, refinements and expectation suites. Settlement follows the job-budget rules.
- When the month's budget is used up, no new agent run starts in the project. Running work finishes within its reservations. The board shows the month's settled spend, reserved spend and limit, with a banner while the budget is used up. Neither the autopilot nor an ask policy can raise the budget.
- Tests with an injected clock cover refusal, a month rollover, refinements before any job, and authorization.

## Comments

- 2026-10-25: The user chose job and project budgets plus an installation halt, with no installation-wide dollar budget.
