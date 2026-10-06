---
type: WorkItem
id: 367f867f-b810-4d0e-a643-132eba00ab96
title: Name evidence files so agent harnesses accept them
triage: ready-for-agent
execution: completed
---

# Name evidence files so agent harnesses accept them

Recorded on 2026-10-06 from the Axpilot #266 retrospective's proposal 8, carried into the batch retrospective of the #266 and #275 runs. It concerns the evidence files that [short summaries](../../workflow-adaptation/issues/02-short-summaries.md) introduced.

## Evidence

The #266 checkpoint 1 dispatch asked the implementer for its failing-test evidence and a final report at `report.md`, holding gate results and deviations. The implementer wrote its per-slice `evidence.md` files. Its harness refused `report.md` with the reason "Subagents should return findings as text, not write report files". The orchestrator recorded that report from the hand-back. Later dispatches added an improvised fallback: return the content as text if the harness refuses a report file. Runners and reviewers in the same run wrote evidence files named by role and revision without trouble.

The verification brief calls each runner's, reviewer's, and verifier's output "the full report" written to "a new evidence file". The orchestrator's dispatch used the same word for the implementer's file. The brief states no fallback.

## Scope

Change the verification brief and orchestrate's dispatch rule. The orchestrator names each agent's output file as an evidence record, for example `evidence.md` or a role-and-revision name, and never as a report file. The brief states the fallback: when a harness refuses the write, the agent returns the full content as text and says so. The orchestrator then writes it unchanged to the named evidence path and records that it did. This fallback brings the full content into the orchestrator's context, so it stays an exception. Summary limits, review axes, and gates are unchanged.

## Acceptance criteria

- N1: Each dispatch names its evidence file path, and the orchestrate skill and verification brief describe that file as an evidence record rather than a report file.
- N2: The verification brief states the fallback: when a harness refuses the write, the agent returns the full content as text and says so, and the orchestrator writes it unchanged to the named path and records the fallback.
- N3: The canonical templates and generated development copies agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-10-06: Completed in [PR #39](https://github.com/Zokiio/context/pull/39) after independent Standards and Specification reviews and their renewals passed. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

## Acceptance

[06-evidence-file-name-20261006.md](../acceptances/06-evidence-file-name-20261006.md)
