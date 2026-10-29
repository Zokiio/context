---
type: WorkItem
id: 2f4b63a9-3347-403b-b745-6e11c480297f
title: Observe merged work in a registered development environment
triage: needs-info
execution: unstarted
---

A person can see whether merged factory work is running in a development environment and whether that environment's checks passed there. The project's own pipeline deploys. The factory reads the revision the environment reports and records what it observed, separately from local checks.

This waits for the Gups pilot. The factory must have merged Gups work that reached `main`, and gups-dev must be able to report its running revision.

## Spec

- [Factory environments](../../../factory-environments/spec.md)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- An installation administrator can register and list environments with a key, a display name, a revision command, an observation window, and a fixture flag. Registration refuses a key that an existing check profile uses as its environment label.
- Project policy lists environment checks for each registered environment. Saving the policy refuses a required local check whose profile names a registered environment. It also refuses an environment check whose profile names a different environment.
- Recording a merge in any mode queues one environment attempt for each environment in the current policy. Only a worker claiming the environment stage receives it, and the job's implementer is refused.
- When the reported revision contains the job's candidate, each environment check runs in a clean worktree at that revision. The revision is read again afterwards, and a change records those checks as unavailable.
- When the reported revision does not contain the candidate, or the repository does not have it, the attempt retries until the observation window closes. It then records the environment as not deployed, naming the last revision it saw.
- Each observation retains the environment, the candidate and merge revisions, the observed revision, the check, the status, the observer, the attempt, and the time. Observations never change the job's status, its acceptance, or the trusted-merge count.
- The job's API and CLI output include the observations. The Done card shows one fact per environment: passed, failed, not deployed, or waiting. Run shows the observations folded under each environment.
- A restart marks interrupted environment attempts unknown, and an authorized person can queue the observation again.
- Fixture tests cover the cases listed in the spec's verification section. The operator guide explains registering an environment and running an environment worker, and `CONTEXT.md` gains the spec's three terms.

## Comments

- 2026-10-25: Planned from the environment gap analysis. The user chose to observe after merge first and to keep this ticket gated on the Gups pilot.
