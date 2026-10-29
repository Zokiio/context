---
type: WorkItem
id: c134d8cc-edb8-4d13-87ff-58179ce6afc4
title: Notify a project webhook when work waits on a person
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Factory HTTP and browser access](../../../../docs/factory-http.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Show who each decision waits on](02-decision-owners.md)

## Blocked by decisions

None.

## Acceptance criteria

- A project administrator can set, test and clear one HTTPS webhook URL per project. The controller stores it, and the API never returns it in full or shows it to anyone else. Plain HTTP is accepted only for a loopback receiver. `ctx factory serve --public-url` sets the base address for links.
- The service sends one notification when a card starts waiting on a person, using the decider rule from ticket 02, and one when no builder has started an admitted build within 15 minutes. It sends again only after the card leaves that state and returns to it, or the decision changes.
- A failed check that starts an automatic fix sends nothing. The message after the last allowed fix names the failing checks.
- The payload is a versioned JSON object with the project, work item ID and title, the decision or stall, the usernames it waits on, the task page URL and the time. It carries no requirement text, diff, worker output or check output.
- The service records each notification before sending it. A failed or uncertain send is retried with backoff for up to 24 hours, and a restart resumes pending sends. A duplicate delivery is acceptable; a dropped one is not. Administrators see the last delivery error on the project page.
- Tests with a local HTTPS receiver cover each trigger, deduplication across polls and restarts, retry after a failed send, the stalled-build rule, and the absence of record and worker content.

## Comments

- 2026-10-25: The pilot team's chat isn't chosen yet; Slack and Discord are both possible. This ticket delivers the generic JSON payload, and ticket 04 adds the chat format once the chat is known. Email and direct messages are out of scope.
