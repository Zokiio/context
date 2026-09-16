# Ticket 05 verification

Workflow actor: Codex coordinating agent. Verification finished at 2026-09-16T12:46:05.214535+00:00. No human approval is asserted.

The [retained verification record](05-acceptance-verification.json) identifies the tested working tree based on `b266094c0e924606a0dd52fed3483355dacdc8db`, with a SHA-256 manifest of source and fixture files, exact commands and results, toolchain, and retained reader digest. This source identity describes the actual verification run; a later commit or merge does not replace it.

The original [ticket](../../records/session-orientation/issues/05-verify-prerequisite-acceptance.md) owns the criteria.

## Criteria observations

| Criterion | Evidence |
| --- | --- |
| 1 | Public-operation cases reject absent, multiple, wrong-type, duplicate-identity, and subject-mismatched current acceptance links. Only the authored current link is used. |
| 2 | Acceptance metadata cases cover human/workflow attribution, explicit-offset timestamps, historical origin/revision, integer version support, lowercase digests, and separately attributed optional human approvals. JSON retains unknown metadata. |
| 3 | Whole-body and criteria comparisons distinguish known stale fingerprints from unavailable or malformed information. A combined stale-and-unknown case preserves both findings. |
| 4 | Markdown-tree snapshot cases cover inline and reference links, exact one-link/one-inline-digest structure, malformed and conflicting entries, required sections, and nonempty evidence. |
| 5 | Membership tests require every directly selected Spec, Context, and blocking Decision source, and validate additional explicit requirement snapshots. |
| 6 | Authorized-root, external-record, missing/unreadable/binary source, raw-byte digest, shared-source budget, and exact-limit tests exercise the common reader and retained provenance. |
| 7 | Self-snapshot and self-evidence cases fail conservatively; subject-ticket snapshots compare the filtered requirement fingerprint. Historical acceptance records remain present and are not recursively expanded. |
| 8 | Missing evidence is unknown, changed evidence is stale, and a historical revision is retained without requiring current HEAD equality. Text/JSON assertions do not represent the report as a test rerun. |
| 9 | Direct leaf acceptance releases a dependent. Unstarted, in-progress, and cancelled prerequisites block; completed non-leaf prerequisites remain unknown during this documented bootstrap stage. |
| 10 | The CLI suite checks acceptance status, workflow actor and human approvals, historical revision, evidence references, ready dependent output, known-stale exit 0, and missing-evidence partial exit 1. |
| 11 | Public-operation and CLI suites cover the profile, freshness, membership, scope, attribution, and direct-edge cases through real temporary files, alongside existing context-reader regression tests. |
| 12 | Current acceptances for completed tickets 01 through 04 were checked with the retained intermediate reader. A clarification about preparing the Acceptance heading changed a required convention, so new decisions preserve the original test revisions and prior records. Before/after observations record stale then valid results. This slice follows the same procedure after its verified closeout. |

## Verification outcome

All retained commands passed. Real-project orientation returned 1 in both formats, complete=false and inventoryComplete=true. Reader commands left source files and Git state unchanged. The project still has unmigrated legacy profiles and an incomplete project manifest. Non-leaf prerequisite evaluation is deliberately unknown until ticket 06; these partial-report causes do not invalidate the separately verified acceptance records.
