# Final standards recheck of physical setup locks

Observed by workflow reviewer `Codex /root/merge_review_standards` at `2026-09-17T09:57:40Z`.

Reviewed setup commit `c59d12e6886874a00d42b821be24615a93c95ef6` and the setup documentation at `5a1e91fe88a32bde2b32b1f3ae3e19cdb456db7f`.

Result: the P2 case-alias lock-order finding is resolved. No new material finding.

The writer now orders and deduplicates locks by physical file identity. The unchanged independent probe that failed at `a0ecc4c` passes at this commit. Path casing no longer changes the acquisition order of the same physical lock pair.

Independent probes also passed for symlink/hardlink deduplication and stable ordering, rejection of directory/FIFO/device sidecars, release of the first lock after second-acquisition failure, and temporary-file cleanup plus subsequent lock reacquisition after rename failure. Four selected repository concurrency tests passed, including opposing home/destination roles and shared/personal processes. All nine selected top-level tests ran without skips.

Filesystem details remain private to the core's platform helpers. Unsupported identity queries fail before configuration replacement. Dry-run and unchanged plans bypass lock creation. The documented platform limitations, retained sidecars, and permission snapshot behavior agree with the implementation. The earlier optional P3 duplicated-directory-probe judgement call remains unchanged.

## Reproducibility

`standards-physical-lock-source-before.json` records the initially reviewed SHA-256 values. `standards-physical-lock-source-pinned.json` confirms that all five lock/writer source hashes match `c59d12e` exactly.

In an isolated `git archive` export, ran `go test ./internal/discovery -run <selected review and concurrency tests> -count=1 -timeout=30s -v`, exit 0. `standards-physical-lock-probe.json` contains the exact command and output. The original probe SHA-256 remains `0ee270f7a03b7c98ca5892753cad01800f111ad8e58adaed69251d7cc9eda4b3`; the additional probe SHA-256 is `8723351474acd602370306f32553894fea8682830aca1bb52cc1230609b27166`.

Runtime probes ran on macOS. Other platform identity helpers received static review. No reviewed worktree files or prior reports were edited. The coordinator and spec reviewer retain responsibility for full checks and the original permission regressions.
