---
type: WorkItem
id: b6fd9f7f-d59d-40ca-bcb6-c41eafe69ca9
title: Publish a checked candidate as a draft pull request
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria

- A project administrator binds a registered project to one GitHub repository through the CLI and the API. The binding names the owner and repository, the Git remote, the default target branch, the merge method (`merge` or `squash`), and a reviewer map from GitHub login to Waymark username. Binding fails unless `gh` is logged in and can push to the repository. An unbound project behaves as before.
- When every required local check passes, the service pushes the candidate to `refs/heads/waymark/<job>` and opens a draft pull request against the target branch. The title is the work item's title. The body, written by the service, lists the criteria, each check's result, the requirements no locked test covers, the job ID, and a request to ask for changes instead of pushing.
- A repair's candidate fast-forwards the same branch, and the service updates the body. Every push is a compare-and-swap against the tip the service last pushed. The service never force-pushes and never pushes outside `refs/heads/waymark/*`.
- The job and its card show the pull request's number, URL, and head commit. A head the service did not push blocks every later host step, and the card says the branch has commits Waymark did not build.
- With no required host checks, the job becomes `ready_for_review` once the pull request exists. The service marks the pull request ready for review exactly while the card waits in Approve, and returns it to draft when a repair starts.
- Cancelling, rejecting, or superseding the job closes its pull request with a one-line reason and keeps the branch.
- API calls go through `gh api`, and pushes use `gh` as the Git credential helper. A sentinel token never appears in assignments, events, artifacts, receipts, or logs. Agent and check processes started by the worker or solo run without `GH_TOKEN`, `GITHUB_TOKEN`, or `GH_ENTERPRISE_TOKEN`.
- The service records its intent before each push, pull request creation, draft or ready change, and close. After a restart or an uncertain reply, it reconciles from the remote ref and a lookup by head branch, without duplicate pushes or pull requests.
- Offline tests use a fake `gh` and a bare repository as the remote. The operator guide documents binding, the branch namespace, the login's reach, and the head rule.

## Spec

- [Bind a project to a repository](../../../factory-code-host/spec.md#bind-a-project-to-a-repository)
- [Delivery flow](../../../factory-code-host/spec.md#delivery-flow)

## Context

- [Code host ADR draft](../../../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None

## Blocked by decisions

None
