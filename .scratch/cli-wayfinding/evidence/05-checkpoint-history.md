# Checkpoint history verification

Actor: Codex /root. Date: 2026-09-19.
Tested revision: `25ef67077e0acb000263341344e455e7885eedef`.
Agent implementation revision: `1c500a0d7f10c31d64c8f604b816d93fe642a12e`, with identical product files after integration.

## Results by criterion

1. Graph evaluation retains parsed observations and predecessor order, sorts observation IDs, and selects leaves only after complete valid inspection. Tests cover a linear history, several roots, duplicate/mismatched identities, self-links, and cycles without a timestamp selection rule.
2. Actual CLI probes preserve both competing successors and their separate comparisons. An explicit successor referencing both accounts becomes the unique leaf even with an older timestamp. Complete conflicts return exit 0.
3. Dangling predecessors and cycles return invalid graphs; unfinished publications and bounded inspection return incomplete graphs. Candidates stay empty. A fixture containing both known invalidity and an unfinished write gives invalid precedence while retaining parsed notes.
4. Superseded snapshots stay not_loaded. Removing an older snapshot does not make recovery partial. Corrupting a candidate snapshot leaves its graph available but produces a partial comparison and exit 1.
5. Tests exercise discovery, metadata, and snapshot budget boundaries. Collection stops after the first breach and retains omitted-path findings. Discovery sorts only the observed subset. Quarantine remains outside discovery.
6. Earlier multi-note restrictions are removed. Five contract fixture tests construct actual files and digests for the supplied example states. Independent CLI reports also pass the required fields, nullability, arrays, graph references, comparison consistency, and diagnostic checks.
7. Concurrent publication tests preserve both successors. The independent unfinished/quarantined fixtures return unknown selection before quarantine and the same complete competing leaves afterward. Every independent read probe verifies unchanged fixture bytes.

## Checks

`go test ./...`, a fresh CLI build, eleven retained-checkpoint probes, and twelve graph probes passed at the tested revision. The implementation agent also reports passing full race, vet, build, and diff checks at its source-identical commit. Final combined-workflow verification will rerun the repository checks against the final source.

- [Go output](05-tests.txt)
- [Checkpoint probes](05-checkpoint-probes.json)
- [Graph probes](05-graph-probes.json)

The prior standards finding about duplicated timestamp validation is resolved by `recordread.ValidOffsetTimestamp`, shared by acceptance and recovery parsing. Both reported snapshot/timestamp defects were independently [rechecked](04-spec-recheck.md). The prior [standards report](04-standards-review.md) remains unchanged.

This evidence covers reader behavior. Skill authoring and actual fresh-agent continuation remain tickets 06 and 07. There is no human approval assertion.
