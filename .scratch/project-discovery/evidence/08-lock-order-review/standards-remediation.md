# Standards review of setup remediation

Observed by `Codex /root/merge_review_standards` at `2026-09-17T09:39:39Z`. Reviewed setup `933d9c7fe593a75dffc6134d706c55c372fc05ab...a0ecc4cf143886e687c072ceffc9c8d1a0963a86` and docs `2777f087a5e320594e09c5cda1f35d62e66c5a80...b8008d67cb37b57bfd27e56af9d732585f4f6249`.

## Concrete finding

**P2, PR #9, `internal/discovery/setup_write.go:134`.** `slices.Sort(locks)` orders spellings, while `CanonicalPath` retains case aliases. On a case-insensitive filesystem, a shared checkout named `Beta` and the same physical home named `alpha` or `ALPHA` produce opposite orders for the same two physical locks. One process can hold the destination lock while another holds the registry lock, each waiting for the other. This defeats the local-writer coordination required by `docs/vision.md`, Product constraints: the persistence layer "checks expected revisions and coordinates local writers."

A bounded probe observed the opposite lock requests through the real `SetupPlan.apply` path and failed. It intercepted acquisition and stopped before document writes; it did not leave deadlocked processes. Order physical lock identities consistently and add a case-alias regression before merge.

## Other standards and documentation

No other hard breach found. Platform syscalls and unsafe ACL decoding remain private to the core's operating-system helpers. Dry-run and unchanged plans avoid lock creation. The documented owner, group, mode, ACL, unsupported-platform, and lock-footprint behavior matches the implementation apart from the concurrency edge above. Reopened tickets preserve earlier acceptance as history.

The earlier P3 possible Duplicated Code finding remains an optional judgement call. This remediation does not require that refactor.

## Evidence

Used pinned `git diff`, source/test reads, `git status`, and dependency lock-code inspection. Fourteen retained review-manifest digests matched; the three original standards reports also matched byte-for-byte.

The isolated-export command `go test ./internal/discovery -run '^TestStandardsReviewCaseAliasLockOrder$' -count=1 -v` exited 1. Details are in `standards-lock-order-probe.json`; the probe is `standards-lock-order-probe.go.txt`, SHA-256 `0ee270f7a03b7c98ca5892753cad01800f111ad8e58adaed69251d7cc9eda4b3`. No reviewed worktree files were edited. Full runtime validation remains with the coordinator and spec reviewer.
