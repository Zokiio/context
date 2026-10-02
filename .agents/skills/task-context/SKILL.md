---
name: task-context
description: Assemble a known local ticket's authored requirements, blockers, and context with the ctx reader before implementation or acceptance verification.
---

<!-- Generated from internal/initialization/templates/task-context.md. Run go generate ./internal/initialization after editing the template. -->

# Read task context

Build the current reader with `go build -o <temporary-ctx> ./cmd/ctx` from the repository directory. The repository binding selects `.scratch/records` and authorizes the repository directory, including root `CONTEXT.md`.

Read a known ticket with explicit project scope:

```sh
<temporary-ctx> context --project <absolute-repository-directory> \
	--ticket <ticket-path-relative-to-records>
```

If the caller supplied narrower scope, preserve it. For direct records access, replace `--project` with `--bundle <absolute-records-directory>` and repeat `--allow-source` for each authorized document root. Use the caller's requested limits or the reader defaults.

Capture the exit status and JSON. Read every returned source's `text` together with its `path` and `reasons`, or supply the full JSON to the coding agent handling the ticket. Completion means the coding agent has received the selected source text, not merely a source list or summary.

If either completeness field is false, report the diagnostics and resolve missing required context before implementation that depends on it. Exit status `2` requires correcting the invocation or operation failure. Cycle warnings can accompany a complete result.

Continue the assigned implementation or verification using the ticket's acceptance criteria. Authored links define selection; inspect additional code or guidance as the task requires. Keep relevant documents without authored links separate from reader defects.

Context completeness means the selected sources were supplied. It does not establish work readiness or accepted completion. Inspect orientation's separate checks before choosing work. For completion or reassessment, follow [Record acceptance](../../../docs/agents/acceptance.md).

When recording an acceptance trial, run `<temporary-ctx> version` and retain its output alongside the ticket path, invocation, exit status, completeness fields, delivered source digests, and observed development outcome. The command reports embedded version, revision, and modified status. A modified or unstamped binary remains visibly uncertain; its output does not identify uncommitted changes. Keep that uncertainty in the report rather than attributing the binary to a clean revision.
