---
type: WorkItem
id: 73f680ec-6bd1-48b0-8c4b-2481c1530494
title: Choose the refiner, test author and judge per project
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#role-profiles)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- An administrator registers role profiles with a key, a role (`refiner`, `test_author` or `judge`), a display name and an immutable argument array. As with executor profiles, a registered profile never changes.
- Project policy names at most one profile per role. A project that names none uses the installation's configured command for that role, so existing installations behave as before.
- A refinement, test run or judgment keeps the profile it started with. Changing the project's choice never alters a running one.
- Each refinement, test run and judgment records its profile key, harness and model. Run, History and the Agents page show them.
- CLI and HTTP expose registration and per-project selection. The Agents page lets a project administrator pick a profile for each role, and read-only users can see the choice.
- Solo registers its built-in Claude role profiles under stable keys.
- Service, HTTP and browser tests cover selection, the fallback to the installation command, freezing at start and authorization.

## Comments

- 2026-10-25: Joakim chose per-project role profiles and recording over an enforced rule that the judge's harness differ from the builder's. The agent-review plan can add a `reviewer` role.
