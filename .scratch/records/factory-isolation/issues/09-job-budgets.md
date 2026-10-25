---
type: WorkItem
id: f2abc34d-c37b-42ab-9726-15550fabea1a
title: Cap a job's agent spend by reserving each run's cap
triage: ready-for-agent
execution: unstarted
---

A job's agent runs cannot together spend more than its budget. Each run's enforceable cap is reserved before the run starts and settled to the cost the run reports.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Executor profiles may declare `runCapUSD`. The assignment carries it, and the agent adapter passes it to Claude Code's `--max-budget-usd`, taking precedence over a cap in the argv.
- Project policy gains `jobBudgetUSD`, where zero means no limit, and admission freezes it. The budget covers build attempts and repairs, judge runs, and dispute test-author proposals for the job.
- In the transaction that assigns a build attempt or starts a judge or test-author run for the job, the controller reserves that run's cap. It refuses when the job has less left than the cap; a judge that cannot be reserved leaves the dispute with a person. A reported cost settles the reservation. A cost above the cap is recorded as observed. An attempt that never reports a cost keeps its full cap reserved until reconciled.
- In a project with a job budget, admission and claim refuse executor profiles without `runCapUSD` and harnesses that cannot enforce a cap, including Codex, and say why.
- A job without enough budget left waits with the reason "budget reached", and the board shows it as a decision. A human project administrator can raise that job's budget, an attributed and recorded action, or stop the job. Neither the autopilot nor an ask policy can raise it.
- Metrics and the board include judge cost and show settled and reserved spend separately.
- Service and HTTP tests with fixture runs and reported costs cover refusal, settlement, recorded overshoot, a retained reservation for an `unknown` attempt, and who may raise a budget.

## Comments

- 2026-10-25: The only spend control today is Claude Code's per-run `--max-budget-usd` in executor argv. Cost is known only from the agent's report at run end, and judge cost is missing from metrics. CodeAF charges after each run and lets in-flight work finish; reserving first keeps this budget a ceiling.
