---
type: WorkItem
id: 2f0e308e-a049-4719-bbeb-614c5ac93ba8
title: Ask how an approved retrospective change ships
triage: ready-for-agent
execution: completed
---

# Ask how an approved retrospective change ships

Recorded on 2026-10-06 from the batch retrospective of the parallel Axpilot #266 and #275 runs. It concerns the retrospective skill that [ticket 03 of the workflow adaptation](../../workflow-adaptation/issues/03-retrospective.md) delivered.

## Evidence

After the #266 retrospective, the user approved an edit to Axpilot's AGENTS.md. The agent applied it as a commit on an unpushed local branch. The user then had to ask twice before it became Axpilot PR #292. Axpilot forbids pull requests without an explicit request, so the agent was right not to open one. The approval question never asked how the change should ship. The #275 retrospective avoided the gap by adding its approved documentation commit to the work item's open pull request.

## Scope

Change the retrospective skill's "Report and apply approved proposals" section. When the skill asks the user to approve a project guidance or automated check proposal, the same question asks how the change ships: on the work item's open pull request, or as its own. The skill follows the project's publication policy and opens no pull request the user did not choose. Tracker follow-ups and ctx feedback are unchanged.

## Acceptance criteria

- S1: When the retrospective skill asks the user to approve a project guidance or automated check proposal, the same question asks whether the change goes on the work item's open pull request or gets its own.
- S2: The skill publishes an approved change only by the route the user chose, within the project's publication policy.
- S3: The canonical template and generated development copy agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-10-06: Completed in [PR #43](https://github.com/Zokiio/context/pull/43) after independent Standards and Specification reviews and their renewals passed. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

## Acceptance

[05-retro-shipping-route-20261006.md](../acceptances/05-retro-shipping-route-20261006.md)
