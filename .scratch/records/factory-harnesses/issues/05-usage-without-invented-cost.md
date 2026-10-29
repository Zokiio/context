---
type: WorkItem
id: d8d8eda6-b251-49a2-9b14-dc068741001f
title: Report agent usage without inventing a cost
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#harness-contract)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md#metrics-are-measured-not-estimated)

## Blocked by

- [Check the native adapters against live providers on demand](03-live-adapter-check.md)

## Blocked by decisions

None.

## Acceptance criteria

- An agent result and a role run record usage in the harness's own units: dollars for Claude, input, cached and output tokens for Codex, and AI credits for Copilot. A dollar cost is present only when the harness reported one.
- The Codex adapter reads token usage from its `turn.completed` events, using the committed live transcript.
- Metrics sum only reported dollar costs and count runs whose cost is unknown. Insights shows how many runs reported a cost, for example "cost known for 4 of 6 runs", and never shows an unknown cost as $0.
- Older records without the new fields keep their reported costs.
- Unit tests cover a Claude, a Codex and a no-usage result, and a metrics test covers mixed known and unknown costs.
