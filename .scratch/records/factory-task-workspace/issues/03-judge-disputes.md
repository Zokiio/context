---
type: WorkItem
id: 3b90e1e3-cc70-46c9-a116-8cc36d59382a
title: Judge implementation disputes and show the next responsible agent
triage: ready-for-agent
execution: completed
---

## Spec

- [Approved judge and handoff behavior](../../../factory-judge/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- The implementer can submit a structured, correlated dispute and stop without accepting or committing a candidate. Authorized people can raise a dispute on a terminal failed or reviewable run; ordinary failure handling remains available.
- A configured independent read-only judge records its decision, rationale, evidence, and next owner. Implementation defects return to the builder; test defects or missing coverage reach the test author; intent changes, weaker acceptance, unavailable judgment, and repeated disputes reach the person.
- Test revisions retain the old run and frozen tests, obey the configured approval policy, create a linked replacement run, and require fresh verification and approval. Human return to planning supports changed intent without automatic dispatch.
- Authorization, expected versions, durable idempotency, cancellation holds, deadlines, bounded cycles, and restart recovery prevent duplicate or stale handoffs. Unknown execution cannot be replaced.
- Run visibly shows the judge's decision and next responsible agent, including pending human approval, active routing, and original/replacement links. History retains the earlier decisions and evidence. Polling and stale responses preserve unsaved input.
- CLI and HTTP expose the same operations, and deterministic service/worker/browser checks exercise the complete handoffs and failure cases. The change is added to PR #55.

## Acceptance

- [Verified judge handoffs](../acceptances/03-judge-disputes.md)

## Comments

- 2026-10-24: The user approved implementation of the judge design and specifically requested the decision and next agent in Run, in the same PR.

- 2026-10-24: Implemented in the combined PR #55. Go 1.25 full tests, targeted race checks, vet, controlled browser regressions, and the real service/browser judge-to-test-author-to-replacement flow passed at `7bfae39227a2ddbf6cd463adbaf32127dd01cc04`. Native model providers were not invoked during verification.
