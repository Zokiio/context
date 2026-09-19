# Standards review: `2fc3a52...25ef670`

Reviewed the pinned history implementation at `25ef670`. The later reassessment records are excluded. Public-reference update 07 is intentionally outside this review.

## Documented standards

No violations found. `internal/resumption` owns graph evaluation and candidate comparison, while `internal/cli` continues to render the resulting report and choose exit status. Cache inspection stays read-only and preserves the bounded-read, physical-path checks already established for this reader.

The prior duplicated RFC 3339 offset validation is resolved. [timestamp.go](/private/tmp/context-cli-history-review/internal/recordread/timestamp.go) now owns `ValidOffsetTimestamp`; [acceptance.go](/private/tmp/context-cli-history-review/internal/orientation/acceptance.go) and [note.go](/private/tmp/context-cli-history-review/internal/resumption/note.go) use it. The shared validator rejects the former `+00:60` case and keeps the callers' domain-specific error handling.

## Possible smells

None supported by the changed code. The graph routine has one responsibility: validate the observed predecessor graph and derive leaf candidates. Candidate snapshot loading stays in `inspectCandidate`, so graph selection does not become coupled to snapshot parsing. The diagnostic de-duplication code has a different contract from existing reader-diagnostic normalization because it must retain the distinction between a null attribution and an authored empty value.

## Verification

`git diff --check 2fc3a52...HEAD` and `go test ./internal/recordread ./internal/resumption ./internal/orientation ./internal/cli` passed.
