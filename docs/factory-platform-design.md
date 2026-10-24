# Factory platform architecture

Waymark turns project intent into scoped development work and evaluates the resulting evidence. The first platform provides a shared service, CLI, browser workspace, and external process workers. Gups is a later development target. The [platform specification](../.scratch/factory-platform/spec.md) owns the delivery criteria.

## Project knowledge and operations have separate owners

The existing OKF bundle remains authoritative for project goals, work items, linked requirements, decisions, and Acceptance records. Registering a project records its stable identity and existing locations. It does not copy a second editable backlog into the service.

The controller owns private operational state in SQLite: users, project grants, profiles, jobs, implementation claims, attempts, observations, and events. One process owns an installation's store and workspace root. Multiple clients use that service. Independent installations and Git checkouts do not share claims.

The [controller decision](adr/0005-factory-controller-owns-operational-state.md) records why transactions were selected over a custom journal. Both candidate designs covered all user surfaces. The independent comparison selected SQLite to reduce persistence and recovery machinery. The final design retains explicit worker receipts, policy-owned checks, and reconciled OKF publication.

## Readers supply the admitted requirements

The controller calls the existing task-context and orientation application operations with one bounded source capture. It requires complete selected context, complete identity inventory, and a ready, eligible selected work item. Service authorization is a separate check. Unrelated historical diagnostics remain visible without becoming invented prerequisites.

An admission captures the exact source text and digests, criteria and ticket fingerprints, resolved base commit, selected executor, project-required checks, deadline, and repair limit. The controller prepares an empty current Acceptance section before taking the fingerprints when necessary. Existing `ctx context`, `ctx orient`, and `ctx resume` operations remain read-only.

The project administrator defines executable profiles and required checks. A coordinator chooses an allowed executor but cannot omit required checks. Fixture profiles support acceptance only in a project explicitly designated for fixtures. Reader limits can be raised explicitly for a registered project without changing the CLI defaults.

## A job coordinates attempts

Submission atomically records the admitted job and its implementation claim. An idempotent request returns the original result. A conflicting active job cannot acquire a second claim for the same work item.

The workspace owner allocates an isolated Git worktree at the admitted base. An external worker claims an assignment containing the attempt and dispatch identities, immutable context, configured argument array, workspace, and limits. The worker registers durable intent before starting a process. Its result is an execution observation and a candidate commit.

Verification uses a separate clean worktree for the exact candidate and an independent worker identity. Each required check retains its outcome, command, environment, observer, attempt, artifacts, candidate, and requirement fingerprint. An independent reviewer records a decision for that same candidate and requirement version.

Running checks against the candidate means running the candidate's own tests. Project policy therefore names protected paths, and admission freezes them. An implementation candidate that changes a protected path fails before verification, so expectations stay outside the implementer's write authority even though verification executes inside the candidate.

Repair creates another bounded attempt and preserves prior failures. Changed requirements, changed candidates, missing checks, missing review, or uncertainty prevent acceptance. A worker's completion message and process exit are insufficient to accept a work item.

## Recovery preserves uncertainty

The service retains intent before external effects. Restart reconciles unfinished attempts and acceptance publication rather than assuming that an old process stopped. A worker retains its own dispatch receipt so a repeated assignment cannot silently become another invocation.

Cancellation requests a stop. It is distinct from a terminal observation. Missing heartbeats, a lost acknowledgement, or a wrapper timeout retain an uncertain outcome and the implementation claim. An administrative resolution records its actor, exact attempt, evidence, and observed result. It does not turn an assumption into exactly-once execution.

## Acceptance returns to the project record

Acceptance checks the exact candidate, current selected requirements, all required independent verification, and the current review. It records a publication plan before writing files.

Publication retains immutable UTF-8 evidence, creates an attributed OKF Acceptance record, and updates the ticket through an expected-revision operation. The ticket has one current Acceptance link. Interrupted publication compares each planned destination by digest and refuses conflicting content.

The controller reports accepted only after the existing orientation reader validates the published decision. The reader checks the decision's provenance and freshness; it does not rerun the worker or authenticate a provider account. Historical acceptance retains its tested revision. A newer repository HEAD alone does not invalidate it.

Generated evidence and decisions follow project retention policy. A fresh checkout without locally retained artifacts reports unknown acceptance until the decision and evidence are restored together.

## Native agents run through one adapter

`ctx factory agent` adapts Claude Code and Codex to the worker protocol. It supplies the admitted context and rules on standard input, keeps the agent from committing, and commits the agent's worktree changes as the candidate. Session, turns, and cost are execution observations, not evidence. A command may report progress notes, which the controller records as events without treating them as outcomes.

A project's own instructions for an agent follow its built-in rules, which win where the two disagree. The service enforces its guarantees itself, so instructions can shape how an agent works but cannot lift a check. Each run records the instructions it was given, and a job freezes them at admission like the rest of its policy.

## Decision models are advisory

An optional local decision model answers bounded questions in one forward pass, starting with whether each acceptance criterion names a checkable result. Answers carry the model, its digest, and a versioned question, and are cached by input digest. They inform people before admission and are copied into the admission snapshot, but readiness, admission, verification, and acceptance never read them.

## People make two decisions per change

A person agrees what done means before the build and approves the change after it. Every other step is either an agent's work or a worker's check. The board derives each card from the stored records alone: refinements, expectations, jobs, attempts, and events. It therefore needs no state of its own and cannot disagree with the record.

Approving a change records a review, an acceptance, and optionally a merge. Each keeps its own checks, and all of them share one decision ID, so a retry finishes the remaining steps and metrics count a single decision.

## The ask policy delegates decisions to a person's autopilot

A project's ask policy says which decisions a person still makes. The autopilot carries out the rest as the person who saved the policy or gave a go-ahead, and it holds no authority of its own. Saving a policy therefore requires the capabilities it delegates. Delegated records name their reason, so the history shows what a person decided and what was decided on their behalf. Merging without asking requires a record of merges with no reported problem, and a reported problem lowers the policy again.

## Merging is a separate human decision

Acceptance binds a decision to an exact candidate, and merging moves a branch. They stay separate records so a person decides when accepted work reaches a branch, even when one approval requests both. A merge rechecks the acceptance and builds a merge commit that names the job and approver. Verification workers then run the required checks on that commit, and the branch moves only when they pass, by fast-forward or compare-and-swap. A merge is recorded as checkout, ref, or contained. Merge checks are tagged on their attempts and never change the job's own status. Any failed, unknown, expired, or interrupted check fails the merge without moving a branch. Admission treats an accepted but unmerged factory prerequisite as unmet, so dependent work cannot start from a base that lacks it. Prerequisites accepted outside the factory are left to the reader's checks.

## Holdout expectations stay outside the implementer's reach

An expectation author writes tests from the criteria before implementation. It may change only protected paths, and a person accepts the result. A job started from accepted expectations uses their commit as its base, so the protected-path check keeps the implementer from changing them. The tests reach a branch only with the implementation, through the verified merge.

## Refinement precedes admission

A refinement proposes rewritten criteria and at most three blocking questions for one work item before admission freezes it. The refiner only reads; a person answers, edits, and accepts. Acceptance changes the authored ticket through an expected-revision write and records the answers under Comments, so the record shows how its criteria were shaped. The decision model scores criteria before and after as an advisory comparison. Refinement is refused while an active job holds the work item, because that job's requirements are already frozen.

## Metrics are measured, not estimated

Project metrics are computed from jobs, attempts, events, artifacts, refinements, and expectations. They report outcomes, rework, lead times, human decisions, and agent cost as observed. An agent's reported cost is an observation it made about itself.

## Shared access uses one policy

The service authenticates users and workers and checks capabilities for every operation. Project grants cover reading, coordinating, verifying, reviewing, accepting, project administration, and stage-specific worker execution. UI controls reflect these permissions; the service enforces them independently.

A project policy may let the coordinator review their own job, for a single-person project. The reviewer must still be human and never the implementer. `ctx factory solo` runs the service and both workers in one loopback process for that case.

The first platform uses private bearer credentials. The service binds to loopback by default. Shared network access uses an explicitly configured TLS listener. The browser renders project records and worker output as text and keeps credentials out of URLs.

Service authorization does not confine a native coding agent's filesystem, commands, network, or provider tools. The operator selects and trusts executable profiles and worker environments. Supporting another execution environment requires an adapter with honest identity, receipt, cancellation, and reconciliation semantics.

## Platform acceptance precedes target-project work

The first integrated demonstration uses the real service, CLI, browser, temporary OKF records, real Git worktrees, distinct users, and offline fixture workers. It covers the complete successful flow and the failure cases in the [platform specification](../.scratch/factory-platform/spec.md#whole-platform-acceptance).

Fixture results establish the exercised offline behavior. They do not establish live-provider compatibility or unattended operational readiness. Gups pilot work follows completion of the platform's own acceptance gate.
