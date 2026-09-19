# Compact-path independent review

No remaining findings in ticket 08, the compact specification, reader reference, guide, or current renderer diff.

The draft initially omitted task selectors from healthy commitments and shortlist entries. A real CLI fixture reproduced this. The implementation now retains root-relative selectors in commitments, in-progress work, and shortlist rows; only identifiable affected-work rows omit their paths. The same fixture passes after rebuilding.

Focused verification passed:

- Healthy `left/task.md` and `right/task.md` remain distinct, copyable selectors despite equal titles and filenames.
- An authorized missing source in sibling directory `records-other` remains explicitly outside `records`.
- Four remote relationships retain separate referring paths and authored links, while shared missing-source findings retain affected work.
- Authored goal text is unchanged. An invalid project identity preserves original absolute finding paths and reports an unknown root.
- Detailed and JSON reports preserve absolute source paths. Focused CLI tests cover unchanged JSON, result nonmutation, unknown roots, grouping, and existing partial-state behavior.

Commands: `go build -o /tmp/ctx08-independent ./cmd/ctx`; `go test ./internal/cli -run 'TestCompact|TestOrientationJSONIsUnchangedAndDetailConflicts' -count=1`; independent temporary-filesystem CLI probes. No broader recovery review or suite rerun.

Reviewed renderer SHA-256: `3741902b5a9c705a85dc94e76dabd3e85a6c315a187d573043b61b579b10e69a`.

No code, documentation, ticket, or PR edits were made. Only this disposable report and temporary probe artifacts were written.
