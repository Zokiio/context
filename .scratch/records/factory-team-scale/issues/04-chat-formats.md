---
type: WorkItem
id: af5ef52f-1669-4e24-b8df-0d6b9f1e7618
title: Format notifications for the pilot team's chat
triage: needs-info
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Blocked by

- [Notify a project webhook when work waits on a person](03-notification-webhook.md)

## Blocked by decisions

None.

## Acceptance criteria

- The project webhook setting accepts a format for the pilot team's chat, alongside the generic JSON payload. Build only the format the pilot needs.
- A notification renders in that chat as one short message with the project, the work item title, the decision, the people it waits on and a working link to the task page.
- Titles and usernames cannot trigger mentions. In Slack, the service escapes `<`, `>` and `&`. In Discord, it sends `allowed_mentions` with an empty `parse` list. No `@everyone`, `@here`, channel or role ping reaches the channel.
- Tests check the payload against recorded fixtures for the chosen format. A manual send to a real channel is reported separately.

## Comments

- 2026-10-25: Needs the pilot team's chat: Slack or Discord, possibly Microsoft Teams. Joakim doesn't know yet. Move to `ready-for-agent` once it is known.
