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

## Blocked by decisions

None.

## Acceptance criteria

- The operator guide has a team installation section. It covers an always-on host with a TLS listener and `--public-url`, a service-manager unit, and workers that run as a different OS user from the service, with no human token files on the host.
- The same section covers adding teammates with read, coordinate, review, accept and merge, the pilot ask policy (always before and after the build, autopilot on), and the project webhook.
- The guide documents a backup taken while the service runs: an online SQLite backup of `factory.db` plus a copy of the evidence directory. Restore goes into an empty data directory on the same project paths. The guide states that attempts running at backup time come back as unknown and need reconciliation.
- A rehearsal test follows the guide with an owner, a teammate and an administrator against a TLS service with fixture workers. The owner plans a change, and a local receiver gets the notification. Its link opens the task signed in, and only the teammate is offered the approval. Approval verifies and merges the change.
- In the same rehearsal, a backup taken during a running build restores into a new data directory. The completed jobs, reviews and evidence are intact, and the running attempt is unknown.
- The guide lists what the pilot still needs from the push, PR and CI plan, and what this installation doesn't provide: other worker hosts, sandboxing and high availability.

## Comments

- 2026-10-25: A real pilot also needs the push, PR and CI plan's work on pushing merges, committing and pushing the service's record writes, and fetching the base. It isn't linked here because those tickets aren't on this branch yet.
