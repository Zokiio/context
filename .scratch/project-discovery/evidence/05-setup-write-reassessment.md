# Setup write reassessment

Actor: Codex /root/configuration_pr_review, coordinated by /root, with the permissions helper /root/setup_permissions_fix.
Observed: 2026-09-17T11:38:58+02:00.
Tested revision: `a0ecc4cf143886e687c072ceffc9c8d1a0963a86` from `https://github.com/Zokiio/context`.
Environment: Go 1.27.1 on darwin/arm64, effective UID 501. The tracked working tree was clean during final checks.

This is new verification after independent review found that two shared/personal setup writers could both succeed and leave conflicting declarations, and that replacement lost owner/group or ACL information. The previous setup decision and evidence remain unchanged.

## Reproduction and correction

The original race probe and the new deterministic `TestSetupSerializesSharedAndPersonalProcesses` reproduced two successful writes followed by a discovery conflict. The new test failed in both writer orders before the coordination change. It uses separate processes, real filesystem locks, symlinked Home, cwd-relative inputs, and explicit rename barriers without timing sleeps. After the change, exactly one writer succeeds; the second reports a conflict and discovery selects the winning binding.

The first coordination design could deadlock when two different Home values made each shared destination the other registry. `TestSetupOrdersLocksAcrossDifferentHomes` reproduced both writers holding the other's next lock. Ordering the locks fixes that cycle. Cooperating setup writers retain the registry lock and the shared destination lock through revalidation and rename. Lock ordering avoids cycles; it does not discover another user's unseen registry or create a cross-user conflict policy.

Changed shared setup can now create the personal registry's `.context` directory and `config.md.lock`, plus its own destination sidecar. It does not create a personal `config.md`. These lock locations must be writable. Locks are retained, and a failed attempt can leave lock directories or sidecars, but does not replace target configuration bytes. No-op and dry-run bypass locking; personal setup does not write the checkout, including when its mode is 0555.

Permission preservation runs before sync and rename. It preserves owner, group, mode, and supported ACLs, verifies the replacement, and refuses a write if those permissions cannot be preserved. Snapshot checks now include ownership and change time so group- or ACL-only edits invalidate a prepared plan. An Apply-level regression changes then restores mode, leaving mode, content, size, and mtime unchanged; the stale plan is rejected.

## Verification results

| Command | Exit | Observation |
| --- | --- | --- |
| `go test -json -count=1 ./...` | 0 | All six test packages passed without cached execution. |
| `go test -race -json -count=1 ./...` | 0 | All six test packages passed under the race detector. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build ./...` | 0 | All packages built. |
| `go build -o .cache/setup-race/ctx ./cmd/ctx` | 0 | Built the reader used for this reassessment. |
| `GOOS=linux GOARCH=amd64 go test -c -o .cache/setup-race/discovery-linux.test ./internal/discovery` | 0 | Linux test binary cross-compiled. |
| `GOOS=windows GOARCH=amd64 go test -c -o .cache/setup-race/discovery-windows.test.exe ./internal/discovery` | 0 | Windows test binary cross-compiled. |
| `GOOS=darwin GOARCH=amd64 go test -c -o .cache/setup-race/discovery-darwin-amd64.test ./internal/discovery` | 0 | Darwin amd64 test binary cross-compiled. |
| `git diff --check` | 0 | No whitespace errors. |

The native test and race runs skipped `TestPreserveSetupPermissionsRetainsDifferentOwner` because changing to a different UID requires root. Current-owner preservation, a different permitted group, special mode bits, ACL copying/removal, and permission-only change detection ran successfully. `TestSetupConcurrentProcessWriter` is intentionally skipped as a top-level helper; the process regressions execute it in child processes.

Linux and Windows runtime tests were not run. Linux preserves POSIX access ACLs and refuses mismatched security, system, or trusted attributes that it cannot preserve. Darwin refuses unsupported protection flags or unreadable/unsupported ACL data before rename. Other platforms permit new-file creation and no-op setup but refuse replacement of existing files until their permissions can be preserved safely.

## Criterion reassessment

| Criterion | Result |
| --- | --- |
| 1. Setup flags, guided terminal flow, deterministic noninteractive use, no reader prompts | Pass. Full CLI tests cover guided input, flag errors, exact dry-run output, summary-before-write behavior, output failure, and subsequent read commands without prompts. |
| 2. Validate inputs and produce shared relative or personal absolute paths | Pass. Full core tests retain manifest/directory/root validation, nested-cwd rebasing, symlink semantics, canonical personal matching, reserved-profile checks, and FIFO rejection. The process regression also exercises relative inputs and a symlinked Home. |
| 3. Dry-run, no-op, replacement, and preservation | Pass with the platform limits above. Existing YAML alias/tag/merge, exact body, and identity tests pass. New tests verify owner/group/mode/ACL preservation, inherited-ACL removal, both lock footprints, no-op/dry-run without Home writes, and personal setup without checkout writes. |
| 4. Atomic writes, concurrent-change detection, intact previous configuration | Pass. Cross-process shared/personal writes serialize; opposing lock orders cannot deadlock. Destination edits and permission-only changes reject stale plans. Injected permission, temporary-file, sync, close, and rename failures leave previous bytes usable. Registry-lock failure does not write the target. |
| 5. Required behavior tests | Pass. The full suite covers shared/personal registration, aliases, rebasing, conflicts, dry-run, replacement, concurrent edits, and write failures, plus the independently reported races and permission regressions. |

## Task context and source identity

Before correction, the current reader at `933d9c7` delivered complete task context and all ticket-05 prerequisite checks passed. The ticket was reopened under the existing coordinated claim. The final reader from the tested revision repeated these commands in `/Users/zoki/code/context/.cache/worktrees/discovery-setup`:

```sh
.cache/setup-race/ctx context --bundle /Users/zoki/code/context/.cache/worktrees/discovery-setup/.scratch/records --ticket project-discovery/issues/05-connect-records.md --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-setup --max-files 500 --max-bytes 8388608
.cache/setup-race/ctx orient --json --bundle /Users/zoki/code/context/.cache/worktrees/discovery-setup/.scratch/records --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-setup --max-files 500 --max-bytes 8388608
```

Both exited 0 with no diagnostics. Context had `complete` and `traversalComplete` true. Orientation had `complete` and `inventoryComplete` true, and all ticket-05 readiness checks passed. The changed ticket text was read in full; all other selected texts and reasons matched the full context already delivered.

Delivered sources before completing the reassessment checklist:

| Source | SHA-256 | Selection |
| --- | --- | --- |
| `.scratch/records/project-discovery/issues/05-connect-records.md` | `43fcdf1f9ed915ba15939d1f3a1d308d1f27c15da57965481649a8a3b4f5629d` | root |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` | spec |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` | context |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` | context |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` | context |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` | context |
| `.scratch/records/project-discovery/issues/02-resolve-scope.md` | `384ade6bcb331613f540a4207b5369fe51e0c3bae202827fc74a5a459cb97ed3` | blocked_by |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `abf93aecb0627c77cb82664c7b5c4e2cf53c077f5632b917406feed5fdb02578` | blocked_by |

The committed implementation has these source digests:

| Source | SHA-256 |
| --- | --- |
| `go.mod` | `9eef81ff328e077a4450046dc2f8c28c995056083fdd392da7af97662b2c5239` |
| `internal/discovery/setup_write.go` | `7e13d8daab8fab94d49501b00fbee7052af151d3fb5f453a107d6f80e74bb8b3` |
| `internal/discovery/setup_permissions_unix.go` | `86d45b02cbe225dd142036451808fef5616bd817d73ad7fc7ae6fccd9a3ce690` |
| `internal/discovery/setup_permissions_darwin.go` | `328c4a8f1083e40225b08a513142fac3f3c5bf70251b8723319fc38f651b2276` |
| `internal/discovery/setup_permissions_linux.go` | `76a0752e3c7358234277b8b941bfb14775529c4105eeb46a6b85bc8f355055a9` |
| `internal/discovery/setup_permissions_other.go` | `74fdb3f04ffb6c98eb0c845df89b7db932a714d1e8bf4d35c0ec13974ecb0a72` |

Development outcome: all five ticket-05 criteria passed at the tested revision with the runtime limits recorded above. Documentation and the final independent-project/workspace trial remain coordinated under ticket 06.
