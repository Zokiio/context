# Task workspace and observable factory runs

Approved by the user on 23 October 2026. Implement the agreed task workspace plan and publish one combined PR. The implementation baseline is the run-graph stack through #54 plus the builder profile fix from #49. The combined PR also includes the factory research and long-term proposal from #44 and #46; those documents retain their original proposal status and do not expand this milestone's requirements.

## Outcome

A person can write and manage a task without accidentally starting agents, then follow and intervene in its real execution from a permanent task page. Task, Run, and History views share a stable work-item identity. Board drawers become optional previews rather than the primary place to write or decide.

## Required behavior

- Render descriptions and agent notes as safe Markdown, including headings, lists, links, and code blocks. Provide a spacious responsive editor and preview, structured criteria, dependencies, and editable spec/context links.
- Saving a new task defaults to an unqueued draft. Queueing explicitly adds a current commitment and explains the project's ask policy. Starting uses existing readiness and authorization checks. Removing an unstarted task from the queue prevents automatic pickup.
- Keep task disposition separate from document lifecycle and execution. Existing records remain authoritative; no duplicate backlog is introduced. Preserve existing API/CLI creation compatibility through an explicit new draft option.
- Preserve unsaved edits, focus, caret, selected tabs, board navigation, and run inspection state across background updates. Warn before discarding edits. A stale save retains the local draft and exposes the latest record for comparison.
- Active or uncertain execution freezes requirement edits. Existing runs and evidence retain their admitted requirement snapshots when current records later change.
- Archive and restore preserve identity, relationships, and history. Restore returns to an unqueued state. Show affected dependents. Permanent deletion is limited to unreferenced, never-run drafts. Active or uncertain refinement, test authoring, implementation, or verification prevents removal.
- Show the current run stage, actual progress, available worker output and artifacts, check results, elapsed time, last contact, and reported cost. Do not present unavailable or final-only usage as live observations.
- Refresh run information independently of board changes and open editors. Provide optional follow-active-step behavior without resetting manual selection, scroll, or expanded logs. Show connection freshness and failures.
- Expose authorized stop, review, repair, merge retry, and reconciliation controls where applicable. A stop request is distinct from confirmed termination or an unknown outcome. Explicit stopping prevents automatic restart.
- History retains previous jobs, individual attempts, refinement and test proposals, decisions, delegated authority, original failures, logs, and tested revisions.
- Keep revision checks, authorization, idempotency, and interruption recovery at the service boundary, including operations affecting both the work item and project commitment links.

## Verification

Use deterministic fixture workers and a real browser to exercise saving without execution, editing, queue/start, failed checks, output inspection, repair, review, verified merge, stopping, archive/restore/deletion, stale saves, unsaved navigation, and reconnect. Inspect resulting service records and Git revisions. Check long Markdown and responsive layouts. Keep generated evidence ignored. Provider compatibility is separately attributed and is not implied by fixture results.
