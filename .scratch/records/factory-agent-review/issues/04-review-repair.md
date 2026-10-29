---
type: WorkItem
id: 311220e7-59a4-4ff5-89f0-e5af2cef5248
title: Send blocking agent findings to a bounded repair
triage: ready-for-agent
execution: unstarted
---

On 2026-10-25 the user chose to have blocking findings start a repair without the autopilot, the way the judge routes an implementation defect.

The findings reach the builder through the repair brief that the agent harnesses plan (`.scratch/factory-harnesses/spec.md`) owns. That brief tells a repair attempt about failed checks, protected-path rejections, the previous agent's final message and a person's "send back for a fix" note. This ticket adds one section to it and no second channel.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Review each verified candidate with an independent agent](01-agent-review-stage.md)
- [Tell a repair why the previous attempt failed](../../factory-harnesses/issues/01-repair-brief.md)

## Blocked by decisions

None.

## Acceptance criteria

- A blocking agent review of the job's current candidate starts a repair in the transaction that records it, acting as the job's coordinator. It does so only when repairs are left, the deadline hasn't passed, no execution is uncertain, the requirements are unchanged and the coordinator still has `coordinate`. It does not wait for the autopilot.
- The repair counts against the admitted repair limit that failed checks also use. Its event names the agent review that started it.
- The repair brief gains an agent review section with the blocking findings, their axis, location and source, attributed to the agent reviewer. Advisory findings are not sent.
- The new candidate goes through the required checks and a fresh agent review.
- When no repair is possible, the person gets the Approve decision with the blocking findings first. No outcome accepts the change with open blocking findings.
- A stop, a stale job version or a late result starts no repair.
- The board shows the existing "Fixing" card with the number of problems, and Run's Agent review step names the fix it started.
- Deterministic checks cover blocking then repaired then clear, blocking until the limit and then the person, a failed check and a blocking review sharing the limit, a coordinator who lost `coordinate`, a stop during the review, and a late result.

## Comments

- 2026-10-25: Triage stays `needs-triage` until the repair-brief ticket from the agent harnesses plan exists on the same branch. Then link it under Blocked by and set `ready-for-agent`. That plan was not yet written when this ticket was split, so a link here would not resolve.
- 2026-10-25: Both plans are now on `feat/factory-gap-plans`. The repair-brief ticket is linked under Blocked by, and triage is `ready-for-agent`.
