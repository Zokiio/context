---
type: WorkItem
id: f2450eb9-775f-48f7-a86d-23af5339e6af
title: Poll bound trackers and import what pickup triggers match
triage: ready-for-agent
execution: unstarted
---

Refresh is manual after tickets 01 and 02. This ticket makes the service poll each binding and adds pickup triggers, so only matching issues enter the project. Triggers here only import. Queueing on a person's behalf comes in ticket 05.

## Spec

- [Factory intake](../../../factory-intake/spec.md)

## Context

- [Tracker intake decision](../../../../docs/adr/0006-tracker-intake-ownership-and-delegated-pickup.md)

## Blocked by

- [Import a repository's open GitHub issues as draft work items](01-import-github-issues.md)
- [Keep agreed requirements when the tracker changes](03-keep-agreed-requirements.md)

## Blocked by decisions

None.

## Acceptance criteria

- A project administrator saves pickup triggers on a binding. Each trigger has a name, label and assignee conditions combined as all or any, and the action `import`.
- A refresh imports only issues that match at least one trigger. Each imported item records which triggers matched, and the Task page shows them in folded details. A binding created before triggers existed keeps importing every open issue through one import trigger with no conditions.
- The service polls every binding at an interval set by `--intake-interval`, five minutes by default, and `0` turns polling off. "Refresh now" still works. Polls for one binding never overlap.
- The first poll reads every open issue in scope. Later polls ask for issues updated since the last successful poll, with one minute of overlap, so closures and label changes arrive without reading every issue. A restarted service resumes from the last successful poll.
- A rate-limit response backs off until the tracker's reset or retry time. A failed poll changes no record and keeps the last successful time.
- A poll imports at most 100 new items and reports how many more matched.
- When a binding has not polled successfully for two intervals, the board's top bar shows one line, such as "GitHub not reached since 10:02". The Task page shows each item's snapshot age.
- Tests drive polling with a controllable clock against the fake GitHub and Jira servers. They cover matching with all and any, the no-condition migration, incremental reads, rate limiting, the import cap and restart.
