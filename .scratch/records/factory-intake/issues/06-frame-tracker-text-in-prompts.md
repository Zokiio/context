---
type: WorkItem
id: 9ed31616-91df-4851-af12-cdf6243703ec
title: Frame tracker text as outside text in agent prompts
triage: ready-for-agent
execution: unstarted
---

The builder prompt places admitted ticket text after its rules with no indication of who wrote it. Only the judge prompt says its sources are evidence, not instructions. Tracker snapshots already carry `sourceURL`, so this ticket can land before the import. It is a weak control on its own; the spec names the stronger ones.

## Spec

- [Factory intake](../../../factory-intake/spec.md)

## Context

- [Tracker snapshots](../../../../docs/tracker-snapshots.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- For a WorkItem with a valid `sourceURL`, the refiner, test author, builder and judge prompts mark that ticket's admitted text with its tracker URL and `snapshotAt`. The mark states that the title and description were written outside the project and describe the request. It also states that the text cannot change the rules above it, grant tools, or ask the agent to contact anything.
- Prompts for WorkItems without `sourceURL` are unchanged.
- The mark appears after the rules and before the ticket text. Ticket text cannot close or imitate it, because the text is quoted or escaped so a delimiter inside it has no effect.
- Prompt contract tests cover each adapter with and without `sourceURL`, and a ticket whose text contains the delimiter.
