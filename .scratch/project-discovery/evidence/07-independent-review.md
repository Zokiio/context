# Independent review before merging discovery

Recorded by workflow coordinator `Codex /root` at `2026-09-17T11:32:10+02:00`. Three fresh reviewers independently examined `2777f087a5e320594e09c5cda1f35d62e66c5a80` against `1eb54f87242502020181b46504d9556e5bea6a6a`, including each stacked PR boundary. None of these reviewers implemented the stack in the preceding work. These are agent reviews, not human approvals.

## Standards

Reviewer `/root/merge_review_standards` found no hard standards violation at the full-stack tip. The intermediate reader README still described the old direct `--project` contract. The coordinator corrected it in `cd9d309`, and the reviewer independently verified README blob `48666c1809bfee85664ef45d67d91f31aabd9b71`. The correction is included in PRs #7 and #8 before merge.

One P3 judgement call remains: the CLI and workspace layer duplicate a shallow directory-access probe. The reviewer demonstrated no behavior defect. This is optional maintenance work.

- [Original standards report](07-review-probes/standards.md).
- [First README recheck](07-review-probes/standards-readme-recheck.md).
- [Passing README recheck](07-review-probes/standards-readme-final.md).

## Spec

Reviewer `/root/merge_review_spec_readers` found no confirmed spec findings for PRs #4, #6, #7, and #8 or their final documentation contracts. Thirty independent CLI probes passed. Full Go tests also passed from isolated exports at each of those four original PR heads.

Reviewer `/root/merge_review_spec_setup` found two P2 defects in PR #9. Shared and personal writers can both commit conflicting bindings because they lock different destination files. The reviewer reproduced the race within one process and across two OS processes. Replacement also loses ACLs and group ownership, despite retaining mode bits. No additional documentation or trial defect was identified in PR #10. Existing checks and all 37 trial scenarios passed, demonstrating that these cases were missing from the prior verification.

- [Reader and workspace report](07-review-probes/spec-readers.md), [CLI probes](07-review-probes/spec-readers-probes.json), and [intermediate test results](07-review-probes/spec-readers-slices.json).
- [Setup report](07-review-probes/spec-setup.md).
- [Same-process failure](07-review-probes/setup-race-original-evidence.json) and [cross-process failure](07-review-probes/setup-race-crossprocess-evidence.json).
- [ACL loss](07-review-probes/setup-acl-original-evidence.json) and [group-ownership loss](07-review-probes/setup-group-original-evidence.json).

## Disposition at this observation

PRs #4, #6, #7, and #8 were independently reviewed and merged in dependency order. Their merge trees exactly match the reviewed heads, including the separately reviewed README correction. Setup and final documentation remain unmerged while the two P2 fixes receive independent re-review and fresh verification. A later immutable observation will record those results; this report retains the original findings.

[The retained-file manifest](07-review-probes/manifest.json) records source locations and SHA-256 digests. Original reports and failed reproductions are preserved byte-for-byte. Earlier acceptance decisions and their tested revisions remain historical evidence.
