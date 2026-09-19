# Retained checkpoint verification

Actor: Codex /root. Date: 2026-09-19.
Tested revision: `c89b46ed6800b7c873bd6edb1f6b968c41f72acc`.
Agent implementation revision: `a7399711924c1e49b82ff4991882b70b6777320a`, with the same product files after cherry-pick.

## Results by criterion

1. Strict RecoveryNote parsing validates every required field and section, recursive duplicate YAML keys, identity and directory agreement, UUIDs, RFC3339 time, and digests. Parser tests preserve exact body and YAML metadata, including special values through the existing orientation encoding.
2. Cache tests exercise bounded entry enumeration and byte lookahead, exact-fit and extreme integer limits, fixed file paths, malformed directories, and symlink rejection. Namespace selection stays within the selected checkout, project, and task. Note prose remains returned text and does not select files.
3. Snapshot tests validate actual retained reader output, its exact digest, required schema fields, source digests, and root identity. Public CLI probes confirm that corrupt or missing snapshots leave the graph available while making comparison partial.
4. Comparison fixtures cover changed and unchanged sources, uncommitted changes, metadata text, authorized root relocation, missing sources, source ordering, non-root rename behavior, and reader-produced partial snapshots. An unmatched source from an incomplete selection remains unknown.
5. The authorization-loss CLI probe removes the previous document allowance. Recovery withholds its text and reports an outside-scope diagnostic. Missing historical paths remain unavailable. The text renderer labels reported checks as historical and does not turn the fixture's unsupported test claim into acceptance.
6. Malformed notes and interrupted publication produce explicit partial reports. Multi-observation stores and predecessor history return explicit operation failures in this intermediate slice. Ticket 05 owns removal of that limitation.
7. `go test ./...` and `go build -o /tmp/context-wayfinding-ctx ./cmd/ctx` passed at the tested revision. The independent public-CLI driver passed all eleven cases and checked unchanged fixture file bytes around every resume command. Each report passed the independent JSON/nullability/consistency checker. The implementation agent also reports full race, vet, and build checks at its source-identical commit.

## Retained outputs

- [Go tests](04-tests.txt)
- [Independent CLI probe results](04-checkpoint-probes.json)

The probe driver creates real temporary records, captures actual context bytes, computes their digests, and removes only its own temporary fixture directories. This evidence does not treat the illustrative JSON examples as executed reports. It establishes reader behavior, not model understanding or human approval. Independent spec review is a separate follow-up.
