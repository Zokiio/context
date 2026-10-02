# Orchestration trial conclusions

Status: stable

These conclusions record delivery and verification limits as observed on 2026-10-02. GitHub publication identities were verified on that date. Original trial observations and local acceptance decisions retain their historical meaning.

## Isolated fixture

The [fixture WorkItem](../records/orchestration-trial/issues/01-explicit-orchestrator-trial.md) delivered the explicitly invoked project-local [orchestrate skill](../../.agents/skills/orchestrate/SKILL.md) and a standalone criterion-report fixture. The fixture reports authored observations with their evidence source and tested revision. A criterion with no observation retains a visible verification gap. It makes no acceptance decision and adds no verification view to the production `ctx` CLI.

The final evaluated implementation was `19de8f9694bab33bda00278ecb9890c316e8a4d0`. Seven committed fixture regressions and four independently derived cases passed. The independent workflow audit supported all six authored criteria, including scoped context delivery, correction integrity, and combined verification. The human responses were "looks good" for checkpoint 1 and "continue" for checkpoint 2.

[Waymark PR #24](https://github.com/Zokiio/context/pull/24) merged at `2026-10-02T07:43:06Z` with merge commit `038eaa58b61a402df5bf2dd27b7eb63668660a09`. The trial supports the checkpoint workflow mechanics. Active human review time and comparative review effort remain unmeasured.

## Axpilot #241

[Axpilot issue #241](https://github.com/Zokiio/Axpilot/issues/241) is closed. Summary now observes layout when it mounts, so pointer and keyboard resizing work immediately on first open. The local implementation completed at `67ca601a32c4575bcb6110e57c342241c45a42d5`.

Local independent combined verification supported all six criteria. The 89 component tests ran at `63d3513c8dc4761770d407cf3380baa7a51c374d`. The tested production and component sources were later verified byte-identical to the final candidate. The 16 Chromium and WebKit browser cases ran at the final candidate, `67ca601a32c4575bcb6110e57c342241c45a42d5`, with an isolated mock DesktopAPI. The human response "good" approved the checkpoint before independent combined verification.

[Axpilot PR #253](https://github.com/Zokiio/Axpilot/pull/253) merged at `2026-10-02T08:10:56Z` with merge commit `d8efa021142edb26203f5772b333ba457680197d`. Native-host and live-provider behavior remain unmeasured. The trial supplies no comparison of review effort or speed. Original completion evidence recorded the issue as open because it preceded the merge. That observation remains unchanged.

## Delivery and recorded acceptance

The fixture, `ctx init`, and CLI wayfinding WorkItems previously listed as current commitments record completed execution. Their delivery is distinct from acceptance reassessment. The original project orientation was partial because linked-source changes made historical acceptance stale, and the local `ctx-init` Acceptance record was missing.

Ignored local decisions and evidence are unavailable in a fresh checkout unless restored together or recreated through the acceptance procedure. Orientation continues to report the fixture's acceptance as unknown when its linked local decision is absent, even though execution is completed. Successful checks and merged PRs do not replace an Acceptance record. These conclusions renew no historical acceptance.

## Chosen next work

On 2026-10-02, the user authorized two WorkItems, both in progress pending human review and merge:

- [Publish current project and orchestration trial conclusions](../records/workflow-maintenance/issues/01-project-status-conclusions.md).
- [Install portable explicitly invoked checkpoint orchestration](../records/workflow-maintenance/issues/02-portable-checkpoint-orchestration.md).

The [discovery proposals](discovery.md), including status-authoring migration, remain deferred. Managed claims, automatic routing, supervision, and broader workflow adoption have no new authorization from these trials.
