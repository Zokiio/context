---
type: WorkItem
id: 27be582d-0f90-4c66-b341-7ae8195223b8
title: Operate shared projects and jobs in the web workspace
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- Provide a usable authenticated workspace with project navigation, work items, job submission, queue and job detail.
- Show requirements, workflow state, attempts, candidate revision, events, checks, reviews, and remaining action from real service responses.
- Expose permitted cancellation, repair, verification, review and acceptance operations, while the service independently enforces authorization.
- Render empty, loading, denied, failed, stale and uncertain outcomes without misleading success states or unsafe HTML.
- An integrated browser check exercises a complete fixture-backed workflow and at least one denied or incomplete path against the same service used by the CLI.

## Spec

- [Factory platform](../../../factory-platform/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)

## Blocked by

None

## Blocked by decisions

None

## Comments

2026-10-22: Selected by the user as part of the complete factory platform, before target-project pilot work. Parallel implementation uses the same frozen interface contract and separate owned paths. Whole-platform acceptance remains a separate gate.
