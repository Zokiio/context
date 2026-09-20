# Record acceptance

Use this procedure when completing a work item, reassessing changed requirements or evidence, or backfilling a completed record. Reader commands remain read-only. Follow the project's tracker and evidence-storage conventions when authoring records. For an external tracker, use the [snapshot profile](../tracker-snapshots.md). A local Acceptance assesses captured requirements and does not update or close the remote issue.

## Complete or reassess a work item

1. Read the ticket through the [task-context workflow](../../.agents/skills/task-context/SKILL.md). Verify its criteria against actual results and identify any missing evidence. Record failures and untested criteria without marking them passed.
2. Retain the results in an immutable UTF-8 evidence document. Identify the commands or observations, actor, date, and revision tested. If a commit does not identify an uncommitted working tree, include a source digest manifest. A later merge revision does not replace the original tested revision.
3. Check the authoritative ticket and settle its requirements, relationship links, and acceptance checklist. Refresh a tracker snapshot if its source changed. Prepare an empty current Acceptance section, including its surrounding line breaks, before taking fingerprints. A new blank line before that heading belongs to the retained body; filling the section's link later does not. Record completion only after every criterion has evidence. Routine completion notes belong under Comments.
4. Run the supplied `ctx orient` binary using the verified project binding, or explicit `--bundle` and `--allow-source` roots. Obtain the current fingerprintVersion, ticketSHA256, criteriaSHA256, and whole-source digests. Inspect any partial-result diagnostics. Unknown acceptance or readiness is not a passing check.
5. Create a new Acceptance record using the profile below. Snapshot all directly selected Spec and Context files and linked blocking decisions, plus any additional accepted requirement sources. Link at least one evidence source. Use current digests from the same observed files and preserve evidence provenance.
6. Link the new record from the ticket's single current Acceptance section and record observed execution as completed. For a tracker snapshot, this is a local observation and does not establish remote completion. Preserve the previous decision and its old link in history. The new decision's author and time describe the current judgment, even when its evidence is historical.
7. Run orientation again. Verify that this ticket's acceptance is valid and fresh. Inspect its prerequisite edges and resolve failed or unknown supported checks before claiming those edges are satisfied. Retain the resulting observation separately from earlier immutable evidence.

Changes to any linked requirement or evidence file can stale acceptance, including edits to linked conventions. Reassess the effect and create a new decision. Preserve both the prior decision and the original test results. Unchanged historical acceptance does not need the current Git HEAD to equal its tested revision.

## Acceptance profile

An Acceptance is an OKF record inside the project bundle. Its YAML frontmatter contains these fields:

| Field | Required value |
| --- | --- |
| type | Acceptance |
| id | Nonempty stable ID, unique in the project |
| title | Nonempty authored title |
| projectId | The subject project's ID |
| workItemId | The subject WorkItem's ID |
| actor | Mapping with kind equal to human or workflow, and a nonempty identity |
| decidedAt | Timestamp with an explicit UTC offset |
| testedRevision | Mapping with nonempty origin and revision |
| fingerprintVersion | Integer 1 |
| ticketSHA256 | Lowercase hexadecimal SHA-256 of the ticket requirement fingerprint |
| criteriaSHA256 | Lowercase hexadecimal SHA-256 of the criteria fingerprint |

Optional `humanApprovals` is a list whose entries retain their own human actor, decidedAt, and testedRevision. A workflow decision or automated review is not a human approval. These fields attribute assertions; the reader cannot authenticate the named actor.

The record has required level-two Requirements and Evidence sections. Each snapshot is one Markdown list item containing one local file link and one inline-code SHA-256 digest. Requirements can be explicitly empty when the subject selects no requirement files. Evidence contains at least one source.

All directly selected Spec and Context documents and linked blocking decisions require snapshots. Additional snapshots are explicit accepted requirements and must remain available and unchanged. Evidence can describe a remote observation, but orientation reads the retained local UTF-8 record and does not fetch remote or binary evidence.

An Acceptance cannot snapshot itself or use itself as evidence. A snapshot pointing to the subject ticket uses ticketSHA256 rather than its whole-file digest. The current decision covers the entire Acceptance criteria section; partial acceptance cannot satisfy a prerequisite.

For fingerprint values, use the current orientation report. The [orientation specification](../../.scratch/session-orientation/spec.md#acceptance-fingerprints-and-freshness) owns the byte encoding and independent conformance requirements. Keep that implementation reference separate when adapting this procedure for another project.
