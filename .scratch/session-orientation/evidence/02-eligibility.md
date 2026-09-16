# Ticket 02 verification

Workflow actor: Codex coordinating agent. Verification finished at 2026-09-16T12:07:09.348176+00:00. No human approval is asserted.

The [retained verification record](02-04-verification.json) identifies the tested working tree based on `9666b19340834db9360c43c12dc2f40920e16c21`, with a SHA-256 manifest of source and fixture files, exact commands and results, toolchain, and retained reader digest. This source identity describes the actual verification run; a later commit or merge does not replace it.

The original [ticket](../../records/session-orientation/issues/02-list-eligible-work.md) owns the criteria.

## Criteria observations

| Criterion | Evidence |
| --- | --- |
| 1 | Operation tests expose four named checks and preserve all reasons; fail takes precedence over unknown, then ready. |
| 2 | Missing criteria are unknown; explicit empty, heading-only, and checkbox-only criteria fail. Link and code criteria remain structurally nonempty. |
| 3 | Authorized available selected documents pass; missing, malformed, outside-root, and invalid-UTF8 context cases remain unknown. Self-contained work needs no Spec. |
| 4 | Empty and literal None declarations pass. Missing, malformed, and nonempty unsupported relationships remain unknown and partial. |
| 5 | Mixed-state operation tests require unstarted, committed, ready-for-agent, and ready independently. Missing pickup metadata excludes work. |
| 6 | Active and backlog groups remain separate; uncommitted ready work stays in backlog and completed/cancelled records remain inventory only. |
| 7 | Partial commitment tests retain known positive membership. Duplicate IDs, invalid project identity, and incomplete inventory cannot authorize pickup. |
| 8 | Missing unrelated project information leaves independently established readiness and eligibility intact while the overall report is partial. |
| 9 | Shortlist follows authored commitment order with deduplication; other records retain stable identity/path ordering and task-context references. |
| 10 | The CLI readiness fixture verifies text/JSON facts, a complete known-blocked report with exit0, and missing-context partial output with exit1. |
| 11 | Public-operation malformed/partial/state cases and CLI readiness fixture pass. The frozen ticket01 build demonstrated the expected red behavior before implementation. |
| 12 | This immutable evidence and the shared verification manifest retain actual commands, outcomes, tested source identity, and read-only observations. |

## Verification outcome

All retained commands passed. Real-project orientation returned 1 in both formats, complete=false and inventoryComplete=true. Reader commands left source files and Git state unchanged. Nonempty decision and dependency semantics remain explicitly unsupported in this slice. Legacy records and the unpopulated project manifest also remain partial until migration.
