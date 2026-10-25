---
type: WorkItem
id: 8394db44-71a5-4878-b163-dbe5435acfae
title: Count GitHub checks on the pull request head as evidence
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- An installation administrator registers a host check profile with its key, the reporting app, the check run name or commit status context, its definition paths, and a wait that defaults to 30 minutes. A project requires it by key, and admission freezes it.
- Project policy refuses a required host check whose definition paths its protected paths do not cover.
- The service observes required host checks on the exact head commit under a dedicated worker-kind observer identity. Each observation records the head commit, requirement fingerprint, observer, check run ID, app, conclusion, details URL, completion time, and an artifact with the host's JSON and its digest. Acceptance publishes this evidence.
- Conclusions map as the spec's table says. Only `success` passes. A check with the same name from another app is ignored and noted. A check still pending or absent when its wait expires is `unavailable`.
- The job becomes `ready_for_review` only when every required local and host check has passed on the head commit. Results for any other commit never count.
- A failed host check sends the job to `verification_failed`. The repair brief names the check, its summary, and its details URL.
- A required check that is `skipped` or `unavailable`, local or host, leaves the job not verified instead of failed. The autopilot does not repair it. The card says "Not verified" with the check and what was reported. A person can check again, which starts a new observation, ask for a fix, or cancel.
- Offline tests cover each conclusion, the wrong app, a missing check, results for an older head, an unreachable host, the protected-path refusal, and the autopilot leaving an unverified job alone. The operator guide documents host check profiles and the mapping.

## Spec

- [Host checks](../../../factory-code-host/spec.md#host-checks)

## Context

- [Code host ADR draft](../../../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md)
- [Software factory research](../../../../docs/research/software-factory-2026-10-22.md)

## Blocked by

- [Tell a repair attempt why it is repairing](01-repair-brief.md)
- [Publish a checked candidate as a draft pull request](02-draft-pull-request.md)

## Blocked by decisions

None

## Comments

- 2026-10-25: The not-verified rule changes local checks too. Today a local check that reports `skipped` or `unavailable` fails verification, and the autopilot repairs code that did not fail.
