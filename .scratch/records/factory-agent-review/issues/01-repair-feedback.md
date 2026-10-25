---
type: WorkItem
id: 86200b0a-aa85-4540-885c-20e01af9b0e0
title: Tell a repair attempt why its candidate was sent back
triage: ready-for-agent
execution: unstarted
---

When a person clicks "Send back for a fix", `Review` stores their note as a `changes_requested` review and the browser starts a repair. `assignmentFor` in `internal/factory/workers.go` adds only a resolved judge decision to the repair builder's instructions. The builder starts from the rejected candidate without the person's note.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- A repair attempt that starts from a candidate a person reviewed with `changes_requested` receives that review's summary and findings in its instructions, attributed to the person, together with the reviewed candidate revision.
- Only the review of the candidate the repair starts from is included. Earlier reviews and approvals are not.
- The note is bounded and framed as feedback to address within the frozen requirements. It changes neither the admission, the protected paths nor the requirement fingerprint.
- Repairs after a failed check and repairs the judge routes keep their current instructions.
- The path that carries the note accepts other attributed feedback, so ticket 05 can send the agent reviewer's findings through it.
- Regression checks cover the service's assignment for a person-requested repair and for a failed-check repair, and the prompt the agent adapter renders from each.
