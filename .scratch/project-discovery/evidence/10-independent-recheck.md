# Independent review after setup corrections

Recorded by workflow coordinator `Codex /root` on 2026-09-17. The reviewers independently rechecked source commit `c59d12e6886874a00d42b821be24615a93c95ef6`. These are workflow reviews; no human approval is asserted.

## Standards

[The Standards reviewer](10-review-recheck/standards.md) found no remaining material finding. The original case-alias lock-order probe now passes. Independent probes also passed for symlink/hardlink aliases, special-file rejection, unlocking after acquisition failure, and temporary-file cleanup after rename failure. All nine targeted tests ran without skips. The platform and sidecar documentation matches the implementation.

The earlier optional P3 duplication of shallow directory probing remains a maintenance suggestion, with no demonstrated behavior defect.

## Spec

[The Spec reviewer](10-review-recheck/spec.md) found no remaining setup finding. Native macOS tests passed with race detection; native Linux setup, permission, and CLI suites passed. Original shared/personal concurrency and group/ACL-loss regressions passed. All 200 independent ACL revocations rejected stale writes and retained the revoked ACL, original bytes, and inode; 137 attempts naturally shared the original ctime. CLI checks also covered dry-run/no-op immutability, source authorization, YAML values, and registration identity.

Windows and other platforms were not runtime-tested by this reviewer. Windows and alternate architectures received compilation checks; supported behavior and safe refusal are documented explicitly.

## Integration and retained history

The first four reviewed PRs were merged in order. Setup source was integrated into the documentation branch at `f880dc3`, with all 110 program sources exactly matching the tested commit. The README conflict was resolved by keeping the final discovery documentation; it already contains the corrected selector migration and links to the detailed reader reference. The latest trial link was then updated. [All 17 local documentation links resolve](10-review-recheck/documentation-links.json).

[The new trial](06-reassessment-trial.json) passes all 37 scenarios against the exact tested binary. Setup and documentation acceptance records are being reassessed before their merges. The final closeout records the resulting acceptance state.

The [original review](07-independent-review.md), [case-alias failure](08-lock-order-review/standards-remediation.md), and [equal-ctime ACL failure](09-acl-snapshot-review/finding.md) remain unchanged. The [retention manifest](10-review-recheck/manifest.json) identifies these final reports by source and SHA-256. Original acceptance decisions retain their original tested revisions.
