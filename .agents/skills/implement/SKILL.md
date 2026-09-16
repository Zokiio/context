---
name: implement
description: "Implement a piece of work based on a spec or set of tickets."
disable-model-invocation: true
---

Implement the work described by the user in the spec or tickets.

For local tickets, read the tracker conventions and use /task-context before implementation. Respect the approved blocker graph. When choosing unspecified work and orientation supports the required checks, use `ctx orient` with explicit scope and choose from its explained eligible list.

Record execution as in-progress before implementation. Keep triage, execution, and computed readiness separate. During bootstrap, an unsupported orientation check remains unknown; use the approved dependency order and retained verification without claiming that the command passed that check.

Use /tdd where possible, at pre-agreed seams.

Run typechecking regularly, single test files regularly, and the full test suite once at the end.

Once done, use /code-review to review the work.

Fix findings and rerun affected checks. Retain criteria evidence against the tested revision, including a source manifest for uncommitted code. Before reporting completion, follow [Record acceptance](../../../docs/agents/acceptance.md), including its staged bootstrap procedure. Preserve historical decisions and human approval attribution.

Commit your work to the current branch.
