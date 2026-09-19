# Local orientation and resumption

Status: specified for implementation after [independent review and reassessment](independent-review.md#reassessment-on-2026-09-17). The contract fixes cover bounded discovery, interrupted-publication recovery, and explicit nested JSON shapes. No implementation or acceptance trial has run.

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

1. Project title, the selected record-store root, authored goals, and evaluation/inventory completeness. Do not generate a prose summary of authored goals; preserve their text. The root establishes the base for displayed record paths.
2. Current commitments in authored order, with execution and readiness shown separately.
3. Needs attention: shared blocking decisions, prerequisite failures, missing required sources, and unknown checks, with affected commitments and source references.
4. In-progress work and the existing shortlist for new pickup, explicitly distinguished.
5. Guidance to obtain task context, resumption, or expanded detail. Full project-reference listings and routine source/relationship provenance belong in detail and JSON, not the compact overview.

The user refined the operator presentation after the live CLI demonstration on 2026-09-19. Show the record-store root once. Use root-relative paths for actionable records and findings inside that store, preserving directories and same-filename distinctions. Keep external paths explicit and preserve original paths when a reliable root is unavailable. Do not guess a Git root or use cwd as a substitute. Omit repeated generated project.md source lines, routine from/link annotations, and path repetition in affected-work lists when title and ID identify the work. Preserve enough referring-path/link detail to locate unresolved or ambiguous relationships. These are display rules only; full detail and JSON retain exact paths, provenance, and values.

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

### Recovery from abandoned publication and history limits

The skill may retire an unfinished observation only after establishing that its writer has stopped, for example through the owning session's completed/terminated state or an explicit human confirmation that it is no longer running. Age, missing heartbeats, or absence of `note.md` alone is insufficient. If writer state cannot be established, leave the directory intact, report uncertainty, and continue independent reconstruction without declaring a unique current note.

After establishing that the writer stopped, recheck that `note.md` is still absent and that no finalized note names this observation as a predecessor. Move the whole unfinished directory to `<task-namespace>/quarantine/<fresh-uuid>/` without replacing any destination. Retain a small `recovery.md` beside it describing its original ID, actor, time, reason, and basis for establishing writer termination. The observations reader does not traverse quarantine. If a finalized note or an inbound predecessor link is discovered, stop this narrow recovery procedure; do not silently remove a finalized or referenced observation.

Refresh `resume` after quarantine. The previous valid leaves can now be evaluated normally; a new observation names the leaves actually used. A discarded partial publication is never named as a predecessor. This is a skill-directed filesystem procedure, not a new CLI mutation or work claim. It relies on confirmed stopped writers, not a distributed locking guarantee.

Ordinary checkpoint accumulation can exceed the default entry or byte budgets. First offer the explicit limit overrides so all notes can be inspected. For a bounded fresh start, stop all writers for this task and explicitly choose a task-cache reset. Move the entire `observations/` directory to a new uniquely named directory under `quarantine/`; retain a recovery record. Do not select a winner among conflicting notes or delete individual predecessors. Reconstruct against current requirements and publish a new root with an empty predecessor list. State that continuity with the old graph is no longer automatically evaluated and that previous notes remain only in the quarantined archive. This reset can lose convenient comparison and continuation state, but never removes authoritative records or accepted evidence.

The reader ignores quarantine, performs neither procedure automatically, and does not infer that a source is safe to remove. A reset or quarantine must abort if the stopped-writer precondition cannot be established. Automatic pruning and archive restoration are outside this milestone.

## Bounded and authorized reading

Current context and orientation retain the existing defaults of 100 files and 1,048,576 source bytes. Within `resume`, they share one capture of each current source's bytes and one collection budget; parsing, hashes, readiness, and source output must use that captured content. Each source is counted once. This is not a filesystem-wide atomic snapshot. Cancellation and changed/unavailable sources produce explicit partial results or operation failures under existing reader conventions.

Collect the Project manifest first for identity, then the task context in its existing traversal order, then orientation's remaining inventory and selected sources in its existing order. Reuse already captured bytes at each step. Reaching the shared limit can leave useful task context alongside incomplete orientation; preserve both results and their separate completeness flags. Do not suppress unknown project checks just because task context was complete.

Cache reads have an independent default budget of 200 files and 4,194,304 bytes, configurable through the cache-limit flags. Count directory entries considered during observation discovery against the cache-file limit as well as files read, so malformed caches cannot cause unbounded enumeration. Stop at the first breach, return available data and pending-path diagnostics, and suppress a unique-current claim when the candidate inventory is incomplete. Do not truncate a source body and then present it as complete.

Count each resolved cache entry once; reading an enumerated file does not count it twice. Enumerate `observations/` with bounded directory reads, requesting no more than the remaining entry allowance plus one lookahead entry. That one entry detects a limit breach; it is not included as inspected data. Never read the entire directory merely to sort it. Sort only the observed names before processing them. If discovery completes, processing is lexical UUID order. If it is partial, the sorted observed subset is not promised to be the globally first lexical subset and may vary across runs. Report that limit and keep graph selection unknown.

Process `note.md` before any candidate `context.json`. Count observation directory entries and each attempted fixed file path once; do not enumerate arbitrary files inside observations. Directory entries count toward the entry limit; only file content contributes to the byte limit. Use bounded file reads with at most one lookahead byte to detect oversized content. Read note metadata to determine the graph, but load snapshot bodies only for current candidates. Older snapshots remain retained without being eagerly read. Validate a candidate's snapshot only after complete graph discovery; if discovery is incomplete, return observed notes with no claim that they are current candidates. Quarantine is outside observation discovery and its entries do not consume this budget.

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

These statuses describe note selection, not snapshot validity. One selected note can be `available` while its retained snapshot is corrupt; the graph remains valid but comparison and the top-level report are incomplete. For incomplete or invalid graphs return no selected candidate IDs, retaining parsed observations and diagnostics for investigation.

`comparison` identifies each candidate separately. Each source entry reports old/current source references and digests, status `unchanged`, `changed`, `added`, `removed`, or `unknown`, and authorized previous/current text when present. Return whole text; the text renderer may show a deterministic line diff. Preserve exact input bytes for hashing. Changes to metadata or comments may alter whole-source comparison even when an existing requirement fingerprint stays unchanged; label this as source text change, not necessarily changed requirements.

Only report `added` or `removed` when both selected source sets were collected completely. Otherwise an unmatched source is `unknown`: omitted information cannot establish absence. A source no longer selected is removed from task context, not necessarily deleted from disk. Order entries by current task-context source order, then unmatched retained source order. Emit one comparison per current candidate in lexical observation-ID order; do not merge competing baselines.

If there is no baseline, `comparison` reports `baselineAvailable: false`, with no assertion that current files are unchanged. This expected absence does not by itself make the command partial. A selected baseline that is corrupt, withheld, missing, or incomplete does make comparison incomplete and the report partial. A fully known conflict is complete information about an ambiguity, not permission to choose a winner.

The default text shows project/task identity, checkout and completeness, note status, observed progress, source changes or comparison limits, current blockers, reported checks and verification gaps, and source pointers. Distinguish authored historical observations from current checked facts. Use wording such as “reported checks” and “completion not established by this note.” Do not infer unrecorded verification gaps beyond unavailable support and the existing acceptance facts; skills assess the substantive evidence.

### Nested JSON field contract

All fields in the following tables are required. `?` means a value may be JSON null, never an omitted property. Arrays are always arrays, including empty arrays. Unknown report fields may be added compatibly; consumers ignore unrecognized fields. A breaking meaning/type change requires a new schema version. Existing nested `orientation` and `context` objects retain their own version-1 schemas.

| Object | Fields and types |
| --- | --- |
| `scope` | `recordsDirectory: string`, `workingDirectory: string`, `cacheRoot: string`, `projectId: string?`, `taskId: string?` |
| `recovery` | `status: enum`, `inventoryComplete: boolean`, `graphStatus: enum`, `observations: Observation[]`, `candidates: string[]` |
| `Observation` | `id: string`, `projectId: string`, `taskId: string`, `observedAt: string`, `actor: string`, `predecessors: string[]`, `ticketPath: string`, `checkoutRevision: Revision?`, `source: FileReference`, `body: string`, `metadata: object`, `snapshot: SnapshotReport` |
| `Revision` | `origin: string`, `revision: string` |
| `FileReference` | `path: string`, `sha256: string?` |
| `SnapshotReport` | `path: string`, `recordedSHA256: string`, `observedSHA256: string?`, `status: enum`, `sourceCount: integer?` |
| `comparison` | `baselineAvailable: boolean`, `complete: boolean`, `candidates: CandidateComparison[]` |
| `CandidateComparison` | `observationId: string`, `baselineAvailable: boolean`, `complete: boolean`, `sources: SourceDifference[]` |
| `SourceDifference` | `status: enum`, `previous: ComparedSource?`, `current: ComparedSource?` |
| `ComparedSource` | `path: string`, `sha256: string?`, `text: string?`, `availability: enum` |
| `Diagnostic` | `code: string`, `severity: error or warning`, `message: string`, `path: string?`, `from: string?`, `link: string?`, `observationId: string?` |

`scope` paths are canonical absolute paths. `cacheRoot` is the derived cache location even when absent. `FileReference` identifies a note's captured bytes; `body` is its exact Markdown body after frontmatter removal. `metadata` preserves authored frontmatter, using the existing orientation JSON encoding for non-JSON YAML values, including tagged objects and non-string mappings. Required typed fields remain validated independently of the preserved metadata. Do not emit a fabricated Observation for invalid frontmatter; identify the unavailable observation by diagnostic path instead.

`recovery.observations` contains every successfully parsed, identity-matching note observed, ordered by ID. This preserves earlier observations without a second, ambiguous reference shape. `recovery.candidates` always refers to IDs in this array. Predecessors are preserved exactly as authored; they are guaranteed to resolve within this array only when `graphStatus` is `valid`. A dangling predecessor remains visible with an attributed diagnostic when the graph is invalid. `candidates` is empty when selection is unknown and otherwise contains all graph leaves in lexical ID order. `graphStatus` is `valid`, `incomplete`, `invalid`, or `not_evaluated`. A missing or empty observations directory gives `status: absent`, `inventoryComplete: true`, `graphStatus: valid`, and empty arrays. Missing project/task identity gives `status: unknown`, `inventoryComplete: false`, `graphStatus: not_evaluated`. A fully inspected malformed graph is `invalid`; inspection stopped by limits or unreadable/missing note files is `incomplete`. A known invalidity takes precedence if both occur. Neither state can establish selected candidates.

`SnapshotReport.status` is `not_loaded`, `valid`, `incomplete`, `invalid`, `unavailable`, or `withheld`. Older observations and notes without established candidacy use `not_loaded`. `observedSHA256` is null unless full snapshot bytes were read; `sourceCount` is null until schema validation succeeds, otherwise the recorded source count. `invalid` means schema/identity/digest validation failed; `unavailable` means required bytes could not be read; `withheld` means otherwise valid historical content is outside current authorization; `incomplete` means the retained reader result is partial; `valid` means a complete validated authorized snapshot. When conditions overlap, invalid takes precedence, then unavailable, withheld, incomplete, valid. The report returns no source text from an invalid snapshot.

`comparison.candidates` always contains one entry for each selected `recovery.candidates` ID, including entries whose snapshot failed. A CandidateComparison has `baselineAvailable: true` when at least one validated, authorized retained source is available for comparison; an incomplete snapshot may therefore have an available but incomplete baseline. It is complete only when retained and current collections are complete and every selected comparison is evaluable. Aggregate `baselineAvailable` is true if any candidate baseline is available. Aggregate `complete` requires complete current context, known recovery selection, and complete candidate comparisons. With no notes, current context can be complete while `baselineAvailable` is false and the candidates array is empty. These are observations of availability, not claims that files are unchanged.

`ComparedSource.availability` is `available`, `unavailable`, or `withheld`. Available entries contain exact text and its validated digest. Unavailable entries have null text and a digest only if an observed or retained digest is known. Withheld entries contain null text; their path/digest explain the withheld comparison but do not grant authority to read that source. A null side means absent from a completely known selection. When absence cannot be established, the difference is `unknown` and null may also denote an unobserved side; it must not be interpreted as deletion. `SourceDifference.status` follows the comparison rules above. Root relocation is one comparison with different paths and matched stable task identity, subject to current source authorization.

Top-level `diagnostics` is the normalized union of current-reader and recovery findings. Existing nested reader diagnostics remain unchanged. When promoting a current-reader diagnostic, set `observationId` to null and fill absent attribution fields with null. Cache findings use the implicated observation ID when known. Deduplicate only identical code, severity, path, from, link, and observation ID; messages do not determine identity. Order current-context findings first, then orientation-only findings, then cache findings in observed-path and code order.

Top-level `complete` requires `orientation.complete`, `context.complete`, `comparison.complete`, complete recovery inventory, and a valid recovery graph. Complete conflicting notes are compatible with `complete: true`. A graph-valid selected note with an invalid snapshot is not. `not_loaded` on a superseded snapshot is expected and does not make the report partial.

The [JSON contract examples](examples/resumption-results.json) contain five projections of this exact shape: absent notes, one available note, conflicting leaves, a selected note with an invalid snapshot, and incomplete inventory. They deliberately omit the unchanged `orientation` and `context` payloads; the examples declare those completeness inputs separately. They are input for implementation contract fixtures, not reports generated by existing code. Their digest fields illustrate encoding; implementation fixtures must supply actual source files and recompute the matching digests.

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
- Oversized-directory fixtures verify that discovery reads at most the remaining entry budget plus one lookahead entry before stopping; sorting occurs only over observed names. Partial discovery never claims a global lexical prefix or a selected candidate.
- A stopped writer's unfinished observation can be deliberately quarantined and followed by a complete resume without removing finalized competing notes. Unknown/live writers are not retired by age. History-limit reset preserves the entire old graph outside active discovery and makes reconstruction explicit.
- JSON fixtures cover absent, available, conflicting, invalid-snapshot, and incomplete-inventory cases with stable object shapes and nullability. Graph validity and snapshot validity remain separate.
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
