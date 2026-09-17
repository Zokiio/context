# Setup write correction verification

Actor: Codex `/root/configuration_pr_review`, coordinated by `/root`, with permission implementation by `/root/setup_permissions_fix` and independent Standards and Spec reviews.
Observed: 2026-09-17.
Tested revision: `c59d12e6886874a00d42b821be24615a93c95ef6` from `https://github.com/Zokiio/context`.

All five ticket-05 criteria passed at this revision. The original setup acceptance and evidence remain unchanged. The [earlier write reassessment](05-setup-write-reassessment.md) recorded passing suites at `a0ecc4c`; later independent checks found case-alias lock ordering and equal-timestamp ACL races. This observation includes those corrections and their independent rechecks.

## Verification and source identity

The [check record](05-setup-write-final/checks.json) retains exact commands, exit statuses, source digests, and the reader binary digest. Uncached `go test -count=1 ./...`, uncached `go test -race -count=1 ./...`, `go vet ./...`, and `go build ./...` passed. [Full test output](05-setup-write-final/tests.json) and [race output](05-setup-write-final/race.json) retain all results. The six test packages passed. The subprocess helper's top-level skip is intentional; process tests execute it separately. Native macOS could not exercise changing to a different owner at UID 501; the independent Linux root run did.

Linux amd64/arm64, Windows amd64, and Darwin amd64 discovery test binaries cross-compiled. The reader binary built at the tested revision has SHA-256 `0d5620b44c86342e48559b1f941f5081f3e7194c10370deb18913c0c9bdd0871`.

The later normal merge `679e0731ab30b6cf0533c7b1935dfa3fefbf7d97` brought in the reviewed README selector correction from `af979d2`. Only `readme.md` changed. The [before](05-setup-write-final/source-before-merge.json) and [after](05-setup-write-final/source-after-merge.json) manifests have identical digests for all 110 files under `cmd`, `internal`, `go.mod`, and `go.sum`. The tested revision remains `c59d12e`, not the documentation merge.

## Independent rechecks

The [Standards recheck](05-setup-write-final/standards-physical-lock-final.md) is clean. Its original failing case-alias probe and independent sidecar alias, special-file, lock-release, and temporary-file cleanup probes passed. All nine selected top-level tests ran without skips.

The [Spec recheck](05-setup-write-final/spec/spec-setup-recheck-c59d12e.md) is clean. It independently exported `c59d12e` and confirmed that all 321 exported files stayed unchanged. Native macOS race tests and native Linux setup, permission, and CLI suites passed. Same-process and separate-process shared/personal races produced one successful write, one conflict, and valid discovery in both commit orders; matching declarations coexist.

Linux preserved UID 1001, GID 1002, mode 0640, and the exact POSIX ACL. Every one of 200 ACL revocation attempts rejected the stale write and retained the revoked grant, original bytes, and inode. Those attempts included 79 equal-change-time cases before Apply and 58 during Sync. The retained `spec` directory contains commands, outputs, image/binary/source/probe hashes, and the probe sources. Windows and Darwin amd64 were compiled but were not runtime-tested.

## Corrected behavior and limits

Changed setup operations coordinate through the physical personal-registry sidecar. Shared writes also lock their destination. Lock files are ordered and deduplicated by physical identity, so case, symlink, and hard-link aliases cannot reverse the order or lock the same file twice. Sidecars remain in place. A changed shared operation can create the registry's `.context` directory and `config.md.lock` even without a personal configuration. The registry and destination lock locations must be writable; failed writes can leave these lock artifacts but do not replace configuration bytes. Personal setup does not write the checkout. Dry-run and no-op bypass lock creation.

Prepared plans capture exact permission metadata together with document bytes. Apply compares fresh snapshots and restores only the captured permissions. ACL changes are detected directly even when filesystem timestamps coincide. Unreadable or unsupported permissions, protection flags, and security attributes that cannot be retained cause safe refusal before replacement.

Darwin and Linux implement physical lock identity and permission preservation. Windows supports new configurations when its file-ID query is available and refuses existing-file replacement whose permissions cannot be preserved. Other platforms refuse changed setup because stable lock identity is unavailable. No-op and dry-run remain available. Coordination applies to cooperating writers sharing the injected Home; it does not discover another user's unseen registry.

## Acceptance criteria

| Criterion | Result |
| --- | --- |
| Setup flags, guided terminal flow, deterministic noninteractive use, and no reader prompts | Pass. Full CLI tests and independent CLI checks passed. |
| Input validation, source authorization, and shared-relative or personal-absolute paths | Pass. Existing path and resolver suites passed; independent authorization, alias identity, and cwd probes passed. |
| Dry-run, idempotence, replacement, metadata/body/permission preservation | Pass with the platform limits above. Independent probes verified no-write behavior, YAML tags, unrelated alias values, exact body bytes, owner/group/mode, ACL copy/removal, and permission inspection failures. |
| Atomic writes, concurrent-change detection, intact previous configuration | Pass. Shared/personal processes serialize; physical lock ordering handles aliases and opposing homes; exact ACL snapshots reject stale writes. Failure and cleanup probes leave the previous file usable. |
| Required behavior tests | Pass. Full suites cover shared/personal registration, aliases, nested-cwd rebasing, conflicts, dry-run, replacement, concurrent edits, and write failures. New deterministic regressions reproduce and prevent the independent findings. |

## Task context

The current reader was built before correction and again after the documentation merge. Task context and orientation used explicit `--bundle` and `--allow-source` paths for this worktree, `--max-files 500`, and `--max-bytes 8388608`. Context was complete with complete traversal and no diagnostics. All selected source texts and reasons matched the full context already delivered. Orientation was complete with complete inventory, no diagnostics, and all four ticket-05 readiness checks passing. Final acceptance and prerequisite results are retained separately in the closeout observation.
