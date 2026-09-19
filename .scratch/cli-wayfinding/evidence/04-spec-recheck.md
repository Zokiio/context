# Ticket 04 supplemental spec finding recheck

Both findings from `04-spec-review.md` are resolved at `2fc3a524a436018bcfd240090d348d28e0dd7b9a`, which includes timestamp fix `5ec85de3fd9340d7123287f4f86a9cd42057d2f9`.

Built the revised CLI and repeated the original independent filesystem probes:

- `observedAt` offsets `+00:60` and `-23:60` now return exit 1, incomplete reports, invalid recovery graphs, `recovery_invalid_note`, and no selected candidate IDs.
- Exact context output captured from a ticket declaring `type: " WorkItem "` now returns exit 0, a valid retained snapshot, and a complete comparison.

This recheck covers only the two reported defects. The original review is preserved. The authorized review checkout switch completed; `/tmp/context-cli-checkpoint-review` is clean at `2fc3a52`.
