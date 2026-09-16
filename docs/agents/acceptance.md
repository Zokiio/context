# Record acceptance

Use this procedure when completing a work item, reassessing changed requirements or evidence, or backfilling a completed record. Skills edit authored files explicitly. Both reader commands remain read-only.

## Complete or reassess a work item

1. Read the ticket through the [task-context workflow](../../.agents/skills/task-context/SKILL.md). Verify its criteria against actual results and identify any missing evidence. Record failures and untested criteria without marking them passed.
2. Retain the results in an immutable UTF-8 evidence document. Identify the commands or observations, actor, date, and revision tested. If a commit does not identify an uncommitted working tree, include a source digest manifest. A later merge revision does not replace the original tested revision.
3. Settle the ticket's requirements, relationship links, and acceptance checklist. These bytes are fingerprinted. Record completion only after every criterion has evidence. Routine completion notes belong under Comments.
4. Run `ctx orient` with explicit project and allowed-source roots. Obtain the current fingerprintVersion, ticketSHA256, criteriaSHA256, and whole-source digests. Inspect any partial-result diagnostics. Unknown acceptance or readiness is not a passing check.
5. Create a new Acceptance record using the profile below. Snapshot all directly selected Spec and Context files and linked blocking decisions, plus any additional accepted requirement sources. Link at least one evidence source. Use current digests from the same observed files and preserve evidence provenance.
6. Link the new record from the ticket's single current Acceptance section and set execution to completed. Preserve the previous decision and its old link in history. The new decision's author and time describe the current judgment, even when its evidence is historical.
7. Run orientation again. Verify that this ticket's acceptance is valid and fresh. Inspect its prerequisite edges and resolve failed or unknown supported checks before claiming those edges are satisfied. Retain the resulting observation separately from earlier immutable evidence.

During bootstrap, retain evidence even before fingerprint output and acceptance validation exist. Once fingerprint output is implemented, author records for already verified slices. Label them as awaiting validation until the acceptance checker exists.

The staged exception lasts until every required check is implemented. Ticket 05 can have valid, fresh acceptance while its non-leaf prerequisite checks remain unsupported and unknown until ticket 06. Use the approved blocker order and retained verification to complete these bootstrap slices. Distinguish acceptance validity from prerequisite satisfaction, and never report an unsupported edge as passed. Missing evidence or invalid records remain unresolved gaps even during bootstrap.

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

## Reproduce version-1 fingerprints

Use the current operation's values for normal authoring. For independent verification, follow this byte encoding exactly.

The ticket sequence begins with one byte string containing the complete Markdown body after frontmatter removal, excluding every exact level-two Comments and Acceptance section and its nested content. Retain every other body byte in document order. Use parsed Markdown structure so apparent headings in fenced code do not become boundaries.

The criteria sequence contains the complete raw source span of each Acceptance criteria section in document order. Each span begins at its heading and ends immediately before the next level-one or level-two heading, or at the end of the body.

For either sequence, append effective raw reference definitions used by retained content when the definition's bytes are not already retained. Preserve their line endings. Append each additional definition once in first-use order, including a definition located inside an excluded section. An unrelated comment edit stays irrelevant, but a definition edit that changes retained meaning remains relevant.

Encode the sequence in this order:

1. The UTF-8 domain tag, `ctx.ticket-body.v1` for the ticket or `ctx.acceptance-criteria.v1` for criteria.
2. One zero byte.
3. The number of byte strings as an unsigned 64-bit big-endian integer.
4. For each string, its byte length as an unsigned 64-bit big-endian integer, followed by its bytes.

Compute SHA-256 over that stream and render lowercase hexadecimal. Preserve original whitespace, Unicode bytes, and line endings. Linked requirement and evidence files use SHA-256 over the whole raw file instead of this encoding.

The [orientation specification](../../.scratch/session-orientation/spec.md#acceptance-fingerprints-and-freshness) owns the contract. Independent conformance fixtures verify interoperability without calling the production fingerprint helper to manufacture expected values.
