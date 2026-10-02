# Use ctx with this project's tracker

The authoritative tracker is {{.Tracker}}. Preserve its requirements, status, and workflow. Read existing project instructions before selecting a task or changing its records. Init creates an empty Project manifest and does not create work items or replace a backlog.

The selected record store is `{{.Records}}`. Keep authored guidance and records in Git according to project policy. Store disposable snapshots, raw tracker responses, binaries, and evidence under `{{.Local}}`. Keep generated Acceptance records under `{{.Records}}/acceptances/` and recovery observations under the checkout's `.context-cache/`. These local files are disposable or retained verification material, not a second authoritative backlog.

## Select a real task

Use the existing tracker, including local roadmaps and plans. If its identity is unclear, ask which tracker is used. Offer local file tracking only when none exists. Select the required task documents explicitly, including relevant context maps, domain glossaries, and decisions. Put file links under the ticket's level-two Spec or Context headings. Other prose links do not select task context.

Keep task records inside the bundle. Plain Markdown documents can live outside it in explicitly authorized source directories. Paths in record links are relative to the containing file; a leading slash means the bundle root. Add source roots through `ctx setup --allow-source` after reviewing the complete intended list. A changed binding requires separate review and explicit setup replacement.

## Local reader records

Use these fields when authoring local reader input. Preserve an existing record's stable ID; generate a UUID for a new identity.

- A Project needs `type: Project`, `id`, and `title` in YAML frontmatter, with level-two Goals, Current commitments, and Open decisions sections. Commitments link directly to chosen WorkItems. Empty sections or `None` declare no commitments or decisions.
- A WorkItem needs `type: WorkItem`, `id`, `title`, and `triage`. The optional document `status` is draft, stable, or deprecated. Record observed execution separately as unstarted, in-progress, completed, or cancelled. Unknown execution remains unknown; remote status and triage do not supply it.
- Each WorkItem needs nonempty Acceptance criteria, Blocked by, and Blocked by decisions sections. Relationship sections contain local file links or an explicit empty declaration. Missing information is unknown. Spec is optional. A completed execution state alone does not satisfy a dependency.
- A Decision needs `type: Decision`, `id`, `title`, and `decisionState: open` or `resolved`. A resolved decision needs a nonempty Resolution section. Route authoritative questions through the existing tracker. Local Decision records represent that authority only when the project workflow permits it.

Use the project's triage mapping. The current reader considers `ready-for-agent` for pickup, separately from commitments, execution, and readiness. A local reader convention does not authorize changes to the tracker.

## Tracker snapshots

When the task is owned outside the local reader records, retain its original source and retrieval time. Create an identified local WorkItem snapshot only when needed. Keep its Project and WorkItem IDs stable across refreshes. Record the authoritative `sourceURL`, `sourceUpdatedAt` when available, and `snapshotAt` with an explicit UTC offset. Disclose unavailable timestamps rather than inventing them. For a local plan, identify its source path and observed revision or digest.

Copy requirements faithfully. Record local format adaptations and execution observations separately. An open remote issue or a ready label does not establish unstarted execution. The current reader reports missing execution as unknown and an invalid profile; do not invent a value to silence that diagnostic. Missing blocker information is also unknown.

Refresh the source before continuation and acceptance. Preserve earlier captures referenced by evidence or recovery notes. If retrieval fails, report freshness as unverified. A successful context read establishes readability of the capture, not remote freshness. Local decisions and snapshots do not update the authoritative tracker.

Keep disposable snapshot bundles under the local directory when the tracker owns all their content. Binding that snapshot bundle is an explicit follow-up setup choice. Keep durable task refinements beside authoritative local records when the project workflow allows them. Do not put disposable snapshots in Git merely because init created a durable records directory.

## Verification and acceptance

Use [Record acceptance]({{.AcceptanceLink}}) when completing or reassessing a task. Retain original tested revisions, evidence, and digests. Fresh checkouts lack ignored local acceptance records, so their acceptance remains unknown until records and evidence are restored together or checks are rerun. Local acceptance does not establish a tracker update, human review, merge approval, or hosted CI success.
