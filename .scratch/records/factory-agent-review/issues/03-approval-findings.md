---
type: WorkItem
id: be1fa9bd-b613-4c99-b9de-e4e2aa17e7af
title: Show agent findings where the person approves
triage: ready-for-agent
execution: unstarted
---

The browser's approval view, in the drawer and on the page. The person's decision stays one decision.

## Spec

- [Agent review plan](../../../factory-agent-review/spec.md)

## Context

- [Factory HTTP and browser access](../../../../docs/factory-http.md)

## Blocked by

- [Review each verified candidate with an independent agent](01-agent-review-stage.md)

## Blocked by decisions

None.

## Acceptance criteria

- The approval view lists the current agent review's findings in their own section, attributed to the reviewer's model. Blocking findings come first, each with its axis, location and source. A clear review folds the section under "Nothing blocking". A failed review gives the reason and tells the person to read the change.
- "Check this yourself" still lists every criterion no locked test checks, whatever the agent review found.
- The person's finding chips ask what the checks and the agent reviewer missed. Their values and the five kinds are unchanged.
- When blocking findings remain, the approve button reads "Approve anyway". No other step or confirmation is added.
- Browser regression checks cover clear, advisory-only, blocking and failed reviews, and the person's input survives polling.
