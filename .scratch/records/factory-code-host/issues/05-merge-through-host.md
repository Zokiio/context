---
type: WorkItem
id: 7cf97f60-9707-4ce8-b0d8-7e00e210d34c
title: Merge an approved pull request through GitHub
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- For a bound project, Approve with the merge switch on and `jobs merge` keep today's rechecks. They then read the pull request, require it open with a head the service pushed for this job, and fetch the target tip.
- When the head contains that tip, the service asks GitHub to merge with the binding's method and the expected head commit. When it does not, the merge fails with a reason that says the target moved. Updating the pull request in that case is a separate ticket.
- The merge record has mode `pull_request`, the pull request URL, and GitHub's merge commit. The service checks that this commit's tree equals the verified head's tree and that its first parent is the fetched tip. A mismatch is recorded and shown, and that merge does not count toward trusted merges.
- The service fetches the merge commit and fast-forwards the local target branch under the existing checkout and ref rules when the local tip is the fetched tip. Otherwise the card says to update the local branch. Dependent work admits from the updated local branch.
- A refusal from GitHub fails the merge with GitHub's message and moves no branch. The existing retry rules apply.
- The service records its merge intent first. An uncertain reply or a restart is reconciled from the pull request's merge state.
- A pull request merged on GitHub before acceptance is recorded as observed, and the job stays unaccepted until a person approves it. A pull request closed on GitHub asks the person to reopen it or cancel.
- The autopilot's approvals merge through the same path.
- Offline tests cover a merge, a stale head, a refused merge, a tree mismatch, an interrupted merge, and merges and closes made outside Waymark. The operator guide documents merging a bound project.

## Spec

- [Merge through the host](../../../factory-code-host/spec.md#merge-through-the-host)

## Context

- [Code host ADR draft](../../../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

- [Publish a checked candidate as a draft pull request](02-draft-pull-request.md)

## Blocked by decisions

None
