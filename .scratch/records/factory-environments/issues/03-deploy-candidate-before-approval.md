---
type: WorkItem
id: 1a492068-ceef-452a-ae96-2d946297c28d
title: Deploy a candidate to a leased development environment before approval
triage: needs-info
execution: unstarted
---

In the vision's Wi-Fi scenario, a change runs in a development environment before teammate review and merge. Here, the factory deploys the job's candidate to an environment it holds exclusively. It confirms the environment runs exactly that candidate and checks it there. The results join the evidence the person approves. Project policy decides, for each environment, whether the deploy is preauthorized or needs a person.

This waits for a pilot with a development environment that accepts unmerged builds and needs results before review. Gups cannot use it, because gups-dev runs only signed commits from `main`.

## Spec

- [Factory environments](../../../factory-environments/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Observe merged work in a registered development environment](01-observe-merged-change.md)

## Blocked by decisions

None.

## Acceptance criteria

- An environment can have a deploy command. Project policy can mark an environment for candidate deploys and set its authorization to `preauthorized` or `ask`, with `ask` as the default.
- After its local checks pass, a job that needs a candidate-deploy environment acquires that environment's exclusive lease. Other jobs wait, and the lease survives a restart.
- Under `ask`, the card shows a deploy decision to a person with the coordinate capability, and the autopilot never takes it. Under `preauthorized`, the agreed plan covers the deploy, and the autopilot proceeds.
- The worker deploys the candidate. The revision command must report exactly that candidate before the environment checks and again after them.
- A deploy failure or a revision mismatch records `unavailable` and does not use a repair. A failed environment check fails verification like any required check.
- Environment observations join the acceptance evidence with their environment and observed revision. Merge rechecks do not deploy.
- An uncertain deploy keeps the lease until a person reconciles it. The controller records which job's candidate each environment last received, and the board shows it.
- Fixture tests cover lease contention, both authorization settings, a revision mismatch, an uncertain deploy, and a restart while the lease is held.

## Comments

- 2026-10-25: The user chose per-environment deploy authorization, with `ask` as the default, and an autopilot that never confirms an `ask` deploy.
