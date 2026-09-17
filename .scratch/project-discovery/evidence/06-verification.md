# Discovery documentation and end-to-end verification

Observed by workflow actor `Codex /root` at `2026-09-17T01:23:26+02:00`. Tested source revision: `8c3c15f98f4ce8e465cfd7667a3a89034e41e128` from `https://github.com/Zokiio/context`. Documentation and the trial driver were committed before verification. Go sources and fixtures remained unchanged throughout checks and trial. The trial then created the shared repository configuration through the actual setup command; its exact proposal and written contents are retained in the trial results.

## Criterion results

| Criterion | Result and evidence |
| --- | --- |
| 1. Document discovery and migration | Pass. The README provides build and routine-read commands. `docs/discovery.md` covers shared/personal profiles, specificity, conflicts, source roots, diagnostics, setup writes, unavailable-cwd aliases, and workspace output. `docs/connect-projects.md` walks through setup, replacement, workspace authoring, and direct recovery. `docs/readers.md` retains the reader schemas, fingerprint, and acceptance contracts. Both references explain the manifestless `--project` to `--bundle` migration. Root checked local links; independent workflow reviewer `/root/discovery_scope` found no concrete command, contract, preservation, or link defect. |
| 2. Real repository, independent project, and workspace trial | Pass. All 37 scenarios passed against the built binary. The independent project and records have no Git repository. Both projects support routine human/JSON orientation; configured roots supply context; personal project/workspace aliases and explicit project/workspace paths select the intended scope. |
| 3. Recovery and relocated data | Pass. The trial covers missing scope, malformed local and personal configuration, broken nearest mappings without fallback, a moved shared checkout, a missing alias checkout, manifestless bundle access, and direct recovery past invalid configuration. Human copyable member commands and structured argument arrays both select the intended records and roots. |
| 4. Revision-specific evidence and repository checks | Pass. `06-build.json` records commands, outputs, exit statuses, source digests, and the binary digest. Full Go tests, race checks, vet, and CLI build exited 0. `06-trial.json` identifies the same revision/binary, every command, expected and actual status, stdout or structured summaries, stderr, and filesystem observations. |
| 5. Writes and workspace semantics | Pass. Every reader/dry-run compares paths, contents, modes, inodes, and modification times before and after. Only the intended configuration and persistent sidecar change during setup; identical setup changes nothing. Workspace membership stays complete with unavailable records, missing checkouts, and a manifestless member. Selecting the available manifestless member produces a partial project report, demonstrating that availability does not certify project validity or readiness. Work status remains unevaluated in both workspace formats. |

## Commands and observations

```sh
python3 /Users/zoki/code/context/.cache/discovery-stack/verify_build.py /Users/zoki/code/context/.cache/worktrees/discovery-docs /private/tmp/ctx-discovery-stack-z6l8_y0p/ctx /Users/zoki/code/context/.cache/worktrees/discovery-docs/.scratch/project-discovery/evidence/06-build.json
python3 .scratch/project-discovery/trial.py /private/tmp/ctx-discovery-stack-z6l8_y0p/ctx .scratch/project-discovery/evidence/06-build.json .scratch/project-discovery/evidence/06-trial.json
```

Both invocations exited 0. The retained build evidence contains the exact underlying Go commands and outputs; the local capture helper only invokes and records them. The repeatable trial driver is committed in this PR.

The trial fixtures are outside the repository so the repository's new shared binding cannot leak into missing-scope scenarios. Child commands use a disposable personal registry, leaving the caller's registry unused. The binary SHA-256 is `3c3dbaeadac1f69196bed3231e1d9099f10c812b7cbb2a42600ca89f4d393483`.

Repository orientation succeeds with explicit limits of 500 files and 8 MiB. The unchanged default limits produce the expected partial report; this is also a passing trial scenario. Independent fixtures fit within the defaults. No product budget was changed.

The shared `.context/config.md` was produced by setup during the trial and is committed with these records. `.gitignore` retains `/.cache/` and ignores the persistent `.context/config.md.lock`. Documentation link checks and `git diff --check` passed.

## Pickup context and readiness

The reader built at `933d9c7` returned all 11 sources with `complete: true`, `traversalComplete: true`, and no diagnostics. The coordinator received every path and inclusion reason and read all ticket texts. Requirement documents were compared byte-for-byte with their full text delivered earlier in this run; all five were unchanged. Preflight orientation was complete with 108 sources and no diagnostics. Ticket 06 was ready and eligible, with all checks and prerequisite edges passing and tickets 01–05 valid and fresh.

```sh
go build -o .cache/discovery-docs/ctx ./cmd/ctx
.cache/discovery-docs/ctx context --bundle /Users/zoki/code/context/.cache/worktrees/discovery-docs/.scratch/records --ticket project-discovery/issues/06-document-and-trial.md --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-docs --max-files 500 --max-bytes 8388608
.cache/discovery-docs/ctx orient --bundle /Users/zoki/code/context/.cache/worktrees/discovery-docs/.scratch/records --allow-source /Users/zoki/code/context/.cache/worktrees/discovery-docs --max-files 500 --max-bytes 8388608 --json
```

Both reader invocations exited 0. Delivered source identities before claiming ticket 06:

| Source | SHA-256 | Selection |
| --- | --- | --- |
| `.scratch/records/project-discovery/issues/06-document-and-trial.md` | `e9495042fa6007d3ded7a7a73e029c9afb4036b991b261a9e4ce13fd95378a87` | root |
| `.scratch/project-discovery/spec.md` | `d54c182412a9ee711e1b7a91e2a5ea6c4174dbd8567304ff44fa109a65c41d67` | spec |
| `docs/vision.md` | `d8d7fb88699c3310cd5f489d6d44dbf342fe813c372c54485400f6765f5aa2e9` | context |
| `CONTEXT.md` | `a003af0f4845f28fe14191dec210c7ceeabf6a5422fb51e7e0ee7e957d366a55` | context |
| `docs/adr/0004-directory-specific-project-selection.md` | `efd4654de6c787345c341c26a0076fb605214b3d9aeecc3a1ec01f93752cefbf` | context |
| `docs/agents/acceptance.md` | `fbde78ab62733d1bbe091fba6a867f30f50076c819ee3b0783ac4dcd2d7bf654` | context |
| `.scratch/records/project-discovery/issues/03-integrate-cli.md` | `6edb6fb0022ec99ea4b737881dfd23727b475b689609e089cbda9e76fc928075` | blocked_by |
| `.scratch/records/project-discovery/issues/04-workspace-navigation.md` | `53e4c106efbbc79f30a62414bc8c8d4d85439224b2a217f03989b728be0c0776` | blocked_by |
| `.scratch/records/project-discovery/issues/05-connect-records.md` | `47cd0187aafba0cbbc411576944bce8a146379e132e7b1fef6c98a8b9c756bde` | blocked_by |
| `.scratch/records/project-discovery/issues/02-resolve-scope.md` | `384ade6bcb331613f540a4207b5369fe51e0c3bae202827fc74a5a459cb97ed3` | blocked_by |
| `.scratch/records/project-discovery/issues/01-read-configuration.md` | `abf93aecb0627c77cb82664c7b5c4e2cf53c077f5632b917406feed5fdb02578` | blocked_by |

These are local checks and workflow review, with no human approval asserted. A separate closeout records acceptance after linking the new decision.
