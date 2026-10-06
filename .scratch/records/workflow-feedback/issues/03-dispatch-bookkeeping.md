---
type: WorkItem
id: c8d5451e-7101-4ca4-ae14-5bc06b54c544
title: Record each dispatch in one step and read packets by section
triage: ready-for-agent
execution: completed
---

# Record each dispatch in one step and read packets by section

Recorded on 2026-10-06 from the Axpilot #266 retrospective's proposal 7 and the batch retrospective of the parallel #266 and #275 runs. [Ticket 02](02-checkpoint-boundary-continuation.md) bounds the orchestrator's context; this ticket cuts the number and size of steps that pay for it.

## Evidence

The batch retrospective split each orchestrator message's input tokens across the tools it called. Input tokens are mostly cache reads.

| Category | #266 | #275 |
| --- | --- | --- |
| Bookkeeping: logging a dispatch, taking a digest, updating progress | about 68 messages, 34.4M (15%) | about 39 messages, 15.2M (11%) |
| Reading files through Bash | 70 messages, 30.6M (13%) | 65 messages, 20.0M (14%) |
| Screen and browser calls | 95 messages, 56.3M (24%), about 592K each | 28 messages, 15.4M (11%) |
| Text-only turns acknowledging a hand-back | 43 messages, 23.9M (10%) | 36 messages, 15.5M (11%) |

The #266 orchestrator read its plan of more than 300 lines and its checkpoint packets in full. Its bookkeeping ran as single-purpose Bash steps named Log, Digest, Record, Update progress, Mark, or Pin.

Screen work is outside this ticket. Axpilot approved its own guidance that moves the integrated demo to one dedicated agent.

## Scope

Change the orchestrate skill's dispatch guidance. The orchestrator writes a dispatch file, records its digest, and appends its log line in one scripted step per dispatch. After the plan review, it reads plans, checkpoint files, and evidence by section when it needs part of one. Retained dispatch records, their digests, and the exclusions are unchanged. A ctx write command for dispatch records is outside scope.

## Acceptance criteria

- K1: The orchestrate skill directs the orchestrator to write each dispatch file, record its digest, and append its log line in one scripted step.
- K2: After the plan review, the skill directs the orchestrator to read plans, checkpoint files, and evidence by section when it needs only part of one.
- K3: Each retained dispatch record still holds the paths, digests, worker identity, and revision the skill already requires.
- K4: The canonical template and generated development copy agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Dispatch checkpoint work by pointer](../../workflow-adaptation/issues/01-dispatch-by-pointer.md)
- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-10-06: Completed in [PR #40](https://github.com/Zokiio/context/pull/40) after independent Standards and Specification reviews and their renewals passed. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

## Acceptance

[03-dispatch-bookkeeping-20261006.md](../acceptances/03-dispatch-bookkeeping-20261006.md)
