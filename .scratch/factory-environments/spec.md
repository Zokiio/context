# Verify changes in a development environment

Verification runs registered commands in a clean worktree at the candidate. Nothing records which environment a change reached, which revision that environment runs, or what was observed there. The [vision](../../docs/vision.md#verification-and-development-deployment) requires local checks and development-environment tests to stay distinct, and untested criteria to show. Passing local tests does not establish delivery to a device.

The user approved this direction on 25 October 2026. It adds no current commitment. Each ticket waits for the trial that needs it, as described under [Gates](#gates).

## What exists

- A check profile has an `environment` label, `local` by default ([projects.go](../../internal/factory/projects.go)). It is free text, and every check result copies it.
- Every required check runs in parallel in its own clean worktree once implementation succeeds. A verified merge reruns all required checks on the merge commit ([workers.go](../../internal/factory/workers.go), [merge.go](../../internal/factory/merge.go)).
- The ask policy's `when_untested` setting approves only when the test author claims a locked test for every criterion ([board.go](../../internal/factory/board.go)). A requirement only an environment can establish already makes the autopilot ask, unless the author claims a local test covers it.

A deploy registered as a required check would race other jobs for one shared environment. It would also redeploy on every merge recheck and record the candidate it was given, not the revision the environment actually runs.

## Observed targets

Gups is the first intended target after platform acceptance. Its `deploy-gups-dev` workflow publishes only commits on `main` that the workflow itself signed, and Flux deploys them. A manual promote also refuses artifacts that workflow did not sign. Its operations guide marks this GitOps path "not live yet". Until then, a person deploys with `deploy/homelab-dev.sh`. Its smoke script runs either against an isolated local compose project or against gups-dev, and reports each component as passed, failed, or not checked. Gups cannot deploy an unmerged candidate.

The vision's Wi-Fi scenario deploys a change to a development environment before teammate review and merge. No factory trial has that environment yet.

StrongDM's [factory site](https://factory.strongdm.ai/) describes twins of Okta, Jira, Slack, and Google services. Each twin was checked against the live service until no further behavioral differences turned up. The twins let LLM-judged scenarios run thousands of times an hour, beyond live rate limits and costs. The site does not describe deploying to environments.

## Decisions

The user made these decisions on 25 October 2026. The [environment ADR draft](../../docs/adr/0010-factory-environments-observe-deployed-revisions.md) records the rationale.

1. **Observe after merge first.** The project's own pipeline deploys merged work. The factory asks the environment which revision it runs, waits until that revision contains the change, runs the environment's checks, and records what it saw. Deploying a candidate before approval comes later.
2. **The controller registers environments.** An environment is operational configuration, like a check profile. It is not a project record. The implementer can edit project files, so it must not be able to change what an environment check runs or how a revision is reported.
3. **The environment reports the deployed revision.** The factory never infers it from what it asked a pipeline or command to deploy.
4. **Environment observations stay separate from local checks.** They are attempts with their own purpose, as merge checks are. They never count toward the required checks that move a job to review.
5. **Each environment sets its deploy authorization.** When the factory deploys a candidate, the project policy marks each environment `preauthorized` or `ask`. `ask` is the default. Under `preauthorized`, agreeing the plan covers the deploy. Under `ask`, a person confirms on the card, and the autopilot never confirms for them. The environment's own access controls remain the enforcement.
6. **No twin concept.** A stand-in for an outside service is project test infrastructure, run by an ordinary check. Its result is local and never counts as a development-environment observation.

## Environment

An installation administrator registers an environment:

| Field | Meaning |
| --- | --- |
| `key` | Stable name, such as `gups-dev`. A key cannot equal an existing check profile's free-text environment label. |
| `displayName` | Shown on cards and in Run. |
| `revisionCommand` | Argv that prints the full commit the environment runs and exits 0. Any other result means the revision is unavailable. |
| `observeWindowSeconds` | How long to wait for the environment to run a change before recording it as not deployed. |
| `deployCommand` | Optional, used only by [candidate deploys](#deploy-a-candidate-before-approval). Argv that deploys a given revision. |
| `fixture` | Marks a fixture environment, valid only for fixture projects, as with checks. |

The record holds no credentials. Commands run in the environment worker's process environment, so access stays with the deployment tooling and the machine that holds its credentials.

A check profile whose `environment` field names a registered environment is an environment check. The project policy lists the environment checks to run in each environment. Policy validation refuses a required local check whose profile names a registered environment. It also refuses an environment check whose profile names another environment. A local result therefore cannot count as an environment result, nor the reverse.

## Observe a merged change

When a factory job's merge is recorded, in any mode, the controller queues one environment attempt for each environment in the current project policy. Environment attempts use a separate worker stage. Only a worker started for that stage claims them, so environment credentials can stay on one machine. The observer must not be the implementer.

The worker runs the revision command. When the reported revision contains the job's candidate in the project repository, the worker checks out the reported revision in a clean worktree and runs each environment check there. A later revision that still contains the candidate is a valid observation. After the checks, the worker reads the revision again. If it changed, the results are recorded as unavailable, because the checks did not observe one known revision.

When the reported revision does not contain the candidate, or the repository does not have it, the worker retries until the observation window closes. It then records the environment as not deployed, naming the revision it last saw. A squash merge removes the candidate from history, so it shows as not deployed.

Each observation retains the environment key, the job's candidate and merge revisions, the observed revision, the check, its status, the observer, the attempt, and the time. Statuses are `passed`, `failed`, `unavailable`, and `unknown`. Environment observations never change the job's status, its acceptance, or the trust count for merging without asking.

The Done card shows one fact per environment: running there and passed, failed there, not deployed, or waiting. A failed check does not report a problem by itself. The card tells the person, who can use the existing problem report. Run shows the observations folded under each environment.

Interrupted environment attempts become unknown on restart, as other attempts do. A person can reconcile them or queue the observation again.

## Requirements that need an environment

At approval the environment has not run the change yet, so an approval card must not present an environment-only requirement as tested. The test author may name a registered environment for a criterion that no local test can establish. That criterion counts as unchecked at approval. The card says it is checked in that environment after merge, and `when_untested` asks a person. After merge, the Done card links the criterion to that environment's observation.

## Deploy a candidate before approval

This is later work for a pilot whose development environment accepts unmerged builds.

- After the local checks pass, the job waits for the environment lease. One candidate at a time holds an environment, across jobs.
- Authorization follows the environment's policy setting: `preauthorized` proceeds, and `ask` shows a deploy decision on the card.
- The worker runs the deploy command with the candidate. It then requires the revision command to report exactly that candidate before running the environment checks, and reads the revision again afterwards.
- A deploy failure or revision mismatch is `unavailable`, because the candidate was not tested there. It does not use a repair. A failed environment check fails verification like any required check.
- Observations join the acceptance evidence with the environment and observed revision.
- An uncertain deploy keeps the lease until a person reconciles it, because the environment's state is unknown.
- Merge rechecks do not deploy. The controller records which job's candidate each environment last received.

## Not planned

- **Twins.** None of our trials has needed a stand-in the project could not run itself. Gups' local smoke mode already starts its dependencies, such as Postgres, in an isolated compose project inside one command. A behavioral twin is a product of its own, and its fidelity is established only against the real service. Revisit when a factory run has a requirement that can be checked only against an outside service its tests cannot call.
- **Device labs as a separate concept.** A device lab is an environment whose checks drive devices.
- Factory-held deploy credentials, production deploys, and rollbacks.
- Environments stored as project records, as in the [roadmap proposal](../../docs/software-factory-plan.md#domain-model-additions).
- Importing deploy results from CI. Ingesting GitHub checks could carry them later.

## Gates

- **01, observe a merged change:** the factory has merged Gups work that reached `main`, and gups-dev can report its running revision. One way is the Flux source revision once GitOps is live.
- **02, requirements that need an environment:** a Gups work item has a criterion only gups-dev can establish, or a test author is seen claiming a local test for one.
- **03, deploy a candidate before approval:** a pilot has a development environment that accepts unmerged builds and needs results before review.

## Related plans

- Merged work reaches gups-dev only after `main` is pushed and its Check workflow passes. Observation uses ancestry, so it works whichever path pushed the merge.
- Candidate deploys serialize jobs on an environment lease. Parallel runs must respect it.
- A stand-in run inside a check's sandbox remains a local observation.

## Glossary

Add these terms to `CONTEXT.md` with ticket 01, not before. Editing it stales every acceptance that snapshots it.

- **Environment**: a registered deployment target whose running revision the factory can read and in which it can run checks.
- **Environment observation**: a check result from an environment, bound to the revision the environment reported.
- **Deployed revision**: the commit an environment reports it runs, as distinct from the revision it was asked to deploy.

## Verification

Use a fixture environment whose revision command reads a file, with checks that pass and fail. Cover these cases:

- the reported revision contains the candidate
- the reported revision does not contain it
- the revision is unknown to the repository
- the revision changes during checks
- the observation window expires
- a local check names an environment
- the implementer claims an environment attempt
- a restart interrupts an attempt

Report live gups-dev compatibility separately from fixture checks.
