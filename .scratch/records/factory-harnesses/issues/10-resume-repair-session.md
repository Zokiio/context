---
type: WorkItem
id: 5e731da0-bce2-4a42-adaf-f1ca07eac39c
title: Resume the builder's session on repair
triage: needs-triage
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#native-resume-if-decided-later)
- [Resume decision](../decisions/01-resume-native-sessions.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

- [Tell a repair why the previous attempt failed](01-repair-brief.md)
- [Count repairs that pass verification](02-repair-pass-count.md)

## Blocked by decisions

- [Resume native sessions on repair only if briefed repairs keep failing](../decisions/01-resume-native-sessions.md)

## Acceptance criteria

- An ADR records that a repair may resume a provider-local session as an accelerator outside the dispatch record, before implementation starts.
- A repair resumes only the session of the same job's immediately preceding implementation attempt, and only when every check passes. That attempt ended with a known outcome. The executor profile key, harness CLI version, model and worker identity match. The admitted requirement fingerprint, context digest, protected paths and builder instructions match. Waymark generated the session ID, and the transcript exists.
- The resumed session works in the new attempt's workspace with the profile's confinement restated. Codex resume passes `-c sandbox_mode=workspace-write`, and a harness that cannot rebind its working directory starts fresh.
- A failed check starts a fresh session, and the attempt records which check failed. A resumed repair still receives the full brief and admitted context.
- Waymark deletes the transcripts it persisted for resume when the job ends.
- Metrics separate the pass counts of resumed and fresh repairs.
- Fixture tests cover each failed check. The live check covers one resumed repair per supported harness.
