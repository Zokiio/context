# Workspace navigation verification

Actor: Codex /root/discovery_scope, workflow verification.
Observed: 2026-09-17T01:05:01+02:00.
Tested revision: `659d9ac204b332f90c3f0f4cf891ca67e46358c3` from `https://github.com/Zokiio/context`.
Environment: darwin/arm64, Go 1.27.1. The tracked working tree was clean during final checks.

This slice reuses workspace navigation, rendering, and tests from `baa1a3e` on accepted reader CLI base `5c21482`. The only change to `internal/cli/cli.go` routes selected workspace orientation to navigation. The added regression verifies workspace aliases in both formats when cwd lookup fails. `scope.go` and `alias_cwd_cli_test.go` remain unchanged from ticket 03.

## Verification results

| Command | Exit status | Observation |
| --- | --- | --- |
| `go test ./...` | 0 | All six test packages pass. `cmd/ctx` has no test files. |
| `go test -race ./...` | 0 | All six test packages pass under the race detector. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build -o .cache/ctx ./cmd/ctx` | 0 | Reader binary built from the tested revision. |
| `go test -json ./internal/workspace ./internal/cli -run 'Test(Navigate\|Workspace)'` | 0 | 18 top-level tests and 37 tests including subtests pass. No tests skipped. This run confirms that permission and FIFO cases executed. |
| `git diff --check` | 0 | No whitespace errors before the source commit. |

## Criterion results

| Ticket criterion | Result and evidence |
| --- | --- |
| 1. Compact text and distinct version-1 workspace JSON with unevaluated work status | Pass. `TestWorkspaceRendererPreservesContractAndStableText` checks schema version 1, `kind: workspace-navigation`, workspace identity, `complete`, `workStatus: unevaluated`, member fields, diagnostics, and absence of project-inventory fields. Human output explicitly states that work status was not evaluated. CLI tests exercise both formats. |
| 2. Deterministic authored members without readiness evaluation or identity merging | Pass. `TestNavigateKeepsMembersAndRootsWithoutReadingRecords` preserves order and member-specific settings despite invalid manifests and unavailable checkouts. `TestNavigateKeepsDistinctRecordStoresWithSameProjectID` retains separate stores and checkouts. `TestNavigateDoesNotOpenMemberFIFOManifest` proves navigation does not open a manifest. Stable renderer output keeps authored order. |
| 3. Shallow availability and safe member commands with explicit roots | Pass. Availability tests cover accessible directories, absent paths, regular files, broken authored traversal, dangling symlinks, loops, FIFOs, denied access, and unexpected errors. Results distinguish available, unavailable, and unknown with diagnostics. The POSIX shell test round-trips apostrophes, quotes, newlines, substitutions, and other metacharacters without executing a marker command. CLI tests verify argument arrays and copied `--bundle` commands carry only the selected member roots, without invocation or sibling roots. |
| 4. Successful membership enumeration despite unavailable members, with invalid scope and project-only errors | Pass. Navigation remains complete and the CLI returns 0 with unavailable records. Empty workspaces succeed. Invalid and conflicting declarations return 2 with no successful report. `TestWorkspaceScopeSelectionNeverInvokesProjectReaders` checks cwd, path, and alias selection. Task context returns project-selection guidance and never chooses a sole member. `TestWorkspaceCLIAliasWithoutWorkingDirectory` verifies both navigation formats succeed when cwd lookup fails. |
| 5. Required behavior coverage | Pass. The 18 workspace/navigation tests cover empty membership, missing records, available records with absent checkout, conflicting declarations sharing an ID, shell escaping, stable text, output failures, cancellation, and unchanged configuration. Repository suites retain ticket-relative semantics, direct bundle access, selector validation, source authorization, and project report behavior. |

## Task context and readiness

Before pickup, the reader built from `5c21482` delivered all nine selected source texts, paths, and reasons. Orientation inspected 104 sources with complete evaluation and inventory, no diagnostics, ticket 04 ready and eligible, and passing dependencies. Tickets 01, 02, and 03 had valid acceptance.

The following command was repeated using the reader built from the tested revision. It exited 0 with `complete: true`, `traversalComplete: true`, and no diagnostics. The changed ticket was reread in full, and all other selected hashes matched the complete preflight delivery.

Working directory: `/Users/zoki/code/context/.cache/worktrees/discovery-workspace`.

```sh
.cache/ctx context \
  --bundle /Users/zoki/code/context/.cache/worktrees/discovery-workspace/.scratch/records \
  --ticket project-discovery/issues/04-workspace-navigation.md \
  --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-workspace \
  --max-files 500 --max-bytes 8388608
```

Delivered source digests use paths relative to that working directory. They identify the observed ticket state before checking the acceptance checklist.

| Source | SHA-256 | Selection |
| --- | --- | --- |
| `.scratch/records/project-discovery/issues/04-workspace-navigation.md` | `4c00aabf9d985ad608118bffce88d13d6d3f1ab2ebef936b02404d74bc5073c5` | root |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` | spec |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` | context |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` | context |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` | context |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` | context |
| `.scratch/records/project-discovery/issues/03-integrate-cli.md` | `6edb6fb0022ec99ea4b737881dfd23727b475b689609e089cbda9e76fc928075` | blocked_by |
| `.scratch/records/project-discovery/issues/02-resolve-scope.md` | `384ade6bcb331613f540a4207b5369fe51e0c3bae202827fc74a5a459cb97ed3` | blocked_by |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `abf93aecb0627c77cb82664c7b5c4e2cf53c077f5632b917406feed5fdb02578` | blocked_by |

Development outcome: all five criteria passed at the tested revision. Workspace navigation reads selected membership and directory access only. Setup remains outside this slice.
