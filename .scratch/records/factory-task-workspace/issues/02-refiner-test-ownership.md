---
type: WorkItem
id: 9416c988-093c-4414-8c75-81db68c25908
title: Give requirement refinement the configured test ownership
triage: ready-for-agent
execution: in-progress
---

The Count word frequencies trial proposed a new protected test file and asked whether the implementer could create it. The factory already had a separate test author. The refiner received protected paths but no test-author availability, and its prompt did not explicitly forbid creating new protected files. The person's answer corrected the accepted criteria before implementation.

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Refinement inputs identify whether this project can use the configured test author. Availability requires both an author command and protected paths, independently of whether tests are mandatory.
- The refiner's instructions explicitly prohibit the implementer from creating, editing, renaming, or deleting files matching protected paths.
- When a test author is available, refinement assigns new protected acceptance tests to that author and does not ask the person to grant the implementer permission to write them. When unavailable, instructions do not promise that a test author will run.
- Existing task intent, human decisions, and retained run records remain intact. The change does not rewrite or rerun the current accepted requirements.
- Regression checks exercise the service-to-refiner input and the actual model-adapter prompt for available and unavailable configurations.

## Acceptance

## Comments

- 2026-10-24: The user reported that the current run asked the implementer to create tests. Retained refinement history confirmed the question. The subsequent accepted requirements corrected it; the implementation diff contains only strutil/wordfreq.go.
