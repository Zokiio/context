---
type: WorkItem
id: 979f50e8-f397-4472-8ab9-1ef7dd17ff9e
title: Import Jira Data Center issues through the same connection and binding
triage: ready-for-agent
execution: unstarted
---

The team project the user wants to groom runs on Jira Data Center. Its acceptance criteria sit in a section of the description. This ticket adds a second tracker kind behind the operations ticket 01 introduces. No live Jira instance is available to the build, so fixtures stand in for the team's instance until a live trial.

## Spec

- [Factory intake](../../../factory-intake/spec.md)

## Context

- [Tracker intake decision](../../../../docs/adr/0007-tracker-intake-ownership-and-delegated-pickup.md)

## Blocked by

- [Import a repository's open GitHub issues as draft work items](01-import-github-issues.md)

## Blocked by decisions

None.

## Acceptance criteria

- An installation administrator registers a `jira-datacenter` connection with an HTTPS base URL and a private personal access token file. Plain HTTP is accepted only for a loopback address. A project administrator binds it to a Jira project key with an optional JQL clause.
- "Refresh now" reads open issues in scope through REST API version 2 search with pagination, and imports them as unqueued WorkItems under the same rules as GitHub issues.
- The import converts description wiki markup to Markdown for headings, bulleted and numbered lists, bold, italics, monospace, `{code}` and `{noformat}` blocks, and `[text|url]` links. It keeps unknown macros as literal text.
- The list under an `h2.` or `h3.` heading named Acceptance criteria becomes the work item's criteria. "Is blocked by" issue links become Blocked by entries, resolved as the spec describes for GitHub blockers. Jira labels carry triage. The Done status category counts as closed.
- A sub-task's parent key and URL are recorded in frontmatter and shown on the Task page. They do not add context or blockers.
- Cards show the issue key, such as "ABC-123", linked to Jira. The Task page shows tracker-owned fields read-only with an "Edit in Jira" link.
- The cleaning, length limit, comment handling, raw retention, read-only requests and token rules from ticket 01 apply unchanged.
- Tests use a fake Jira Data Center server with recorded responses covering a wiki-markup description with a criteria section, issue links, pagination and a sub-task parent.
- The operator guide states that a Jira Data Center personal access token carries its user's full permissions, and recommends a dedicated user with browse permission only.
