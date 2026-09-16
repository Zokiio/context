# Reader CLI verification

Actor: Codex agent `/root/discovery_scope_cases`.
Observed at: `2026-09-17T00:59:40+02:00`.
Tested source: `feat/discovery-reader-cli` commit `72dd8d7c203dbc14a45dab0f7347771246c5cb29`.
Working directory: `/Users/zoki/code/context/.cache/worktrees/discovery-reader-cli`.

The source commit reuses the reader integration from `01791c2` and the later input guards from `baa1a3e`. It adds the confirmed cwd-independent alias fix. The accepted resolver and record parser remain unchanged.

All ticket 03 criteria passed:

| Criterion | Observed evidence |
| --- | --- |
| Optional scope and consistent selectors | `TestReaderDiscoveryPassesSameExplicitScope` exercises cwd, directory, and alias selection through both readers. `TestWorkspaceScopeSelectionNeverInvokesProjectReaders` retains explicit workspace selection and project-selection guidance. |
| Direct bundle compatibility | `TestBundleBypassesHomeLookupAndDiscovery` reads a manifestless ticket while malformed configuration and unavailable home lookup are bypassed. Direct orientation retains its partial report and exit 1. |
| Reader contracts and invocation roots | Scope/limit tests and existing CLI scripts preserve ticket-relative paths, project JSON, budgets, and exit statuses. `TestDiscoveredReadsPreserveFilesAndInvocationRootsStayTemporary` proves configured roots and invocation roots remain separate and verifies unchanged file bytes, modes, and modification times. |
| Failures and explanations | `TestReadCommandsRejectInvalidSelectorsBeforeDiscovery` rejects empty, repeated, and conflicting selectors. `TestDiscoveryFailuresDoNotEmitReportsOrFallBack` covers missing scope, malformed configuration, broken selected targets, and conflicts. Explained reads preserve stdout JSON exactly and emit provenance only on stderr. |
| Migration and read-only CLI trials | Existing CLI scripts use `--bundle` for direct records. Physical-path tests retain symlink-before-parent traversal and reject unusable original spellings. Reader-only FIFO tests reject special paths without waiting for a writer. |

`TestReaderAliasesWorkWhenCwdLookupFails` covers project aliases in both readers, an unavailable checkout, configured roots, optional absolute invocation roots, and workspace alias selection. `TestReadCommandsKeepCwdErrorsWhenPathsNeedCwd` verifies that relative invocation roots and all ordinary directory selectors retain the original cwd diagnostic. No home or filesystem-root substitute is used.

Workspace orientation rendering belongs to ticket 04. In this slice, selecting a workspace returns guidance without invoking either project reader or choosing a member. Setup is not registered.

Verification commands ran against the committed source:

| Command | Exit status | Duration in seconds |
| --- | --- | --- |
| `go test ./...` | 0 | 1.433 |
| `go test -race ./...` | 0 | 7.232 |
| `go vet ./...` | 0 | 0.359 |
| `go build ./...` | 0 | 0.578 |
| `go build -o .cache/discovery-reader-cli/ctx ./cmd/ctx` | 0 | Not measured |

The test and race suites passed for `internal/cli`, `internal/discovery`, `internal/orientation`, `internal/recordread`, and `internal/taskcontext`. Vet and build emitted no diagnostics. The working tree was clean after verification.

The alias regression was reproduced before the fix with `go test ./internal/cli -run TestReaderAliasesWorkWhenCwdLookupFails|TestReadCommandsKeepCwdErrorsWhenPathsNeedCwd -count=1`, exit 1. Both readers and workspace selection returned `read working directory: cwd was removed` before consulting the saved alias. The final full suites pass the same test.

The reproduction source equals the tested commit except that `internal/cli/scope.go` used `baa1a3e:internal/cli/scope.go`. Its source digest manifest is:

| File | SHA-256 |
| --- | --- |
| `internal/cli/scope.go` before the fix | `baa585f763a73523f5a8f16fd40491d7b8f96821cf943a18dcc585a2655082f1` |
| `internal/cli/alias_cwd_cli_test.go` | `04ba125dcabfd0ca2c4b835cbdba370799872ffa3f76696af51eab2c3921d543` |

Task context and pickup readiness used reader commit `39483dd` through the current source tree before CLI migration. The exact revision is `39483dd87481eaf039f6c2a81dfcaa7df59a4136`.

```text
.cache/discovery-reader-cli/ctx-before context --project /Users/zoki/code/context/.cache/worktrees/discovery-reader-cli/.scratch/records --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-reader-cli --max-files 500 --max-bytes 8388608 --ticket project-discovery/issues/03-integrate-cli.md
.cache/discovery-reader-cli/ctx-before orient --project /Users/zoki/code/context/.cache/worktrees/discovery-reader-cli/.scratch/records --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-reader-cli --max-files 500 --max-bytes 8388608 --json
```

Both commands returned 0 without diagnostics. Task context reported `complete: true` and `traversalComplete: true`. All eight returned source texts, paths, and inclusion reasons were delivered to the implementing agent before edits. Orientation reported `complete: true`, `inventoryComplete: true`, ticket 03 ready and eligible, and passing prerequisite edges through tickets 02 and 01.

Delivered source identities follow. Digests cover the exact source text received before claiming ticket 03.

| Source | Inclusion reason | SHA-256 |
| --- | --- | --- |
| `.scratch/records/project-discovery/issues/03-integrate-cli.md` | root | `5f30678e38ea3e6a4860121da4623bcee6b3cbc4e0b232fe3cb1fe093b6cfb14` |
| `.scratch/project-discovery/spec.md` | spec | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` |
| `docs/vision.md` | context | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` |
| `CONTEXT.md` | context | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` |
| `docs/adr/0004-directory-specific-project-selection.md` | context | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` |
| `docs/agents/acceptance.md` | context | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` |
| `.scratch/records/project-discovery/issues/02-resolve-scope.md` | blocked_by | `384ade6bcb331613f540a4207b5369fe51e0c3bae202827fc74a5a459cb97ed3` |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | blocked_by | `abf93aecb0627c77cb82664c7b5c4e2cf53c077f5632b917406feed5fdb02578` |

Acceptance and closeout use the rebuilt reader with `--bundle`, `--allow-source` set to this worktree, `--max-files 500`, and `--max-bytes 8388608`. This document is immutable verification evidence. The separate closeout observation records acceptance freshness after the ticket link is written.
