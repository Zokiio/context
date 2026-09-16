# Ticket 06 verification

Workflow actor: Codex coordinating agent. Verification finished at 2026-09-16T12:59:35.097267+00:00. No human approval is asserted.

The [retained verification record](06-dependencies-verification.json) identifies the tested working tree based on `b6befb15d4db1f1ba87186a4c324cce34ee7c585`, with a SHA-256 manifest of source and fixture files, exact commands and results, toolchain, and retained reader digest. This source identity describes the actual verification run; a later commit or merge does not replace it.

The original [ticket](../../records/session-orientation/issues/06-evaluate-prerequisite-chains.md) owns the criteria.

## Criteria observations

| Criterion | Evidence |
| --- | --- |
| 1 | Public-operation and CLI accepted-chain cases require completed execution, current valid acceptance, resolved decisions, and satisfied prerequisites at every depth. Changed or unavailable leaf evidence remains visible through an otherwise-valid middle. |
| 2 | Direct and transitive DependencyEdge entries retain source and target references, original links, status, cycle membership, and reasons. Shared prerequisites reuse captured sources and produce consistent results. |
| 3 | Graph cases retain unfinished/cancelled failures together with missing targets, ambiguous identity, malformed declarations, and unavailable acceptance. Known failure determines blocked readiness without removing unknown findings. |
| 4 | Self-dependency and longer-cycle cases block cycle members and dependents. Finite closures identify participating identities and edges, including cycles through unfinished/cancelled work, while unrelated ready work remains eligible. |
| 5 | Fully observed cycle and known-stale cases produce complete orientation with exit 0. A context read of the same cycle remains complete with traversalComplete true. |
| 6 | Graph integration checks exercise exact-fit and partial collection across records, decisions, requirement sources, and evidence. Deduplicated captured sources count once; limit omissions remain unknown and incomplete inventory suppresses shortlisting. |
| 7 | Unrelated unavailable project information and unrelated missing graph targets retain independently proven eligible work where inventory, scope, identity, commitment, and all four checks are known. |
| 8 | The completed operation has no temporary unsupported milestone checks. Text and JSON expose the four checks, acceptance provenance, finite prerequisite graph, deterministic groups, and explained eligibility without choosing work. |
| 9 | Public-operation tests cover accepted chains and shared nodes, cancelled/unfinished work, decisions, stale/missing acceptance evidence, self/longer cycles, cycle dependents, unrelated eligible work, and combined known/unknown results. |
| 10 | Real temporary-project cases cover graph budgets and external-source deduplication. CLI checks distinguish complete-blocked exit 0, partial exit 1, and operation/invocation failure exit 2. Existing task-context regression suites pass. |
| 11 | Integrated verification retains exact commands, tested source manifest, reader digest, and real-project read-only observations. The current Acceptance record is authored after criteria closeout; prior records remain present and current requirement snapshots are rechecked. |

## Verification outcome

All retained commands passed. Real-project orientation returned 1 in both formats, complete=false and inventoryComplete=true. Reader commands left source files and Git state unchanged. All required readiness checks are implemented. The real project remains partial only because its legacy profiles and project manifest await ticket 07 migration. Root review found that cycle participation also needed attributed warning diagnostics; the public-operation regression now verifies one warning per participating edge without changing complete-cycle success.
