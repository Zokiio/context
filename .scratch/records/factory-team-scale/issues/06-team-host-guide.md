---
type: WorkItem
id: b957f49a-7c7f-472e-bd46-e6a4faa5ee4f
title: Document and rehearse a team installation
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Run a local factory with offline workers](../../../../docs/factory-platform.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Sign in once per browser](01-browser-session.md)
- [Show who each decision waits on](02-decision-owners.md)
- [Notify a project webhook when work waits on a person](03-notification-webhook.md)
- [Build from the team's current target branch](07-current-target-base.md)
- [Publish the service's record writes to the team](08-publish-record-writes.md)

## Blocked by decisions

None.

## Acceptance criteria

- The operator guide has a team installation section. It covers an always-on host with a TLS listener and `--public-url`, a service-manager unit, and workers that run as a different OS user from the service, with no human token files on the host.
- The same section covers adding teammates with read, coordinate, review, accept and merge, the pilot ask policy (always before and after the build, autopilot on), and the project webhook.
- It also covers the separate records repository chosen in the [record-location decision](../decisions/01-record-location.md): the side-by-side layout teammates clone, registering the project with the bundle in that checkout, and letting the bot account push to its branch. It covers logging `gh` in as a dedicated bot account rather than a teammate. Every factory pull request is authored by that login. Under a teammate's login, that teammate could never approve a factory pull request on GitHub, and the others could approve their own changes there.
- The guide documents a backup taken while the service runs: an online SQLite backup of `factory.db` plus a copy of the evidence directory. Restore goes into an empty data directory on the same project paths. The guide states that attempts running at backup time come back as unknown and need reconciliation.
- A rehearsal test follows the guide with an owner, a teammate and an administrator against a TLS service with fixture workers. The owner plans a change, and a local receiver gets the notification. Its link opens the task signed in, and only the teammate is offered the approval. Approval verifies and merges the change. A record edit the teammate pushes shows on the board, and the owner's task edit reaches the remote as a commit.
- In the same rehearsal, a backup taken during a running build restores into a new data directory. The completed jobs, reviews and evidence are intact, and the running attempt is unknown.
- The guide lists what the pilot still needs from the push, PR and CI plan, and what this installation doesn't provide: other worker hosts, sandboxing and high availability.

## Comments

- 2026-10-25: A real pilot also needs the code-host plan's pull requests and merges through GitHub (`claude/plan-pr-ci-375e4f`). They aren't linked here because those tickets aren't on this branch. Fetching the base (07) and publishing record writes (08) moved into this plan at that session's request.
