---
type: WorkItem
id: c7754bb9-7c40-4769-b831-12a826bac4cd
title: Start a fresh coordinator for sibling orchestrations
triage: needs-triage
execution: unstarted
---

# Start a fresh coordinator for sibling orchestrations

Recorded on 2026-10-06 from the batch retrospective of the parallel Axpilot #266 and #275 runs. The [workflow adaptation](../../../workflow-adaptation/spec.md) did not adopt parallel implementers on one ticket, and it records the user's manual parallel runs as an observation. No Waymark guidance describes the session that coordinates them.

## Evidence

One session launched the #266 and #275 orchestrations, each for its own work item in its own session. It then mostly relayed the single screen slot between them. It still held the #272 and #274 work items, so its peak context was 754K.

- From launch to the end of the batch, it sent 52 messages at 38.3M input tokens, about 0.74M each. 28 text-only acknowledgement turns cost 20.7M, 54% of the total.
- Relays queued behind a busy sibling. A screen go-ahead sent at 23:15Z reached #266 31 minutes later, after #266 had already collected the evidence headlessly. #275 waited 67.7 minutes for the screen slot while #266 ran its demo.
- The siblings themselves ran without conflict. #275 touched none of #266's files, and both merged about 8 hours after launch.

## Scope

Add guidance for a session that coordinates sibling orchestrations of separate work items. Start it fresh, and keep implementation, review, and close-out work out of it. Each sibling still follows the orchestrate skill with one active writer. Implementation claims, routing, and parallel writers on one work item stay outside scope.

Triage decides whether this guidance belongs in the installed orchestrate skill or the initialization guide, or waits for another parallel batch.

## Acceptance criteria

- P1: Waymark guidance for coordinating sibling orchestrations says to start the coordinating session fresh and to keep implementation, review, and close-out work out of it.
- P2: The guidance keeps each sibling under the orchestrate skill with one active writer, and adds no implementation claims or routing.
- P3: The canonical templates and generated development copies agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Orchestrate template](../../../../internal/initialization/templates/orchestrate.md)
- [Initializing projects](../../../../docs/initializing-projects.md)
- [Orchestration design](../../../workflow-improvements/orchestration-design.md)
- [Domain glossary](../../../../CONTEXT.md)
