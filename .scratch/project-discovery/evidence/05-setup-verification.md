# Setup verification

Actor: Codex /root/configuration_pr_review, workflow verification.
Observed: 2026-09-16T23:15:13Z.
Tested revision: `5aee51f7fc0079c330907f9e4a62e447d6808b21` from `https://github.com/Zokiio/context`.
Environment: darwin/arm64, Go 1.27.1. The tracked working tree was clean during final checks.

This slice adds setup preparation, atomic application, and CLI registration on accepted workspace base `573c220`. It reuses the reviewed implementation from `baa1a3e`, with corrections for preservation of unrelated aliases, tagged metadata, and merge precedence. Writer locking and concurrent-change checks remain unchanged. Reader scope and workspace changes from tickets 03 and 04 remain in the base.

## Verification results

| Command | Exit status | Observation |
| --- | --- | --- |
| `go test -count=1 ./...` | 0 | All six test packages passed. `cmd/ctx` has no test files. |
| `go test -race -count=1 ./...` | 0 | All six test packages passed under the race detector. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build ./...` | 0 | All packages built. |
| `go build -o /tmp/ctx-ticket05-review.IpDutn/ctx-setup ./cmd/ctx` | 0 | Built the reader used for task context and acceptance. |
| `git diff --check` | 0 | No whitespace errors. |

Full test output:

```text
?   	github.com/Zokiio/context/cmd/ctx	[no test files]
ok  	github.com/Zokiio/context/internal/cli	0.802s
ok  	github.com/Zokiio/context/internal/discovery	0.794s
ok  	github.com/Zokiio/context/internal/orientation	0.775s
ok  	github.com/Zokiio/context/internal/recordread	0.123s
ok  	github.com/Zokiio/context/internal/taskcontext	0.333s
ok  	github.com/Zokiio/context/internal/workspace	0.286s
```

Race test output:

```text
?   	github.com/Zokiio/context/cmd/ctx	[no test files]
ok  	github.com/Zokiio/context/internal/cli	5.918s
ok  	github.com/Zokiio/context/internal/discovery	1.911s
ok  	github.com/Zokiio/context/internal/orientation	2.218s
ok  	github.com/Zokiio/context/internal/recordread	1.229s
ok  	github.com/Zokiio/context/internal/taskcontext	1.657s
ok  	github.com/Zokiio/context/internal/workspace	1.449s
```

## Criterion results

| Ticket criterion | Result and evidence |
| --- | --- |
| 1. Flags, guided terminal flow, deterministic noninteractive operation, and no reader prompts | Pass. CLI tests cover exact dry-run output, summary before writing, guided personal registration, nonterminal rejection of missing records, invalid flags before prompts, EOF, and invalid guided answers. Fully specified setup never inspects terminal state. `TestSetupCommandConnectsSubsequentReaderInvocation` runs setup and then discovers its records through the actual reader command with forbidden input. |
| 2. Validate inputs and generate paths from invocation cwd | Pass. `TestSetupSharedRebasesNestedInputsAndPreparationNeverWrites` verifies shared paths from a nested cwd and deduplicated authorized roots. Personal tests verify absolute paths, UUID identity, canonical-directory matching, and retained keys. Invalid records, directory, roots, malformed configuration, and reserved home destinations fail. Symlinked configuration keeps the encountered path base and the symlink. FIFO command cases cover records, directory, roots, both configuration profiles, and project markers. |
| 3. Dry-run, idempotence, replacement, and preservation | Pass. Tests verify exact dry-run bytes without creating configuration, unchanged setup without a lock or file rewrite, explicit replacement, metadata, CRLF body bytes, and file mode preservation. Personal aliases retain identity and reject collisions or ambiguous directories. Preservation regressions cover shared mapping and field aliases, workspace references, personal entry and array aliases, anchors redefined in declaration order, implicit-null anchors, nested scalar alias keys, tagged sequences and mappings, binary, timestamp, custom tags, NaN, and merged fields. |
| 4. Atomic writes, concurrent-change detection, and intact previous files on failure | Pass. `TestSetupApplyDetectsConcurrentDestinationChanges`, `TestSetupRechecksChangesWhileTemporaryFileIsWritten`, and `TestSetupSerializesCooperatingWriters` verify destination rechecks and coordinated writers. Injected temporary-file and rename failures leave the previous usable file intact. CLI output failures prevent all writes. Records are validated before any directory creation, and malformed configuration is never replaced. Conflicting declarations in other files remain errors even with `--replace`. |
| 5. Required behavior coverage | Pass. Core and CLI tests cover shared and personal registration, aliases, nested-cwd rebasing, conflicts, dry-run, replacement, concurrent edits, and write failures. The full repository suite retains selector migration, source authorization, task-context, project orientation, and workspace behavior. |

Setup edits the selected fields in the YAML syntax tree, retaining unknown tags. Aliases are materialized independently before editing and resolved in declaration order. Managed fields follow merge entries so the parser selects the requested values. A parsed-proposal check rejects any unexpected metadata change, and a resolution check confirms the requested directory, records, and effective roots.

An unchanged configuration bypasses rewriting and retains its exact bytes, including recursive aliases. A changed document with recursive or unresolved aliases fails before writing rather than changing unknown meaning. `TestSetupLeavesRecursiveMetadataUnchangedOrRefusesRewrite` verifies both cases.

## Task context and readiness

The initial reader at accepted resolver revision `39483dd` delivered all eight selected source texts, paths, and reasons. Ticket 05 was ready and eligible, with passing prerequisite checks. The agent claimed the ticket before implementing setup core.

After rebasing onto accepted workspace revision `573c220`, the current reader delivered the same complete requirements and passing checks. Ticket 05 remained ready and was excluded from new pickup only because its execution state was already in progress. Tickets 01 through 04 retained valid, fresh acceptance.

Working directory: `/Users/zoki/code/context/.cache/worktrees/discovery-setup`.

The following commands used the binary built from the tested revision:

```sh
/tmp/ctx-ticket05-review.IpDutn/ctx-setup context \
  --bundle /Users/zoki/code/context/.cache/worktrees/discovery-setup/.scratch/records \
  --ticket project-discovery/issues/05-connect-records.md \
  --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-setup \
  --max-files 500 --max-bytes 8388608

/tmp/ctx-ticket05-review.IpDutn/ctx-setup orient --json \
  --bundle /Users/zoki/code/context/.cache/worktrees/discovery-setup/.scratch/records \
  --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-setup \
  --max-files 500 --max-bytes 8388608
```

Both exited 0 without diagnostics. Task context had `complete: true` and `traversalComplete: true`; orientation had `complete: true` and `inventoryComplete: true`. All four ticket-05 readiness checks passed. The final context source texts and reasons matched the full post-rebase delivery byte for byte.

Delivered source digests use paths relative to the working directory. They identify the state before completing the acceptance checklist.

| Source | SHA-256 | Selection |
| --- | --- | --- |
| `.scratch/records/project-discovery/issues/05-connect-records.md` | `bf17aae4ac566c1489ff13c01d33b20ad12672d7b74910c0a5e9c760178a304f` | root |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` | spec |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` | context |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` | context |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` | context |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` | context |
| `.scratch/records/project-discovery/issues/02-resolve-scope.md` | `384ade6bcb331613f540a4207b5369fe51e0c3bae202827fc74a5a459cb97ed3` | blocked_by |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `abf93aecb0627c77cb82664c7b5c4e2cf53c077f5632b917406feed5fdb02578` | blocked_by |

Development outcome: all five ticket-05 criteria passed at the tested revision. Migration documentation and the independent-project/workspace trial remain ticket 06.
