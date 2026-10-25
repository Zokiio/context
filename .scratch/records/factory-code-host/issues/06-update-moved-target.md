---
type: WorkItem
id: f0331553-18b6-412f-b176-dbd4d6d867eb
title: Update a pull request with a verified merge commit when the target moved
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- When the head does not contain the fetched target tip, the merge builds the merge commit on that tip as today and runs the local merge checks on it.
- After they pass, the service fast-forwards the pull request branch to the merge commit by compare-and-swap. It observes the required host checks on that commit, then asks GitHub to merge with it as the expected head. These checks never change the job's own status.
- A failed, skipped, or unavailable merge check, a refused push, or a target that moves again fails the merge and moves no branch. A retry rebuilds on the current tip, and only the latest generation of checks decides.
- Offline tests cover a moved target, a failed host check on the merge commit, and a target that moves during verification.
- A live run against a disposable GitHub repository that the user names covers the whole flow: a draft pull request, a host check, a mapped reviewer's change request and its repair, approval, a merge through GitHub, and the local fast-forward. It is reported separately from the fixture checks, and it runs only after the user's go-ahead at the time.

## Spec

- [Merge through the host](../../../factory-code-host/spec.md#merge-through-the-host)
- [Verification](../../../factory-code-host/spec.md#verification)

## Context

- [Code host ADR draft](../../../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md)

## Blocked by

- [Count GitHub checks on the pull request head as evidence](03-host-checks.md)
- [Merge an approved pull request through GitHub](05-merge-through-host.md)

## Blocked by decisions

None
