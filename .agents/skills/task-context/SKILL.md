---
name: task-context
description: Assemble a known local ticket's authored requirements, blockers, and context with the ctx reader before implementation or acceptance verification.
---

# Read task context

Use the `ctx` reader for a known ticket in this repository. Resolve the repository directory, selected bundle, ticket path, and authorized document directories explicitly. For tickets in this repository, the bundle is `.scratch/records` and the repository directory is an authorized source root because tickets link to root `CONTEXT.md`.

Build the current reader with `go build -o <temporary-binary> ./cmd/ctx` from the repository directory. Invoke that binary with explicit scope:

```sh
<temporary-binary> context --bundle <absolute-bundle-directory> \
	--ticket <ticket-path-relative-to-bundle> \
	--allow-source <absolute-authorized-document-directory>
```

Repeat `--allow-source` for other authorized directories. If the caller supplied a narrower scope, preserve it. Use the caller's requested limits or the reader defaults.

Capture the exit status and JSON. Read every returned source's `text` together with its `path` and `reasons`, or supply the full JSON to the coding agent handling the ticket. Use those sources as the ticket's authored task context. Completion of this step means the coding agent has received the selected source text, not merely a source list or summary.

If either completeness field is false, report the diagnostics and resolve missing required context before implementation that depends on it. Exit status `2` requires correcting the invocation or operation failure. Cycle warnings can accompany a complete result.

Continue the assigned implementation or verification using the ticket's acceptance criteria. Authored links define selection; inspect additional code or guidance as the task requires. Keep relevant documents without authored links separate from reader defects.

Context completeness means the selected sources were supplied. It does not establish work readiness or accepted completion. When orientation is available, inspect its separate checks before choosing work. For completion or reassessment, follow [Record acceptance](../../../docs/agents/acceptance.md).

When recording an acceptance trial, run `<temporary-binary> version` and retain its output alongside the ticket path, invocation, exit status, completeness fields, delivered source digests, and observed development outcome. The command reports embedded version, revision, and modified status. A modified or unstamped binary remains visibly uncertain; its output does not identify uncommitted changes. Keep that uncertainty in the report rather than attributing the binary to a clean revision.
