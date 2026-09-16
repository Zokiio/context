# Scope resolution verification

Actor: Codex /root/discovery_scope, workflow verification.
Observed: 2026-09-17T00:49:03+02:00.
Tested revision: `fbfcacd0c6104ed4810c23973d5c75f3ad61f81b` from `https://github.com/Zokiio/context`.
Environment: darwin/arm64, Go 1.27.1. The tracked working tree was clean during final checks.

The resolver, comparison code, and original resolver tests were reused from `baa1a3e`. The accepted configuration base is `18728ee`. Commit `668104c` adds that resolver and boundary tests. Commit `fbfcacd` allows explicit personal aliases when the caller cannot supply cwd. Configuration uses one parser in `internal/discovery`.

## Verification results

| Command | Exit status | Observation |
| --- | --- | --- |
| `go test ./...` | 0 | All five test packages pass. `cmd/ctx` has no test files. |
| `go test -race ./...` | 0 | All five test packages pass under the race detector. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build -o .cache/ctx ./cmd/ctx` | 0 | Reader binary built from the tested revision. |
| `go test -json ./internal/discovery` | 0 | 55 top-level tests and 160 tests including subtests pass. No tests skipped. This run confirms that filesystem-specific cases executed. |
| `git diff --check` | 0 | No whitespace errors before the source commits. |

## Criterion results

| Ticket criterion | Result and evidence |
| --- | --- |
| 1. One testable, read-only discovery operation with scope and provenance | Pass. `Resolve(context.Context, Request)` accepts cwd, home, kind, and selector explicitly. Temporary filesystem tests exercise it without ambient environment setup. `TestResolveLeavesFilesAndProcessDirectoryUnchanged` verifies unchanged configuration and process cwd. Coalescing tests retain both origins. The operation returns values and errors without printing or writing. |
| 2. Depth, kind, conflict, malformed-candidate, and stopping rules | Pass. `TestResolveNearestDirectoryAcrossStorageLocations`, `TestResolveWorkspaceStopsImplicitProjectFallback`, and `TestResolveProjectWinsAtEqualDepthBeforeWorkspaceConflicts` cover specificity and kind selection. Conflict tests cover records, root sets, and workspace declarations. Registry, malformed local/marker, and stopping-rule tests verify relevant failures and ignored broader files. `TestResolveWorkspaceIgnoresBrokenProjectTargets` verifies explicit filtering before target validation. |
| 3. Physical ancestry, canonical comparisons, diagnostic spelling, and no fallback | Pass. The conformance tests follow symlinks before parent traversal, ignore lexical ancestors, compare physical identity on this case-insensitive filesystem, cross home and Git boundaries, and stop at filesystem root. `TestResolveBrokenSelectedMappingNeverFallsBack`, `TestResolveSelectedBindingChecksOriginalSpelling`, and `TestResolveSelectedRootsAndAliasRecordsKeepOriginalSpelling` reject missing targets and `missing/../` spellings without fallback. Errors retain the start, file, field, authored value, resolved target, and repair guidance. |
| 4. Aliases independent of cwd and checkout availability, with unrelated targets ignored | Pass. `TestResolveAliasesIgnoreCheckoutAndCwdAvailability` covers missing cwd and checkout paths. `TestResolveAliasesWithoutWorkingDirectory` covers empty cwd for explicit project and workspace aliases, without substituting process cwd. Relative nonempty cwd and missing absolute home remain errors. `TestResolveUnrelatedRecordTargetsRemainLazy` covers unrelated symlink loops. Non-directory bindings are ignored when they cannot contain the start. Unresolved permission-hidden bindings produce diagnostics. |
| 5. Required behavior coverage | Pass. The resolver and public conformance suites cover nested scopes, explicit kind filtering, same-depth roots conflicts, symlinks, path component boundaries, malformed registries and local files, agreeing markers and mappings, and the reserved personal registry at physical or symlinked home. Workspace member tests preserve order, keep member roots separate, accept unavailable member targets, and prove members create no cwd bindings. Proposal tests validate the read-only replacement seam used by later setup work. |

## Task context and readiness

Before pickup, the reader built from `18728ee` returned complete task context with all seven source texts, paths, and reasons. Orientation inspected 100 sources with `complete: true`, `inventoryComplete: true`, no diagnostics, ticket 02 ready and eligible, and a passing dependency edge to ticket 01 with valid acceptance.

The following context command was repeated using the reader built from the tested revision. It exited 0 with `complete: true`, `traversalComplete: true`, and no diagnostics. Every selected source was read in full. Unchanged source hashes were compared with the previous delivery; the changed ticket text was reread.

Working directory: `/Users/zoki/code/context/.cache/worktrees/discovery-scope`.

```sh
.cache/ctx context \
  --project /Users/zoki/code/context/.cache/worktrees/discovery-scope/.scratch/records \
  --ticket project-discovery/issues/02-resolve-scope.md \
  --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-scope \
  --max-files 500 --max-bytes 8388608
```

Delivered source digests use paths relative to that working directory. These identify the observed ticket state before the acceptance checklist was checked.

| Source | SHA-256 | Selection |
| --- | --- | --- |
| `.scratch/records/project-discovery/issues/02-resolve-scope.md` | `9dc9241200485750696ac1c145d9e5ffdc2bd4e04e940eec15e6474eec622c0a` | root |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` | spec |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` | context |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` | context |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` | context |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` | context |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `abf93aecb0627c77cb82664c7b5c4e2cf53c077f5632b917406feed5fdb02578` | blocked_by |

Development outcome: all five criteria passed at the tested source revision. Ticket 02 is ready for recorded acceptance. CLI integration, workspace navigation output, and setup writes belong to later tickets. Existing reader selection and completeness behavior passed the unchanged repository suites.
