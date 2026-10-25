# Deliver factory work through the code host

Draft, 25 October 2026. It takes effect when the user selects the [code host plan](../../.scratch/factory-code-host/spec.md) for delivery.

## Context

A factory merge builds and verifies a merge commit locally, then moves a local branch. Teams review code and run CI on their code host, so a factory change today skips both. The vision already allows external CI results as completion evidence and keeps pull requests optional. It also defaults to the user merging after required checks and teammate approval.

The researched factories show how host feedback goes wrong. SuperPlane counts neutral, skipped, and cancelled check conclusions as nonfailing, and it starts an agent on any comment that mentions it. SWE-AF counts a pull request with no checks as passing, and it resolves review threads its own agent claims to have addressed. The [research note](../research/software-factory-2026-10-22.md) pins the sources.

## Decision

A project may bind one GitHub repository. For a bound project, the pull request is an addition to the local verified merge. The service opens it as a draft when the candidate passes the local checks, before the person approves. Host CI and teammates' reviews then arrive while that decision is open. The person's approval merges the pull request through GitHub with the expected head commit, and GitHub's merge commit must have the verified tree.

Host feedback can hold work back but never let it through. A host check counts only when project policy names it with the app that must report it, and only a success on the exact head commit passes. Skipped, neutral, cancelled, missing, and unreachable checks stay unverified and need a person. A host check counts only when protected paths cover the files that define it, because GitHub runs pull request workflows from the pull request's own version of those files. A "Request changes" review from a mapped Waymark reviewer on the current head becomes a recorded change request and repairs within the admitted limit. Approvals and other comments never become Waymark decisions.

The pull request's head must be a commit the service pushed. The service pushes only to `waymark/*` branches, by compare-and-swap, and never force-pushes over a commit it did not push.

The service acts through the operator's `gh` login and keeps the token out of every assignment, record, and agent or check environment it controls. It records its intent before each push, pull request change, and merge, and reconciles from GitHub's state after interruption.

## Alternatives considered

Opening the pull request after the person's approval keeps today's order. But the job's acceptance is then bound to a candidate the review can no longer change, so a teammate's change request would need a new job. Pushing the verified merge straight to the target branch needs no pull request, but protected branches refuse it, and it brings back neither review nor CI.

A repository-scoped token or a GitHub App would narrow what the service can do and give the factory its own identity on GitHub. The user chose the existing `gh` login for no setup. The binding names its credential source, so either can be added later.

Following SuperPlane's or SWE-AF's check policies would let a check that never ran look like evidence. Treating host approvals as the Waymark review would put a second approval route outside the person's decision.

## Consequences

The code host owns the final branch move for a bound project. Waymark's guarantee becomes tree identity with the verified commit, checked after GitHub merges. When the target moves between that check and GitHub's merge, the record says the trees differ and the merge does not count toward trust.

External effects add reconciliation states, and every host step must handle an unreadable GitHub. A required host check that GitHub never reports holds the job until a person acts.

Everything the factory does on GitHub appears as the operator. The operator cannot approve or request changes on factory pull requests, and branch protection that requires an approval needs a teammate. Any process running as the operator can use the same login. The head rule stops such a commit from being merged, and confining agents remains the worker environment's job.

A skipped or unavailable local check stops counting as a failure as well, so the autopilot no longer repairs code that did not fail.
