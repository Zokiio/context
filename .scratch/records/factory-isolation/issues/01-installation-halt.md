---
type: WorkItem
id: 72d5011a-2e79-4e4e-97b6-6a3e59c72e27
title: Halt all factory execution from one switch
triage: ready-for-agent
execution: unstarted
---

An installation administrator can stop every run and keep new ones from starting. Stops the factory cannot confirm stay visible and held instead of being reported as stopped.

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

- A human installation administrator can engage the halt with a reason through `ctx factory halt`, `PUT /v1/halt` and a browser control. Other users and worker principals are refused. Engaging and releasing are idempotent by request ID and record the actor, reason and time.
- While halted, worker claims return `no_work` with the reason `halted`, and the autopilot takes no step. The service refuses to start refinements, expectation suites, judgments and merge checks. An acceptance publication already in progress may finish.
- Engaging the halt stops each active job exactly as a person's Stop does, including the work-item hold, and cancels running in-service agents. Each attempt ends with its actually observed outcome. An unconfirmed stop stays `cancellation_unknown` with its claim held.
- The halt survives restart. `serve` and `solo` start halted, and the autopilot stays idle.
- Release is a separate explicit action. It is allowed while stops are still unconfirmed, and its response lists those runs. It never resumes or re-dispatches work, and stopped work items keep their holds until a person queues them.
- The board shows a halt banner with the reason, actor and time, and shows "Stop not confirmed" on each run whose stop is uncertain. The operator guide documents halt and release.
- Service, HTTP and browser tests with fixture workers cover engaging, refusal while halted, restart, release with unconfirmed stops, and authorization.

## Comments

- 2026-10-25: The user approved the budget and halt model in the isolation plan. No halt control exists in `dev` at `afe9989`; `--no-pilot` applies only at startup.
