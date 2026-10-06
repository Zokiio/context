---
type: WorkItem
id: 041a888f-345e-4a14-aee5-88579b649eef
title: Continue orchestration from records at checkpoint boundaries
triage: ready-for-agent
execution: in-progress
---

# Continue orchestration from records at checkpoint boundaries

Recorded on 2026-10-06 from Axpilot retrospectives. Three runs raised orchestrator context size: #274 proposal 1, #266 proposal 7, and #275 proposal 6. The [specification](../../../workflow-adaptation/spec.md) for pointer dispatch and short summaries named orchestrator input tokens as the measure for the next run. [Ticket 03](03-dispatch-bookkeeping.md) covers the per-step cost.

## Evidence

Orchestrator input tokens by run, mostly cache reads. #254 and #264 ran before the workflow adaptation. The later four are based on Axpilot PR #276, which committed Waymark 541e468 with pointer dispatch and short summaries.

| Run | Session | Orchestrator input | Peak context |
| --- | --- | --- | --- |
| #254 | shared | 52M | not measured |
| #264 | shared | 74M | not measured |
| #272 | own | 63.7M | 478K |
| #274 | second work item in the #272 session | 55.3M | 710K |
| #275 | own, fresh | 141.5M | 674K |
| #266 | own, fresh | 237.1M | 833K |

The #274 retrospective proposed one work item per fresh session, and #275 and #266 followed it. Their orchestrator input still reached 1.9 and 3.2 times the highest earlier run. Context grows within one run, and every orchestrator step pays for the whole context. #275 never compacted its context across both checkpoints, the demo, acceptance, and the retrospective setup.

## Scope

Make checkpoint boundaries hand-off points in the orchestrate skill. After a checkpoint's gates and any human gate pass, the orchestrator continues the work in a fresh session. It starts that session with the environment's session tool when one exists, and otherwise gives the user the prompt that starts it. The new session rebuilds its basis from the plan, progress, dispatch records, evidence, and authorization on disk, as the skill's first section already describes. The skill currently says the orchestrator retains the whole task. Restate that so the retained records hold the task rather than one conversation, consistent with orchestration design choice 13 and the glossary. Gates, review axes, and human review policy are unchanged.

On 2026-10-06 the user chose a fresh session over compaction. A fresh session works in every agent environment. Compaction depends on the harness and keeps a summary the orchestrator did not write to disk.

## Acceptance criteria

- B1: The orchestrate skill names each checkpoint boundary, after the checkpoint's gates and any human gate pass, as the point where the orchestrator continues the work in a fresh session.
- B2: Before the hand-off, the orchestrator confirms that the plan, progress, dispatch records, evidence, and authorization on disk hold what the next checkpoint needs. The fresh session resumes from those records, not from conversation history.
- B3: The orchestrate skill and the initialization guide state that the retained records hold the whole task. The orchestrator role keeps its coordinating responsibility.
- B4: The canonical templates and generated development copies agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Orchestration design](../../../workflow-improvements/orchestration-design.md)
- [Dispatch checkpoint work by pointer](../../workflow-adaptation/issues/01-dispatch-by-pointer.md)
- [Return short check and review summaries](../../workflow-adaptation/issues/02-short-summaries.md)
- [Domain glossary](../../../../CONTEXT.md)
