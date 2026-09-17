# Local orientation and resumption

Status: contract review fixes required. The [independent review](independent-review.md) identified gaps in bounded discovery, interrupted-publication recovery, and nested JSON shapes. The agreed product behavior remains intact, but the recovery contract is not ready for implementation until these gaps are addressed. No implementation or acceptance trial has run.

## Outcome and decision owners

A fresh agent can continue an interrupted real task from its ticket reference, using current project context and temporary working notes without requiring the user to reconstruct the session. Skills author notes and guide behavior. The CLI reads, compares, and reports.

The authoritative decisions are [the usable loop](issues/01-usable-local-loop.md#answer), [orientation presentation](issues/02-intervention-view.md#answer), [handoff behavior](issues/03-handoff-contract.md#answer), and [resumption behavior](issues/04-resume-behavior.md#answer). This specification defines the proposed implementation contract; it does not replace their rationale.

## Command contract

```sh
ctx orient
ctx orient --detail
ctx orient --json
ctx resume --ticket feature/issues/task.md
ctx resume --ticket feature/issues/task.md --json
ctx resume --bundle /work/project-records --checkout /work/application \
  --ticket feature/issues/task.md --allow-source /work/application/docs
```

`context` remains the full task-context JSON reader with its existing semantics. `setup` continues to connect existing records; neither command gains work-record writes.

### Orientation

Default project text becomes the compact overview. `--detail` selects the existing full presentation, extended only where this specification requires it. `--json` retains the full version-1 project report, existing keys, values, arrays, and readiness semantics. Do not replace it with a lossy summary. Reject `--detail --json` and repeated `--detail` with exit status 2.

The compact project presentation contains:

1. Project title, source location, authored goals, and evaluation/inventory completeness. Do not generate a prose summary of authored goals; preserve their text.
2. Current commitments in authored order, with execution and readiness shown separately.
3. Needs attention: shared blocking decisions, prerequisite failures, missing required sources, and unknown checks, with affected commitments and source references.
4. In-progress work and the existing shortlist for new pickup, explicitly distinguished.
5. Project references selected by existing goal/record relationships, and guidance to obtain task context, resumption, or expanded detail.

Group causes only when they have the same check or diagnostic code and resolved source/relationship identity. Equal message text is insufficient. Preserve first affected commitment order; within one commitment preserve existing check order. Keep distinct causes distinct. Do not infer assignees, priorities, delivery estimates, or whether a human must answer a question from prose.

Show all warnings/errors and unknown conditions in compact output, grouped where equivalent. Include references to the full diagnostic details. Do not impose a silent item cap or present incomplete inventory as healthy. Source text may make the overview long; compact means omitting routine passed-check detail, not truncating authored direction.

Use the same orientation result for text and JSON. No cache traversal is required for `orient`; continuation entries point to `resume` rather than claiming that a note was found. Workspace navigation remains unchanged. `--detail` on workspace orientation preserves that navigation report; it does not recursively evaluate members.

### Resumption

`resume` requires one nonempty `--ticket` and no positional arguments. Ticket paths use the existing `context` rules: relative to the selected bundle, absolute paths permitted only inside it. Initial delivery does not add ID-based command lookup. Stable IDs identify cache entries after the ticket is read.

Reuse `--project`, `--workspace`, `--bundle`, `--allow-source`, `--explain-scope`, `--max-files`, and `--max-bytes` with their existing validation. A selected workspace requires explicit project selection even if it has one member. No command prompts in noninteractive mode.

`resume` defaults to readable text; `--json` returns its full structured report. Add optional `--checkout PATH` for choosing the working directory whose cache is inspected. Add `--max-cache-files` and `--max-cache-bytes` as separate positive limits. Each scalar flag may occur at most once. `--allow-source` remains repeatable. Do not add `--detail` to `resume` initially.

The report refreshes project orientation and complete task context, then reads recovery observations for the selected task. Reuse the existing evaluations; do not implement another readiness predicate. In-progress exclusion from a pickup shortlist does not mean that continuing the task is blocked.

## Project, checkout, and cache scope

The selected Project manifest and WorkItem must have nonempty, unambiguous stable IDs. Preserve available context when identity cannot be established, but skip cache lookup and report identity as unknown. This requirement applies to `resume`; manifestless direct `context` remains supported.

Choose the cache root as follows:

| Invocation | Working directory for cache lookup |
| --- | --- |
| Project discovered through a shared or personal binding | Its resolved binding directory, independent of record-store location |
| Project selected through a bare Project marker | The marker directory |
| Explicit `--checkout PATH` | The explicit directory, overriding the inferred location |
| Direct `--bundle` without `--checkout` | Invalid invocation; require the explicit working directory |

Canonicalize this directory using the existing physical-path rules. It must exist and be accessible. It need not be a Git repository. Never use the shared Git common directory, search another checkout, or silently fall back to the record store when the selected working directory is unavailable. Existing alias-based readers can still operate without an available checkout; `resume` requires one.

Explicit relative checkout paths resolve from invocation cwd, never from the bundle or configuration file. With an unavailable cwd, an alias invocation requires an absolute explicit checkout if one is supplied. An empty explicit checkout is invalid.

An explicit checkout selects where to read notes, not which project to read and not which supporting sources are authorized. A mismatch between its notes and the selected project is a diagnostic, never a reason to switch projects. State the selected records and working directory in the report.

Store notes beneath `<working-directory>/.context-cache/resume-v1/`. Namespace entries by SHA-256 of the UTF-8 project ID, then SHA-256 of the UTF-8 task ID, lowercase hexadecimal without separators or newline bytes. Do not use raw IDs as path segments. These digests locate entries; identity fields in each note must still match.

Moving a record without changing its ID preserves lookup. Moving a whole checkout with its cache preserves lookup relative to its new root. A copied checkout has its own independent cache. Branch changes within the same directory require refresh and comparison; record any observed Git revision as provenance, not as a lookup key or proof of unchanged files.

## Recovery observations

One current recovery note is the normal user-facing state. To preserve concurrent observations without a shared mutable file, skills publish immutable checkpoints with predecessor links. The CLI computes the current candidates from those links; it never chooses by creation time.

Layout:

```text
.context-cache/resume-v1/<project-key>/<task-key>/
  observations/
    <observation-uuid>/
      note.md
      context.json
```

An observation UUID is a random UUID generated for that write attempt. `note.md` has UTF-8 Markdown and YAML frontmatter:

```yaml
---
type: RecoveryNote
version: 1
id: 281d6632-b93d-43ce-a5b7-796721964024
projectId: project-stable-id
taskId: task-stable-id
observedAt: '2026-09-17T17:00:00Z'
actor: coding-session-identifier
predecessors: []
ticketPath: feature/issues/task.md
checkoutRevision: null
contextFile: context.json
contextSHA256: <sha256-of-exact-context-file-bytes>
---
```

`checkoutRevision` is null or `{origin: <repository identity>, revision: <observed revision>}`. It is observation provenance only. Do not imply that a commit identifies uncommitted changes. Every check below records its own tested revision rather than inheriting this value.

All shown frontmatter fields are required. IDs, actor, and ticketPath are nonempty strings; IDs are case-sensitive and matched using the same effective trimmed identity as the current record readers. `observedAt` is an RFC 3339 timestamp. `predecessors` is an array of distinct UUID strings and may be empty. Require lowercase, hyphenated UUID spelling for observation IDs and lowercase 64-character hexadecimal SHA-256 digests. `checkoutRevision` is the only nullable field in this note profile. Reject duplicate YAML keys throughout the frontmatter. Required body headings may each appear once; duplicate headings are invalid rather than silently merged.

Required level-two body sections are `Approach`, `Completed`, `Remaining`, `Checks`, `Questions`, `Failed approaches`, and `Next step`. Require the headings; allow empty content or `None` to report nothing recorded. Empty content does not establish completion. Preserve the original body in the report. Existing authored requirements and accepted evidence stay outside the cache.

Under `Checks`, skills record attempted command or procedure, result, tested code identity, environment, and an evidence reference when available. This section is authored prose in version 1. The CLI returns it as reported observations, not machine-certified criterion coverage. Missing attribution is visible for agent review; do not claim an automatic semantic detector can establish whether every sentence is supported.

`Questions` can retain nonblocking exploratory questions. Blocking questions must also be recorded against authoritative work records. Settled decisions and accepted evidence are promoted to their authoritative homes and linked from the note.

Reject duplicate YAML keys, unsupported versions, wrong known-field types, mismatching IDs, duplicate predecessors, self-links, and invalid UUIDs. Preserve unknown metadata for inspection without treating it as instructions. The observation directory UUID and frontmatter UUID must agree.

### Retained requirement content

`context.json` retains the exact version-1 output bytes of a task-context collection used by the session. This includes source text, whole-file digests, reasons, and collection diagnostics, including any uncommitted text. Skills redirect the reader output or copy the retained output bytes; they must not reconstruct snapshots from a model's recollection.

Verify the file digest against `contextSHA256`, validate the context schema and each source's text digest, and confirm that its root WorkItem identity matches the note. An incomplete collection is permitted as an explicitly incomplete observation. It cannot support a complete before/after comparison.

The collected source set contains the root ticket, selected Spec/Context documents, and recursively selected blockers under existing reader rules. It is not the entire repository. Blocked-by decision state is refreshed through orientation; the context reader still does not automatically follow those links. If the full decision text is required, the ticket should explicitly select it as Context under the existing convention.

Old source paths are descriptive, never authority to read an arbitrary current path. Match a retained path to a currently selected, authorized source before comparing. For the root ticket, stable identity permits an explicit moved-root match. Other moved documents appear as removed/added unless a future identity contract is agreed. Do not infer renames from equal content.

### Skill publication and competing notes

Skills create a unique observation directory, write `context.json`, and publish `note.md` last through a same-directory temporary file and rename. Never overwrite an existing finalized observation. Record the candidate IDs actually read and incorporated into the session's work as its predecessors. Re-reading candidates before publication can reveal another session's note, but does not authorize superseding it. Include that new candidate only after deliberately reconciling its observations; otherwise publish a competing successor of the original baseline.

Retain the requirement collection actually used for the work described by the checkpoint. Do not fetch newer requirements just before publication and silently attach them as if the session had used them. After refreshing and evaluating changed requirements, a subsequent checkpoint can establish that new baseline. The note's observation time and any check's tested revision remain separately attributed.

A checkpoint normally supersedes one current observation. Two sessions may each publish successors of the same predecessor; both remain current candidates. No current-pointer file, latest-timestamp rule, or exclusive claim is introduced. Directory creation must fail on an existing UUID, and the skill retries with a new UUID. These conventions preserve cooperative writers; they cannot protect against arbitrary manual deletion or edits outside the convention.

Multiple valid leaf observations mean a conflict, even if their prose happens to agree. A deliberate reconciliation publishes another observation referencing all candidates it reconciles, after inspecting code, requirements, and evidence. The CLI never performs this reconciliation. Preserve the older observations during this milestone; automatic pruning is deferred.

A leaf is an observation not named as a predecessor by any other valid observation in the selected task graph. Validate the entire observed graph before selecting leaves, without using timestamps as graph edges. An empty predecessor array creates a root; multiple roots are permitted but normally lead to multiple current candidates until explicitly reconciled.

An observation directory without a finalized `note.md` is an interrupted or in-progress write. Report it as incomplete rather than treating the previous leaf as an unquestioned current account. Invalid notes, missing predecessor targets, cycles, or duplicate IDs leave candidate selection incomplete. Retain readable candidates and diagnostics; do not silently pick a valid-looking leaf from a partially observed graph.

## Bounded and authorized reading

Current context and orientation retain the existing defaults of 100 files and 1,048,576 source bytes. Within `resume`, they share one capture of each current source's bytes and one collection budget; parsing, hashes, readiness, and source output must use that captured content. Each source is counted once. This is not a filesystem-wide atomic snapshot. Cancellation and changed/unavailable sources produce explicit partial results or operation failures under existing reader conventions.

Collect the Project manifest first for identity, then the task context in its existing traversal order, then orientation's remaining inventory and selected sources in its existing order. Reuse already captured bytes at each step. Reaching the shared limit can leave useful task context alongside incomplete orientation; preserve both results and their separate completeness flags. Do not suppress unknown project checks just because task context was complete.

Cache reads have an independent default budget of 200 files and 4,194,304 bytes, configurable through the cache-limit flags. Count directory entries considered during observation discovery against the cache-file limit as well as files read, so malformed caches cannot cause unbounded enumeration. Stop at the first breach, return available data and pending-path diagnostics, and suppress a unique-current claim when the candidate inventory is incomplete. Do not truncate a source body and then present it as complete.

Count each resolved cache entry once; reading an enumerated file does not count it twice. Discover observation directories in lexical UUID order and inspect `note.md` before `context.json` within each directory. Directory entries count toward the entry limit; only file content contributes to the byte limit. Read complete note metadata to determine the graph, but load snapshot bodies only for current candidates. Older snapshots remain retained without being eagerly read. Validate a candidate's snapshot only after complete graph discovery; if discovery is incomplete, report candidate status as unknown.

Read only the selected project/task namespace. Resolve physical paths and require the cache root to stay beneath the selected working directory, and all observation files to stay beneath that cache root. Reject escaping symlinks, directory aliases, and path traversal. `contextFile` is exactly `context.json` in version 1. Do not follow arbitrary links in note prose or fetch remote evidence URLs. The report preserves them as references.

Authorization for current sources remains the selected bundle and explicit/configured allowed roots. A cached copy does not expand those roots. Retained source text outside today's authorized roots is withheld with diagnostics; cache metadata cannot authorize its own sources. Compare only authorized selected current sources. Historical checkout paths that no longer resolve cannot establish an authorized source match; report the unavailable comparison and reconstruct from current records.

Missing cache or task namespace is an ordinary absence. Unreadable or malformed cache data is different: preserve current facts but mark recovery inspection incomplete. No read command creates cache directories, rewrites notes, repairs snapshots, updates Git ignore files, or changes records.

The adoption instructions keep `.context-cache/` out of Git history using an existing ignore convention or a reviewed ignore entry. Cache disposal and ignoring are separate from tracking authoritative project records. Non-Git projects need no ignore setup.

## Resumption result

JSON is a new version-1 object with `kind: "task-resumption"`. It does not change the version-1 context or orientation formats.

| Field | Meaning |
| --- | --- |
| `schemaVersion`, `kind` | `1`, `task-resumption` |
| `complete` | All selected current facts and recovery observations could be evaluated within the limits |
| `scope` | Canonical records directory, working directory, cache root, project/task IDs where known |
| `orientation` | Full current project orientation result |
| `context` | Full current task-context result |
| `recovery` | Status, inventory completeness, candidate observations, and earlier-observation references |
| `comparison` | Baseline availability, completeness, and source differences for each candidate |
| `diagnostics` | Attributed codes, severity, messages, affected paths, and relevant observation IDs |

Use arrays for lists and null for unavailable scalar values. Return the original note body and metadata for every readable current candidate, with its source path and digest. Preserve authored check statements as observations; expose current structured Acceptance through `orientation`. Do not turn arbitrary check prose into a trusted pass/fail result.

`recovery.status` is `absent`, `available`, `conflicting`, or `unknown`. A unique valid leaf in a complete valid graph is available. Several valid leaves in a complete graph are conflicting. Invalid or incomplete graph inspection is unknown, even when candidate observations can be shown.

`comparison` identifies each candidate separately. Each source entry reports old/current source references and digests, status `unchanged`, `changed`, `added`, `removed`, or `unknown`, and authorized previous/current text when present. Return whole text; the text renderer may show a deterministic line diff. Preserve exact input bytes for hashing. Changes to metadata or comments may alter whole-source comparison even when an existing requirement fingerprint stays unchanged; label this as source text change, not necessarily changed requirements.

Only report `added` or `removed` when both selected source sets were collected completely. Otherwise an unmatched source is `unknown`: omitted information cannot establish absence. A source no longer selected is removed from task context, not necessarily deleted from disk. Order entries by current task-context source order, then unmatched retained source order. Emit one comparison per current candidate in lexical observation-ID order; do not merge competing baselines.

If there is no baseline, `comparison` reports `baselineAvailable: false`, with no assertion that current files are unchanged. This expected absence does not by itself make the command partial. A selected baseline that is corrupt, withheld, missing, or incomplete does make comparison incomplete and the report partial. A fully known conflict is complete information about an ambiguity, not permission to choose a winner.

The default text shows project/task identity, checkout and completeness, note status, observed progress, source changes or comparison limits, current blockers, reported checks and verification gaps, and source pointers. Distinguish authored historical observations from current checked facts. Use wording such as “reported checks” and “completion not established by this note.” Do not infer unrecorded verification gaps beyond unavailable support and the existing acceptance facts; skills assess the substantive evidence.

## Exit statuses and diagnostics

| Status | Meaning |
| --- | --- |
| `0` | Complete evaluation, including known blockers, no recovery note, or fully observed competing notes |
| `1` | Partial report: missing current source, invalid or incomplete recovery data, unavailable identity, or limit breach |
| `2` | Invalid flags/scope, unavailable selected working directory, cancellation, or operation/output failure |

Reports and data diagnostics go to stdout; invocation and operation failures go to stderr. Neither status 0 nor `complete: true` means work is authorized, ready, accepted, or safe to implement without reading the report.

Reuse reader/discovery diagnostic codes for existing failures. New data codes are `recovery_invalid_note`, `recovery_identity_mismatch`, `recovery_incomplete_observation`, `recovery_snapshot_invalid`, `recovery_snapshot_outside_scope`, `recovery_predecessor_missing`, `recovery_cycle`, `recovery_conflict`, `recovery_limit_exceeded`, and `recovery_source_omitted`. Missing cache is a status, not an error. A fully known conflict is a warning. Validation, source authorization, and collection failures are errors causing a partial report. Message wording is not a stable machine contract.

## Ownership and integration

The Go application operation composes current project facts and recovery inspection without printing, prompting, or writing. The CLI resolves scope and renders output. Share source capture and existing evaluation operations rather than duplicating acceptance or readiness logic. A focused recovery module owns cache validation, observation graph selection, and source comparison; it does not own task selection policy or agent execution.

Update the workflow skill/instructions to obtain current task context, retain its exact bytes, maintain recovery checkpoints after meaningful progress, and apply the agreed continuation behavior. Fix examples that still use the old direct `--project` meaning when touching the task-context skill; the current [discovery contract](../../docs/discovery.md) requires `--bundle` for direct records access.

Working notes can help an agent recover after compaction but cannot guarantee an update immediately before an unexpected interruption. State what was observed and when; do not claim complete session history. CLI reports establish what the tool returned, not what a model consumed or understood.

## Acceptance and implementation checks

Run the [agreed real-task trial](issues/01-usable-local-loop.md#answer). Retain the task reference, invocation, current reader source identity, interruption point, source changes, resumed behavior, and observed verification gaps. Confirm the user did not have to reconstruct the previous session. Do not manufacture a performance target.

Required behavioral checks:

- Compact orientation, expanded orientation, and JSON agree on underlying facts. Grouped blockers retain affected commitments, order, sources, and all incomplete/unknown conditions.
- Existing context JSON and selection behavior remain compatible. Existing workspace navigation and setup continue to work.
- One complete recovery candidate, absent notes, competing candidates, missing baseline, missing current source, and an open blocking decision produce the specified distinct reports.
- A retained uncommitted requirement change exposes the actual previous and current text. Metadata-only change is not automatically labeled a semantic requirement change.
- Separate checkouts and projects with identical task IDs do not share notes accidentally. A project with a separate record store and an independent non-Git project both work.
- Interrupted publication, corrupt hashes, malformed metadata, dangling/cyclic predecessors, unreadable files, unauthorized old sources, escaping symlinks, and exhausted current/cache budgets remain explicit and bounded.
- Cooperative concurrent checkpoint creation preserves both branches. Reconciliation is explicit. No timestamp-based winner or silent lost update is introduced.
- Local test observations remain distinct from device/runtime verification and current acceptance. Historical tested revisions are not invalidated merely because Git HEAD advanced.
- Deleting the cache leaves project knowledge and acceptance records intact. Read operations leave records, cache, configuration, and Git state unchanged.

Use real temporary filesystem fixtures for cache and scope behavior, unit checks for graph and comparison rules, and CLI contract fixtures for output/exit statuses. Run the repository's Go test, race, vet, and build checks when implementing. This planning session does not assert that any new checks have passed.

## Delivery sequence

1. Implement compact orientation and the explicit detail option over existing results.
2. Implement bounded recovery observation reading, identity/scope validation, and graph selection.
3. Compose current context/orientation with retained-source comparison and expose `resume` text/JSON.
4. Update skills for checkpoint authoring and continuation, then run the real-task trial and retain acceptance evidence.

These are implementation boundaries, not new tracker commitments. Create authoritative implementation tickets from this specification during implementation planning. General record writers, implementation claims, remote synchronization, automatic cache cleanup, and a custom runtime remain outside this milestone.
