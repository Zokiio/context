# Reassess compact decision ordering

Actor: Codex /root. Observed on 2026-09-19 Europe/Stockholm. Tested revision: `ec6dbb3`.

An independent GPT-6 Astra specification review of `88b7e23` found one P3 defect. The separate decision row listed affected work in ID order, while the specification requires authored commitment order. The [original review](first-slices-spec-review.md) and [earlier slice evidence](02-compact-orientation.md) remain unchanged. The earlier acceptance did not detect this ordering failure.

The renderer now visits current commitments in authored order and selects the matching affected-work references. It preserves the reference values and does not change the orientation result, JSON, or detailed renderer.

`TestCompactDecisionAffectedWorkUsesCommitmentOrder` gives the renderer commitments Z then A and decision references A then Z. It verifies that the decision row presents Z then A. `go test ./internal/cli` passed at `ec6dbb3`; the [retained output](02-order-tests.txt) includes the complete CLI package suite. The CLI build also passed.

All six original criteria remain covered by the earlier integrated full-suite run and the corrected CLI run. Criterion 2 has now been reassessed against the independent finding and its regression test. This is a new workflow acceptance decision and does not alter the earlier reviewer or test observations.

For precision, the initial task-context binary was built at `c3d37e4`, before the record-only start commit `7b93da4`. Its product source is identical at those revisions. Earlier slice evidence uses `7b93da4` as the equivalent initial reader source state. The current reassessment uses a rebuilt `ec6dbb3` reader and freshly collected fingerprints.
