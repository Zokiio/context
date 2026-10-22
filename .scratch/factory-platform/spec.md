# Waymark factory platform

## Direction and authorization

On 22 October 2026, the user chose to build the factory and its platform before pilot work. The user then requested all platform surfaces and authorized coordinating agents to build them. This includes the workflow core, local-first service and CLI, web workspace, and shared multi-user access.

Waymark owns development intent, accepted requirements, readiness, workflow policy, and acceptance. Existing coding agents execute scoped work through adapters. Gups is a later development target. Running Gups or implementing a Gups adapter is not a prerequisite.

No external workflow engine has been selected. Engine installation, paid or provider-backed evaluations, publication, and deployment are outside this build's authorization. Local implementation, offline fixture executions, temporary Git repositories, and an integrated local browser demonstration exercise the platform itself.

## Required outcome

An operator can start one durable Waymark service, register an existing project, admit a ready work item, dispatch a bounded implementation, inspect the candidate and independent evidence, and record acceptance. CLI and browser clients operate on the same state with service-enforced user and project permissions. The complete platform must pass its own acceptance before scheduling Gups pilot work.

## Ownership

- The existing OKF bundle remains authoritative for project goals, work items, requirements, decisions, and recorded acceptance. Registering a project references these records and preserves their identities.
- The factory controller owns operational job state, implementation claims, attempts, deadlines, repair bounds, and reconciliation. It does not maintain a competing editable backlog.
- One workspace owner allocates isolated worktrees at recorded base revisions and retains failed or uncertain workspaces.
- Worker adapters invoke existing tools and retain their execution identities and actual outcomes. A worker's final message is an observation, not acceptance evidence.
- Authorized independent verification and review supply separate outcomes for the exact candidate. Workflow acceptance retains the actor, requirement version, evidence, and tested revision.

## Platform capabilities

### Project and job admission

Resolve the record store, code repository, allowed source roots, and selected work item explicitly. Require complete bounded context and ready, authorized work. Capture the context bytes, requirement fingerprints, repository base, selected executor, required checks, deadline, and repair limit. A read result does not itself authorize execution.

Duplicate requests must not create duplicate work. One active implementation claim may exist for a work item within the service authority. A second operator sees the existing job or an explicit conflict.

### Durable execution

Record intent before dispatch. Preserve attempt identity, workspace, worker identity, state transitions, receipts, candidate commits, artifacts, and actual observations across service restart. Use argument arrays at process boundaries. Do not replace a native agent's model or tool loop.

Reconcile interrupted or expired work before another execution. Missed heartbeats, a wrapper timeout, and cancellation acknowledgement do not establish that external work stopped. Keep failed, cancelled, pending, and unknown outcomes distinct. Do not silently replay an uncertain action.

Repair is explicit and bounded by the admitted limit and deadline. Preserve original failures. Changes to requirements or to a candidate after verification require reassessment of affected evidence.

### Verification and acceptance

Represent each required check as passed, failed, skipped, unavailable, or unknown, with actual command or procedure, observer, time, candidate, and retained evidence. Only an executed passing observation satisfies a required executable check. Missing reviews and checks keep a job unaccepted.

Implementation, verification, review, and acceptance have separate roles and results. A successful process exit cannot automatically accept the work. Acceptance verifies the candidate identity and current selected requirements, preserves original evidence and attribution, and supports the existing Acceptance record contract.

### Shared service and clients

The service authenticates distinct users and enforces project access and operation permissions independently of the UI. The first release may use local access tokens. Do not claim enterprise SSO or distributed service ownership.

The CLI and web workspace use the same application operations. Users can inspect projects, work items, jobs, requirements, events, candidate revisions, checks, reviews, and remaining actions. Authorized users can admit work, request cancellation or repair, and review or accept a candidate. Show loading, empty, denied, stale, failed, and uncertain states honestly.

The browser renders untrusted content as text. Tokens never belong in URLs. Platform state is private operational data outside the authored project record store and outside disposable recovery caches.

## Compatibility

Preserve the behavior and JSON contracts of `ctx context`, `ctx orient`, and `ctx resume`. Their operations remain read-only. Existing configuration, initialization, project identity, and recorded acceptance retain their ownership. The platform adds application services rather than making reader commands dispatch workers.

## Whole-platform acceptance

1. A real local service, CLI, and browser complete the same project and job flow against shared durable state using separate users.
2. An offline fixture worker produces a candidate in a real temporary Git worktree. The demonstration identifies it as a fixture, not model-backed work.
3. Independent verification and review permit acceptance only for the exact admitted requirements and candidate, and the resulting record is inspectable through the existing reader contract.
4. Tests cover duplicate admission, conflicting claims, access denial, restart during active work, stale requirements, changed candidates, missing or skipped required checks, bounded repair, and uncertain cancellation without duplicate dispatch.
5. Existing reader checks pass. New behavior has meaningful focused tests and an integrated browser check. Generated logs, runtime stores, and acceptance evidence remain outside Git.
6. Document the real startup, user setup, worker configuration, recovery, trust boundaries, and verification limits. No broad factory-readiness claim follows from a health endpoint or a worker's completion message.

## Delivery scope

Build the controller, service, clients, workspace/worker integration, and complete platform verification as independently reviewable outcomes. Keep Gups changes outside these outcomes. The architecture and implementation contracts are recorded alongside the platform design before code depends on them.
