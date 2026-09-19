# Standards review: `babaae9...c89b46e`

Reviewed the pinned checkpoint at `c89b46e`. The later public-reference update is outside this slice and is omitted here as requested.

## Documented standards

No violations found. The application logic remains outside `internal/cli`, the CLI owns rendering and exit status, tests use real temporary directories, and recovery metadata reuses the existing JSON-safe conversion rather than copying it.

## Possible smells

1. **Possible Duplicated Code (judgment) — `internal/resumption/note.go:23,94-103,154-164` and `internal/orientation/acceptance.go:9,166-174`.** Both paths validate the same domain value, an RFC 3339 timestamp with an explicit bounded offset, but maintain separate regular expressions and parse routines. They have already drifted: the acceptance expression restricts offset minutes to `00..59`, while the new note path only bounds the hour and therefore accepts `+00:60` under Go's parser. Extract one shared explicit-offset timestamp validator and keep the caller-specific error/reporting behavior in each package. This is small enough to remove the drift without changing the resumption design.

## Verification

`go test ./internal/resumption ./internal/orientation ./internal/cli` passed while the review checkout was pinned at `c89b46e`. No other baseline smells were strong enough to report; the single-observation flow is the deliberate ticket-04 boundary.
