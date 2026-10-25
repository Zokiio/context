# Factory environments report their deployed revision

Status: draft, 25 October 2026. The user approved the decisions in the [environment specification](../../.scratch/factory-environments/spec.md); this record is adopted when that plan merges.

## Context

The [vision](../vision.md#verification-and-development-deployment) says project policy decides whether development deployment is preauthorized, and that "the workflow skill checks that policy and records the target environment and deployed revision". [ADR 0005](0005-factory-controller-owns-operational-state.md) moved operational state and policy enforcement from skills to the factory controller.

Verification runs required checks in clean worktrees at the candidate and reruns them on the merge commit. A deploy run as a required check would race other jobs for a shared environment, redeploy on every merge recheck, and record the revision it was given rather than the one the environment runs.

Gups, the first intended target, deploys only commits on `main` that its own workflow signed. It cannot run an unmerged candidate in gups-dev.

## Decision

The controller registers environments as operational configuration, like check profiles. An environment names a command that reports the revision it runs. An environment observation is bound to that reported revision, never to the revision a deploy was asked for. Environment checks are attempts with their own purpose and never count toward the required checks that move a job to review.

The first supported shape observes merged work after the project's own pipeline deploys it. Deploying a candidate before approval is later work. When it comes, each environment's project policy is `preauthorized` or `ask`, with `ask` as the default. Under `preauthorized`, agreeing the plan authorizes the deploy. Under `ask`, a person confirms the deploy, and the autopilot never confirms it for them. Deployment tooling and access controls remain the enforcement.

The factory has no concept of a twin. A stand-in for an outside service is project test infrastructure run by an ordinary check, and its results are local.

## Alternatives considered

Environment records in the OKF bundle, as the roadmap proposal suggested, would version them with the project. The implementer can edit project files, so it could change what an environment check runs or how a revision is reported.

Registering a deploy as a required check needs no new concept. It loses the deployed revision, cannot hold a shared environment exclusively, and repeats the deploy on merge rechecks.

Making deploy a third delegable ask-policy setting would let the autopilot confirm `ask` deploys. That would turn the per-environment human decision back into a project-wide default.

## Consequences

The vision's sentence about the workflow skill changes when this is adopted. The controller checks deployment policy, and the environment, not the skill or deploy step, supplies the deployed revision.

Under the first shape, environment evidence arrives after acceptance and does not join the Acceptance record. Requirements that only an environment can establish stay unchecked at approval.

An `ask` environment adds a decision beyond the [two per change](../factory-platform-design.md#people-make-two-decisions-per-change), only for candidate deploys to that environment.

A squash merge removes the candidate from the deployed history, so observing it in an environment requires a merge that preserves the candidate commit.
