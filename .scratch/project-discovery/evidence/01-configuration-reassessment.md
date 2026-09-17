# Configuration reassessment

Observed by workflow actor `Codex /root` at `2026-09-17T00:44:26+02:00`. Tested repository revision: `2f4e95a5f172d024aac100d14c56e8a6c02127db` from `https://github.com/Zokiio/context`. All source and test files were committed before these checks. Later changes only settle workflow records.

This reassessment supersedes the earlier configuration decision while preserving its original evidence. The configuration and physical-path core comes from commit `baa1a3ee87aad223cb0fc089744dfe6987054cd9`; this revision also fixes nested YAML alias-key duplicates in the shared Markdown parser. The source worktree was read without modification.

## Criteria results

1. Pass. `TestReadSharedConfigPreservesDocumentAndResolvesSeparateRecords`, `TestReadPersonalConfigAllowsUnavailableRegistrations`, `TestPersonalArraysDefaultToEmpty`, and `TestConfigFieldDiagnostics` cover both profiles, projects, aliases, workspaces, members, and field attribution.
2. Pass. `TestConfigRejectsMalformedAndUnsupportedDocuments`, `TestConfigRejectsDuplicateDeclarationKeysAndAliases`, and the field-diagnostic cases enforce types, required identities, profiles, and versions. `TestParseConfigKeepsItsOwnSourceSnapshot` preserves original source. `TestParseDocumentRejectsResolvedDuplicateKeys` rejects nested and sequence alias duplicates, numeric equivalents, and boolean equivalents. Positive parser tests retain unique alias keys, merge overrides, anchor order, unknown metadata, and Markdown body bytes.
3. Pass. `TestConfigSymlinkKeepsEncounteredRelativeBase` and `TestSavedPathsStayLiteralAndPreserveSymlinkParentTraversal` resolve saved values from the encountered file and traverse symlinks before parent components. Canonical-path and SamePath tests cover missing suffixes, symlink identity, loops, and non-directory traversal. `TestConfigDefersUnresolvableTargetIdentity` retains field failures without rejecting an unrelated registration. Availability of the selected directory remains the resolver's responsibility.
4. Pass. External-package tests use real temporary filesystems for malformed YAML, relative paths, both profiles, duplicate declarations, separate records, source preservation, and filesystem errors. Full tests also exercise existing direct readers after the shared parser change.

## Task context and acceptance observation

The current reader was rebuilt at the tested revision. Every returned source's full text, path, and inclusion reasons was delivered to the coordinator before reassessment. Context exited 0 with `complete: true`, `traversalComplete: true`, and no diagnostics. Fingerprint orientation exited 0 with complete inventory and no diagnostics; the subject's four readiness checks passed.

```sh
go build -o .cache/discovery-stack/01/ctx ./cmd/ctx
.cache/discovery-stack/01/ctx context --project /Users/zoki/code/context/.scratch/records --ticket project-discovery/issues/01-read-configuration.md --allow-source /Users/zoki/code/context --max-files 500 --max-bytes 8388608
.cache/discovery-stack/01/ctx orient --project /Users/zoki/code/context/.scratch/records --allow-source /Users/zoki/code/context --max-files 500 --max-bytes 8388608 --json
```

Verification explicitly raises collection budgets for the growing historical evidence; product defaults are unchanged. Delivered sources:

| Source | SHA-256 |
| --- | --- |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `6119ffc186a584cc6b05ae756d0166be66c4f76f973df8436e484d288f2b96ea` |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` |

## Checks

```text
$ go test ./...
?   	github.com/Zokiio/context/cmd/ctx	[no test files]
ok  	github.com/Zokiio/context/internal/cli	0.575s
ok  	github.com/Zokiio/context/internal/discovery	(cached)
ok  	github.com/Zokiio/context/internal/orientation	0.652s
ok  	github.com/Zokiio/context/internal/recordread	(cached)
ok  	github.com/Zokiio/context/internal/taskcontext	0.449s
exit: 0
```

```text
$ go test -race ./...
?   	github.com/Zokiio/context/cmd/ctx	[no test files]
ok  	github.com/Zokiio/context/internal/cli	5.600s
ok  	github.com/Zokiio/context/internal/discovery	1.334s
ok  	github.com/Zokiio/context/internal/orientation	2.092s
ok  	github.com/Zokiio/context/internal/recordread	1.229s
ok  	github.com/Zokiio/context/internal/taskcontext	1.439s
exit: 0
```

```text
$ go vet ./...
exit: 0
```

The CLI build and `git diff --check` also exited 0. These are local workflow observations, with no human approval asserted.
