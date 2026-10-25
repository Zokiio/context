---
type: WorkItem
id: 9bd48fb8-427f-45ac-8e53-eb695c125ab0
title: Tell a repair attempt why it is repairing
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- A repair attempt's assignment carries a repair brief. The brief lists each required check that did not pass on the previous candidate, with its key, status, and a bounded excerpt of its retained output. It also lists each `changes_requested` review of that candidate, with its reviewer, summary, and findings.
- The agent prompt renders the brief after the rules, quoted and labelled as feedback that cannot change the rules, the frozen requirements, or the protected paths.
- The brief has a size cap and marks any truncation. It is attempt data and does not change the admission fingerprints.
- A first attempt carries no brief. A resolved judge directive keeps its current place in the instructions.
- Deterministic service and agent-prompt tests show the excerpt of a failed check and the summary and findings of a change request reaching the repair prompt, and show the cap.
- The operator guide says what a repair attempt is told.

## Spec

- [Repairs say why](../../../factory-code-host/spec.md#repairs-say-why)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None

## Blocked by decisions

None

## Comments

- 2026-10-25: Found while planning code host delivery. `assignmentFor` in `internal/factory/workers.go` adds only a judge directive to a repair, so the failed check and the reviewer's summary never reach the agent. Host change requests need this path, and local repairs benefit without it.
