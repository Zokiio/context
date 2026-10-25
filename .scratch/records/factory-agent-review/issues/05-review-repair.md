---
type: WorkItem
id: 311220e7-59a4-4ff5-89f0-e5af2cef5248
title: Send blocking agent findings to a bounded repair
triage: ready-for-agent
execution: unstarted
---

On 2026-10-25 the user chose to have blocking findings start a repair without the autopilot, the way the judge routes an implementation defect.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Tell a repair attempt why its candidate was sent back](01-repair-feedback.md)
- [Review each verified candidate with an independent agent](02-agent-review-stage.md)

## Blocked by decisions

None.

## Acceptance criteria

- A blocking agent review of the job's current candidate starts a repair in the transaction that records it, acting as the job's coordinator. It does so only when repairs are left, the deadline hasn't passed, no execution is uncertain, the requirements are unchanged and the coordinator still has `coordinate`. It does not wait for the autopilot.
- The repair counts against the admitted repair limit that failed checks also use. Its event names the agent review that started it.
- The repair's instructions carry the blocking findings, with their axis, location and source, attributed to the agent reviewer, through the path from ticket 01. Advisory findings are not sent.
- The new candidate goes through the required checks and a fresh agent review.
- When no repair is possible, the person gets the Approve decision with the blocking findings first. No outcome accepts the change with open blocking findings.
- A stop, a stale job version or a late result starts no repair.
- The board shows the existing "Fixing" card with the number of problems, and Run's Agent review step names the fix it started.
- Deterministic checks cover blocking then repaired then clear, blocking until the limit and then the person, a failed check and a blocking review sharing the limit, a coordinator who lost `coordinate`, a stop during the review, and a late result.
