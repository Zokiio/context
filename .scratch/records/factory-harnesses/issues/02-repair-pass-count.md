---
type: WorkItem
id: fc7ee3c7-6105-459d-a507-fa10fd0a1ef7
title: Count repairs that pass verification
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#repair-brief)

## Context

- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Project metrics report repair attempts, repair attempts whose candidate passed every required check, and their ratio, computed from retained attempts and checks so existing history counts.
- A repair that fails, ends unknown or is cancelled counts as not passed. Merge checks are excluded.
- Insights shows the repair pass count next to the existing repair count, and `projects metrics` returns the same fields.
- A service test with fixture jobs covers a passing repair, a failing repair and a job without repairs.

## Comments

- 2026-10-25: This measurement resolves the [resume decision](../decisions/01-resume-native-sessions.md).
