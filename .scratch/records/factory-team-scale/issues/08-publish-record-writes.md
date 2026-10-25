---
type: WorkItem
id: 7c71b080-bc29-4a4d-a364-20b071f8bf27
title: Publish the service's record writes to the team
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Where the pilot's records live](../decisions/01-record-location.md)
- [Factory HTTP and browser access](../../../../docs/factory-http.md)

## Blocked by

- [Build from the team's current target branch](07-current-target-base.md)

## Blocked by decisions

- [Where the pilot's records live](../decisions/01-record-location.md)

## Acceptance criteria

- Each of the service's record writes becomes a Git commit: task creation, edits and disposition, refinement acceptance, the ticket update from acceptance publication, and the merge note. The commit's author is the acting person and its committer is the service. Its message names the operation and request ID. It reaches the team's remote by the route the record-location decision selects.
- Before reading records to admit work, and before each record write, the service fetches teammates' record changes. A write whose expected revision no longer matches the fetched record is refused as stale, as today.
- When the remote rejects a push, the service fetches and retries only if the new remote commits touch none of the files it wrote. Otherwise the project holds with `publicationPending` and names the file. The service never force-pushes.
- A restart between commit and push resumes the push from the retained publication plan, and a commit already on the remote is not pushed again.
- Files that the project's ignore rules exclude stay uncommitted. In this repository's convention, that covers generated Acceptance records and evidence.
- Tests against a local bare remote cover each kind of write, a teammate's concurrent edit to another file, a conflicting edit to the same file, a rejected push, and a restart between commit and push.

## Comments

- 2026-10-25: Today these writes land as uncommitted files in the service host's checkout, and nothing commits, fetches or pushes them. The code-host plan's session confirmed that its plan doesn't cover them. It also set two constraints. Record commits must not go on a `waymark/<job>` PR branch, because they would change the head and discard its host evidence. A protected target branch refuses a direct push of record commits.
