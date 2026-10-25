---
type: WorkItem
id: 9efd6195-4b3b-45e0-aee7-b9f725dc08d0
title: Measure what the agent review catches
triage: ready-for-agent
execution: unstarted
---

No trial has shown yet that the reviewer catches what the checks miss. These numbers decide whether it stays on, and whether a second harness is worth adding.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

None.

## Blocked by

- [Send blocking agent findings to a bounded repair](05-review-repair.md)

## Blocked by decisions

None.

## Acceptance criteria

- Project metrics and Insights report agent reviews by outcome (`clear`, `blocking`, `failed`), blocking findings by axis, repairs the reviewer started, and the reviewer's cost within agent cost.
- They report the person's findings, by kind, on changes whose agent review was clear.
- A failed review counts as failed, never as clear.
- The existing finding metrics still count only the person's findings.
- Checks over a fixture history cover each count, including a failed review and a delegated approval.
