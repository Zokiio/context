---
type: WorkItem
id: c4936695-3243-4a70-8671-8ecaf3559042
title: Require sandboxed execution in a project
triage: ready-for-agent
execution: unstarted
---

A project administrator can require that every build, check and expectation suite in the project runs sandboxed, so an unsandboxed profile cannot slip in.

## Spec

- [Isolate factory execution and bound its spend](../../../factory-isolation/spec.md)

## Context

- [Sandboxed runs exchange code as Git bundles](../../../../docs/adr/0009-sandboxed-runs-exchange-git-bundles.md)

## Blocked by

- [Run required checks in a Docker sandbox](02-sandboxed-checks.md)
- [Build candidates in a Docker sandbox](04-sandboxed-builds.md)
- [Write holdout expectations in a sandbox](07-sandboxed-expectations.md)

## Blocked by decisions

None.

## Acceptance criteria

- Project policy gains `requireSandbox`. Only a human project administrator can set it, and setting it is recorded.
- With it set, the service refuses to admit or claim work that uses an unsandboxed executor or check profile, and refuses to start an unsandboxed expectation author. Each refusal names the profile.
- Jobs admitted before the policy changed keep their frozen policy.
- Orientation and the board show that the project requires sandboxes. The operator guide documents the setting.
- Service and HTTP tests cover admission, claim, expectation refusal, frozen jobs and authorization.

## Comments

- 2026-10-25: Sandboxes are opt-in per profile until every execution path supports them. This ticket turns the guarantee on once they do.
