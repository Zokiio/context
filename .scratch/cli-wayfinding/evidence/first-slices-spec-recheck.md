# Supplemental spec finding recheck

Finding: P3, compact decision rows reorder affected commitments by ID.

Status: resolved in `ec6dbb32394e45367fe622999093c2975b4563a9`.

Inspected the immutable commit with `git show` from the pinned review checkout. The renderer now indexes decision-affected references by source path, then emits them in `CurrentCommitments` order. This addresses the reproduced Z-before-A case while preserving the affected references and leaving JSON and detailed rendering unchanged. The added `TestCompactDecisionAffectedWorkUsesCommitmentOrder` supplies opposite commitment/decision orders and asserts the resulting decision row follows commitment order.

This supplemental review covers only the previously reported finding. The parent reports that the regression test and CLI suite passed; this reviewer did not rerun the changed tests from the pinned checkout. `/tmp/context-cli-review` remains clean at `88b7e23`.
