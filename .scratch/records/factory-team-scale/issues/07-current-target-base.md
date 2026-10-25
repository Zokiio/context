---
type: WorkItem
id: 7a0d1b8f-4368-4aa5-b242-3235fecb5a2f
title: Build from the team's current target branch
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Factory architecture](../../../../docs/factory-platform-design.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- A project can name the Git remote and target branch it builds from. A GitHub binding from the code-host plan supplies both when present. A project that names no remote behaves as today.
- Before admitting a job or starting locked tests, the service fetches the target branch from that remote. If the local branch is behind, the service fast-forwards it, moving the working tree when the branch is checked out and the ref otherwise. The job's or the tests' base is the fetched tip.
- If the local branch has commits the remote lacks, or a fast-forward would overwrite uncommitted changes, the service starts nothing for that project. The card says why and what to do. The service never merges, rebases, resets or discards local changes.
- If the fetch fails, nothing starts, and the card shows the error. The autopilot retries on its next pass. A build never silently falls back to a stale base.
- The admission records the fetched remote tip and the time of the fetch.
- Tests against a local bare remote cover these cases: a teammate's push followed by admission on the new tip, a diverged local branch, an uncommitted change that overlaps, an unreachable remote, and a project without a remote.

## Comments

- 2026-10-25: The code-host plan (`claude/plan-pr-ci-375e4f`, `.scratch/factory-code-host/spec.md`) fetches only the target tip at merge time and fast-forwards the local target after a factory merge. Admission still builds from local `HEAD` (`internal/factory/plan.go:182`). That plan's session asked for this fetch to live here. Share one fetch-and-fast-forward path with its post-merge step rather than adding a second.
