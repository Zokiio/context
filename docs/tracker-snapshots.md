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

Use the existing [WorkItem profile](agents/issue-tracker.md#minimal-implementation-ticket-profile) for identity, title, triage, relationships, and criteria. Copy the source requirements faithfully. Record local adaptations separately from the source text, including the evidence for any local execution state or relationship declaration. A remote open state or ready-for-agent label does not imply `execution: unstarted`.

## Refresh and read

1. Retrieve the current task through the project's tracker tool. Retain the response and retrieval time. Compare requirements, state, and relevant comments with the prior capture.
2. Refresh changed source content while preserving local IDs. Retain the previous capture when existing evidence or recovery notes refer to it. If retrieval fails, report freshness as unverified.
3. Link only the local documents needed for the task under Spec or Context. In a multi-context repository, include the relevant context map, glossary, and decisions. Review remote blockers before adapting them into supported local relationships. A missing relationship section does not establish that no blockers exist.
4. Run `ctx context`, read all selected text, and inspect `ctx orient` diagnostics. For continuation, run `ctx resume` and compare current requirements and checkout state with its retained observations.
5. Verify against the captured requirements. Keep evidence and generated Acceptance records local according to project policy. They do not update or close the remote issue, and an automated review is not human approval.

Repeat the freshness check before continuation and acceptance. A successful local read means the captured inputs were readable, not that the remote task is unchanged.

## Two reader frictions from the trial

| Source input | Current behavior | Trial adaptation | Proposed leniency, not implemented |
| --- | --- | --- | --- |
| `None.` in a relationship section | Orientation rejects it as `invalid_relationship_section`; the recognized sentinel is `None` | Normalize the standalone sentinel in the local snapshot and preserve the raw response | Accept a single trailing period on the standalone sentinel. Continue reporting other prose, missing sections, and unresolved links as unknown |
| No execution field | Orientation reports an invalid profile and unknown execution; the task cannot enter its eligible shortlist | Record `in-progress` only after local work was observed, with its local meaning stated | Permit absent execution as unknown in tracker snapshots without treating the omission as malformed. Keep unknown execution ineligible and do not infer it from remote status or triage |

These proposals need focused parser and eligibility tests before implementation. The trial does not justify a new Snapshot record type. Keep the missing-execution proposal separate from relaxing locally authored WorkItem requirements.

## Bootstrap checklist before another trial

- Establish the authoritative tracker. If there is none, offer local file tracking.
- Supply a ctx binary and choose a local records directory. Retain its origin, version or source revision, and checksum. For a build from uncommitted source, also retain a source digest manifest. The consuming project should not need the ctx source checkout to establish binary provenance. Ignore disposable snapshots, evidence, binary files, recovery cache, and configuration locks. Preserve durable project guidance.
- Create a Project manifest selecting one real task. For a tracker-owned task, use the snapshot convention above.
- Discover the agent tool and its existing skills location. Adapt task-context, recovery-notes and its PROFILE.md, and the acceptance procedure. Replace source-repository build instructions and paths. Preserve existing instructions and add concise entry pointers.
- Bind the chosen records and the required source roots with [ctx setup](connect-projects.md). Read the task, work on it, and log setup friction and elapsed time.
- Measure continuation in a fresh session given only the ticket path. Record whether it finds the guidance, reads the checkpoint, checks current facts, and chooses the correct next action.

Use this checklist for the next project. One trial supports these manual steps; it has not established repeated installation requirements for a `ctx init` command.
