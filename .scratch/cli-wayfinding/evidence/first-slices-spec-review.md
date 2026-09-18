# Spec review of CLI slices 01 and 02

Reviewed `7b93da4...88b7e23` in pinned, clean `/tmp/context-cli-review`.

## Required behavior missing or partial

No additional findings.

## Scope creep

None found. Ticket 03's execution-state change is preparation for authorized later work, not an assertion that its functionality is delivered.

## Implemented but wrong

- **[P3] Decision rows reorder affected commitments by ID.** `internal/cli/orientation_compact.go:187-191` copies `decision.AffectedWork` directly. That array is sorted by stable ID in the existing evaluation result, so the separate `decision/open_decision` cause reverses authored commitment order whenever the manifest lists `z.md` before `a.md`. A real filesystem fixture with those two commitments sharing one open decision prints `Z, A` under `blocking_decisions/open_decision`, but `A, Z` under `decision/open_decision`. The specification requires: "Preserve first affected commitment order; within one commitment preserve existing check order." (`.scratch/cli-wayfinding/spec.md:37`). Collect this decision row's affected references in `CurrentCommitments` order without changing JSON or detailed output.

Validation: `go test ./internal/recordread ./internal/taskcontext ./internal/orientation ./internal/cli` passed. Built the CLI and reproduced the ordering issue through an independent temporary project. Inspected shared byte capture, per-role cache authorization, budgets and omissions, compact diagnostic grouping, and detail/JSON/workspace dispatch. No tracked files changed; HEAD remains `88b7e23`.
