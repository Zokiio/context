# Discovery trial after independent review

Observed by workflow actor `Codex /root` on 2026-09-17. Tested program source: `c59d12e6886874a00d42b821be24615a93c95ef6` from `https://github.com/Zokiio/context`. Documentation and trial integration: `f880dc3`. These are local checks and agent reviews, with no human approval asserted.

## Criterion results

| Criterion | Result and retained evidence |
| --- | --- |
| 1. Document discovery and migration | Pass. The README, discovery reference, connection guide, and reader reference cover configuration, selectors, setup, workspace authoring, source roots, diagnostics, and migration from direct `--project` access to `--bundle`. Updated setup documentation explains persistent coordination sidecars, permission snapshots, and platform support. Independent Standards and Spec rechecks found the claims consistent with the implementation. All 17 local documentation links resolve. |
| 2. Repository, independent project, and workspace trial | Pass. All 37 scenarios passed. This repository supports routine orientation and task context without a selector. Independent non-Git records and a separate checkout support configured roots, aliases, explicit selectors, and workspace membership. |
| 3. Recovery and relocated data | Pass. The trial exercises missing scope, malformed local and personal configuration, broken nearest mappings, a moved shared checkout, a missing alias checkout, direct bundle recovery, and workspace member selection through both displayed commands and structured arguments. |
| 4. Revision-specific checks and evidence | Pass. Full uncached Go tests and race checks, vet, package build, and CLI build passed at `c59d12e`. The coordinator compared all 110 program source files with the integrated documentation branch and copied the exact tested binary. The original commands and test revision are retained rather than attributed to a later merge. |
| 5. Write boundaries and workspace semantics | Pass. Reader and dry-run snapshots retain paths, bytes, modes, UID/GID, inodes, and mtimes. Changed setup touches only the intended registration and coordination sidecars; identical setup changes nothing. The existing repository registration was already identical and remained unchanged; independent fixtures exercised real writes. Member availability remains separate from work readiness in both output formats. |

## Reproduction and provenance

```sh
python3 .scratch/project-discovery/trial.py /private/tmp/ctx-discovery-merge-1c25dd9k/ctx .scratch/project-discovery/evidence/06-reassessment-build.json .scratch/project-discovery/evidence/06-reassessment-trial.json
```

Exit status was 0. The [trial record](06-reassessment-trial.json) retains all 37 invocations, statuses, outputs, filesystem observations, script digest, and program source digests. Fixtures were outside the repository and used a disposable personal home. The caller's registry was not used. Default reader budgets remain unchanged; the bounded partial report is an expected passing scenario. Repository orientation uses explicit limits of 500 files and 8 MiB.

The [build provenance](06-reassessment-build.json) reuses the [original uncached checks](06-reassessment-checks.json) after verifying source identity. Binary SHA-256 is `0d5620b44c86342e48559b1f941f5081f3e7194c10370deb18913c0c9bdd0871`. Full command logs are retained in `05-setup-write-final/`. No unchanged test suite was rerun merely to give it a later revision.

The [independent recheck](10-independent-recheck.md) covers the additional concurrency, physical lock ordering, ownership, ACL preservation, and stale-permission cases that the original trial missed. Native macOS and Linux runtime checks pass. Windows and other platforms were not runtime-tested; documented unsupported writes fail safely.

## Task context and closeout

Before this reassessment, the current reader returned all 11 selected ticket and requirement sources with both completeness fields true and no diagnostics. The coordinator received source paths and reasons, read the changed ticket texts, and compared the remaining full texts with previously delivered context. All five selected requirement documents were byte-for-byte unchanged. The setup ticket was still reopened while this trial ran; acceptance closeout waits for its independently verified correction to be recorded.

The README merge preserved the final documentation and corrected selector semantics. All 110 program sources remained identical to `c59d12e`; the subsequent README edit only points to this latest trial. Earlier evidence and acceptance decisions remain unchanged. A separate closeout records fresh acceptance and prerequisite checks after linking the new decisions.
