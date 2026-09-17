# Spec review: configuration, discovery, readers, and workspace navigation

Pinned final revision: `2777f087a5e320594e09c5cda1f35d62e66c5a80`. Reviewed PRs #4, #6, #7, #8, and their contracts in #10 against `.scratch/project-discovery/spec.md`, tickets 01/02/03/04/06, and ADR 0004. Whole-stack base: `1eb54f87242502020181b46504d9556e5bea6a6a`.

No confirmed findings. No missing requirement, incorrect implementation, or material scope expansion identified in these slices. Severity: none.

I built the pinned reader and delivered the complete selected context for all five tickets using explicit `--bundle`, `--allow-source`, `--max-files 500`, and `--max-bytes 8388608`. Every read returned exit 0, both completeness fields true, and no diagnostics. Final orientation likewise returned complete inventory and valid acceptance for all discovery tickets with all supported checks passing.

Thirty independent CLI probes passed in temporary non-Git projects outside repository ancestors. They covered physical symlink-before-parent selection; encountered symlinked-config bases; same-depth root conflicts; malformed registry and marker handling; no fallback from broken selected scope; aliases with missing checkouts; direct manifestless migration; unchanged project schemas and partial budget reports; source-symlink containment; workspace authored order, unavailable members, shallow availability, member-specific roots, and POSIX command round-tripping with quotes and newlines. A read-only hash comparison found no writes. See `spec-readers-probes.py` and `spec-readers-probes.json`.

The discovery, CLI, workspace, and shared-parser test packages pass at the pinned revision. Full `go test ./...` also passed from isolated archives of the four intermediate PR heads: `18728ee`, `39483dd`, `5c21482`, and `573c220`. See `spec-readers-slices.json`. Additional parser probes compared numeric, boolean, null, and merge-key handling with the base revision.

The product worktree remains clean. No tracked files, acceptance records, Git refs, or external reviews were changed. This review does not cover setup internals, which have a separate reviewer.
