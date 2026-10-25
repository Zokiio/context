---
type: WorkItem
id: 72c55f2f-88e6-4db0-9f7c-de1408997a77
title: Import a repository's open GitHub issues as draft work items
triage: ready-for-agent
execution: unstarted
---

The Opsbase grooming trial captured 15 GitHub issues with a hand-written script. This ticket moves that capture into the service: a person refreshes, and the repository's open issues appear on the board as unqueued work items that keep their identity across refreshes. Pickup triggers, polling and queueing come in later tickets.

## Spec

- [Factory intake](../../../factory-intake/spec.md)

## Context

- [Tracker intake decision](../../../../docs/adr/0006-tracker-intake-ownership-and-delegated-pickup.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Tracker snapshots](../../../../docs/tracker-snapshots.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- An installation administrator registers a GitHub connection with a display name and a private token file, and a project administrator binds it to one repository, a records folder that defaults to `imported/<connection>/`, and optional default Context links. The CLI and the HTTP API use the same service operations.
- "Refresh now" from the CLI, the API or the board imports every open issue in the bound repository as an unqueued WorkItem and skips pull requests. Each record carries `sourceURL`, `sourceUpdatedAt`, `snapshotAt` and `execution: unstarted`, and its Comments list every local adaptation.
- The import maps the title, the description, the `## Acceptance criteria` list, `#NN` references under `## Blocked by`, and the five triage labels as the spec's mapping table states. An issue without a triage label gets `needs-triage`. An issue without a criteria section imports without criteria, and orientation reports it not ready.
- A later refresh finds each record by `sourceURL`, never changes its `id` or file name, and writes nothing when the issue is unchanged. It rewrites the tracker-owned fields through an expected-revision write only for an unstarted item with no refinement, test or job history. For any other item it retains the snapshot and leaves the record alone.
- Two records with the same `sourceURL` stop the import for that issue and show a diagnostic. Neither record is written.
- The import removes HTML comments, invisible format characters, bidirectional controls and Unicode tag characters, keeping the zero-width joiner inside emoji. It demotes headings that collide with record sections by one level, and notes each change under Comments.
- The import skips an issue whose description exceeds 4,000 characters, and the refresh result lists it with that reason. GitHub comments are not imported. The Task page shows their count and links to the issue.
- The controller retains each raw response in its operational store with its retrieval time and digest. The token appears nowhere in SQLite, logs, API responses, prompts, assignments or worker environments.
- A test against the fake GitHub server fails if the import sends any request other than a read.
- A card for an imported item shows one source fact, such as "GitHub #170", linked to the issue. The Task page shows tracker-owned fields read-only with an "Edit in GitHub" link and the snapshot time. The service refuses an edit to those fields with a message naming GitHub. Record-owned fields stay editable.
- Tests use a fake GitHub server serving fixtures shaped like the Opsbase capture, including pagination, a pull request in the issues feed, a hidden HTML comment and a bidirectional control. A real-browser check covers the card and the Task page.
- The operator guide and the HTTP reference describe connections, bindings and refresh. `docs/vision.md`, `docs/tracker-snapshots.md` and `CONTEXT.md` reflect ADR 0006.
