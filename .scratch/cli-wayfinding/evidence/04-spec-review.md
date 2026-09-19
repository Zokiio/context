# Ticket 04 independent spec review

Reviewed `babaae9...c89b46e` in pinned, clean `/tmp/context-cli-checkpoint-review`. Multi-node and predecessor-history operation errors are outside this slice's acceptance scope.

## Required behavior missing or partial

No additional findings.

## Scope creep

None found.

## Implemented but wrong

- **[P2] Retained root type disagrees with the current reader.** `internal/resumption/snapshot.go:108-111` compares the root's type literally, although current orientation trims it. For a ticket with `type: " WorkItem "`, actual `ctx context` produces a complete collection and current orientation recognizes the WorkItem. Retaining those exact bytes with matching snapshot/source digests and the same task ID nevertheless produces `recovery_snapshot_invalid` and exit 1. The spec says "`context.json` retains the exact version-1 output bytes of a task-context collection" and requires confirming "that its root WorkItem identity matches the note" (`.scratch/cli-wayfinding/spec.md:123-125`). Use the current reader's effective type interpretation when validating the retained root.

- **[P3] Invalid offset minutes pass timestamp validation.** `internal/resumption/note.go:161-163` bounds timezone hours but not minutes. Go's RFC3339 parser accepts minute 60. Independent CLI checkpoints with `observedAt: '2026-09-19T00:00:00+00:60'` and `'-23:60'` offsets returned exit 0, `complete: true`, and a valid graph/snapshot. The spec requires "`observedAt` is an RFC 3339 timestamp" (`.scratch/cli-wayfinding/spec.md:111`), whose offset minutes are 00–59. Bound the minutes explicitly and retain regression cases for both signs.

Validation: built the pinned CLI and reproduced both findings through real temporary files, using exact captured reader output and recomputed digests. Inspected cache lookahead limits, containment, invalid/partial snapshot handling, comparison authorization and ordering, and text/JSON output. No tracked files changed; HEAD remains `c89b46e`.
