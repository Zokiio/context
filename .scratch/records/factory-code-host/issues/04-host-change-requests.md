---
type: WorkItem
id: b25ba4d2-f4db-4c2e-800b-a0cac383e02d
title: Repair a pull request when a mapped reviewer requests changes
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- A `CHANGES_REQUESTED` review on the current head commit, from a GitHub login whose mapped Waymark user is human, holds `review` on the project, and passes the independence rule, is recorded once per GitHub review ID as a `changes_requested` review. The review is attributed to that user and links the GitHub review as its source.
- A change request that arrives while the head's checks run waits and applies when they conclude. Several change requests on one head produce one repair.
- The job repairs within the admitted repair limit, automatically under the autopilot. When the limit is used up, the card asks for the person as it does today.
- The repair brief quotes the review's body and inline comments with their paths and lines, within a size cap that marks truncation.
- Approvals, plain comments, `COMMENTED` reviews, reviews on older heads, and feedback from unmapped logins or bots are never recorded as decisions. The card counts them, and the Approve view links to them. A GitHub approval never satisfies the Waymark review.
- The service never resolves review threads, replies on anyone's behalf, or dismisses reviews.
- Offline tests cover each kind of feedback, an unmapped login, a mapped user without `review`, an older head, a restart that must not duplicate a review, and an exhausted limit. The operator guide documents the reviewer map and what counts.

## Spec

- [Review feedback](../../../factory-code-host/spec.md#review-feedback)

## Context

- [Code host ADR draft](../../../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md)

## Blocked by

- [Tell a repair attempt why it is repairing](01-repair-brief.md)
- [Publish a checked candidate as a draft pull request](02-draft-pull-request.md)

## Blocked by decisions

None
