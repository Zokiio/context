---
type: WorkItem
id: ed552f35-199f-4840-8e82-014b077cb4c2
title: Show requirements that only a development environment can establish
triage: needs-info
execution: unstarted
---

At approval, an environment has not yet run the change. A requirement only that environment can establish, such as configuration reaching a device, must not count as tested because a local test claims to cover it. The test author names the environment for such a requirement. The approval card and the ask policy then treat it as unchecked.

Today, a requirement without a locked test already makes `when_untested` ask a person. This ticket closes the remaining gap, where the author claims a local test for it. It waits until a Gups work item has a criterion only gups-dev can establish, or until a test author is seen claiming local coverage for one.

## Spec

- [Factory environments](../../../factory-environments/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Observe merged work in a registered development environment](01-observe-merged-change.md)

## Blocked by decisions

None.

## Acceptance criteria

- The test author's output can name a registered environment for a criterion, and the service rejects names it has not registered. The author's instructions say to name one only when no local test can establish the criterion, and never to claim a local test for that criterion.
- A criterion with an environment counts as unchecked at approval. The card says which environment checks it after merge, and `when_untested` does not approve on the person's behalf.
- After merge, the Done card links each such criterion to its environment's observation status.
- Expectations without environment names behave exactly as before.
- Regression checks cover the author's input and output contract, the board derivation, and the autopilot's approval decision.

## Comments

- 2026-10-25: Planned with the environment spec. It is held until a trial shows the need.
