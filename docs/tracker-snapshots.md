# Read an externally tracked task through a local snapshot

Use this manual profile when the tracker owns the task and ctx can only read local Markdown. It records the arrangement exercised with Opsbase issue 165. It is an authoring convention for an existing WorkItem, not a new reader type or a connector.

## Ownership and identity

Keep the tracker authoritative for requirements and tracker status. Retain its raw response alongside the Markdown snapshot. Proposed requirement changes go through the project's tracker workflow. Do not maintain a second editable backlog in the snapshot directory.

Keep the local Project and WorkItem IDs stable across refreshes. Generate UUIDs when first creating those records. Record these additional fields in the WorkItem frontmatter:

| Field | Meaning |
| --- | --- |
| `sourceURL` | Canonical remote task URL |
| `sourceUpdatedAt` | Tracker's update timestamp, with UTC offset |
| `snapshotAt` | Retrieval timestamp, with UTC offset |

These fields document provenance. The current reader does not fetch the URL, validate freshness against the tracker, or interpret these fields as a synchronized status. Retain the raw response and disclose unavailable source timestamps instead of inventing one.

For the missing-execution leniency, identify a tracker snapshot by `type: WorkItem` and a string `sourceURL` that parses as an absolute HTTP or HTTPS URL with a nonempty host. Reserve this field for the authoritative upstream task. Links to related remote material belong in the body and do not mark a snapshot. This explicit authoring signal avoids introducing another record type.

The leniency applies only when the `execution` key is absent. An empty, null, or unrecognized execution value remains invalid. An absent, empty, non-string, or malformed `sourceURL` does not enable the leniency. Ordinary WorkItems retain their required execution field. In every case, unknown execution stays ineligible for pickup, and a source URL does not establish freshness, readiness, or permission to start.

Use the existing [WorkItem profile](agents/issue-tracker.md#minimal-implementation-ticket-profile) for identity, title, triage, relationships, and criteria. Copy the source requirements faithfully. Record local adaptations separately from the source text, including the evidence for any local execution state or relationship declaration. A remote open state or ready-for-agent label does not imply `execution: unstarted`.

## Refresh and read

1. Retrieve the current task through the project's tracker tool. Retain the response and retrieval time. Compare requirements, state, and relevant comments with the prior capture.
2. Refresh changed source content while preserving local IDs. Retain the previous capture when existing evidence or recovery notes refer to it. If retrieval fails, report freshness as unverified.
3. Link only the local documents needed for the task under Spec or Context. In a multi-context repository, include the relevant context map, glossary, and decisions. Review remote blockers before adapting them into supported local relationships. A missing relationship section does not establish that no blockers exist.
4. Run `ctx context`, read all selected text, and inspect `ctx orient` diagnostics. For continuation, run `ctx resume` and compare current requirements and checkout state with its retained observations.
5. Verify against the captured requirements. Keep evidence and generated Acceptance records local according to project policy. They do not update or close the remote issue, and an automated review is not human approval.

Repeat the freshness check before continuation and acceptance. A successful local read means the captured inputs were readable, not that the remote task is unchanged.

## Two reader frictions from the trial

| Source input | Current behavior | Trial adaptation | Handling |
| --- | --- | --- | --- |
| `None.` in a relationship section | Orientation accepts standalone `None` and `None.` | Normalize the standalone sentinel in the local snapshot and preserve the raw response | Implemented: a single trailing period is accepted. Other prose, missing sections, and unresolved links remain unknown |
| No execution field | A marked tracker snapshot may omit execution; its execution remains unknown and it cannot enter the eligible shortlist | Record `in-progress` only after local work was observed, with its local meaning stated | Implemented: permit absent execution as unknown in WorkItems marked by a valid `sourceURL` as defined above without treating the omission as malformed. Keep unknown execution ineligible and do not infer it from remote status or triage |

The leniency uses the existing WorkItem type. Native WorkItems still require execution; neither snapshot metadata nor an unknown execution state satisfies a dependency.

## Bootstrap checklist before another trial

- Establish the authoritative tracker. If there is none, offer local file tracking.
- Supply a ctx binary and choose a local records directory. Run `ctx version` and retain its output. It reports the embedded module version, VCS revision, and modified flag without requiring the ctx checkout. A modified or unstamped build remains visibly uncertain; the output does not identify uncommitted changes. Ignore disposable snapshots, evidence, binary files, recovery cache, and configuration locks. Preserve durable project guidance.
- Create a Project manifest selecting one real task. For a tracker-owned task, use the snapshot convention above.
- Discover the agent tool and its existing skills location. Adapt task-context, recovery-notes and its PROFILE.md, and the acceptance procedure. Replace source-repository build instructions and paths. Preserve existing instructions and add concise entry pointers.
- Bind the chosen records and the required source roots with [ctx setup](connect-projects.md). Read the task, work on it, and log setup friction and elapsed time.
- Measure continuation with and without a checkpoint using the paired procedure below. Give each fresh session only the ticket path.

Use this checklist for the next project. One trial supports these manual steps; it has not established repeated installation requirements for a `ctx init` command.

## Next trial and recovery baseline

Use [Mukabi](https://github.com/Zokiio/Mukabi) with Claude Code for the next trial. Its checkout has project-specific `CLAUDE.md` guidance, and Claude Code is installed locally. This changes the harness axis from Opsbase's Codex trial. Try individual symlinks from `.claude/skills/` to shared skill directories before copying. Preserve existing instructions and skill layouts. Record discovery success, broken links, any copy fallback, and each adaptation in the same friction table. Add hooks only if the selected task requires them. It does not yet establish that Claude discovers the adapted skills or can use this workflow. Confirm those behaviors during bootstrap, preserve the existing tracker, and select one real unfinished task before starting. Do not create a second tracker merely to change an axis.

For the recovery comparison:

1. Pause at a real unfinished step and retain the checkout, records, binary, guidance, and recovery cache. Define the observable correct next action before either run, including the requirements and unfinished work the session must recognize.
2. Run two fresh sessions from that same frozen checkout state and absolute path. Restore the disposable trial checkout between runs after the prior writer stops. Keep the original checkpoint archive outside it. Include the task's cache in one condition and omit it in the other. Keep all other inputs, permissions, model, and harness settings equal. Neither session receives the other's history or output.
3. Run the no-checkpoint condition first, then restore the frozen state and run the checkpoint condition. Record the order. Give each session only the same ticket path. Time launch to the first verifiably correct continuation plan, and separately to its first correct action. Report startup time and a session that fails to continue rather than inventing a successful timing. Keep later tests and record writing outside these recovery intervals.
4. Report both timings and the difference, no-checkpoint time minus checkpoint time. Retain the observations that establish correctness. One pair gives an observed difference, not a stable estimate of time saved across projects.

Compare actual bootstrap edits with the Opsbase log. Only steps that repeat unchanged are candidates for a future init command. Keep the init ticket draft until this trial supplies that evidence.
