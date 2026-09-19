---
name: recovery-notes
description: Maintain or recover a task's local working notes through ctx resume. Use for interrupted work, recovery checkpoints, competing recovery notes, incomplete note publication, or an oversized task note history.
---

# Maintain recovery notes

Use this skill after reading the task with [task-context](../task-context/SKILL.md). Use the local [RecoveryNote profile](PROFILE.md) for note fields and cache layout. The ticket context remains the source for the work being continued.

## Limits

`ctx context`, `ctx orient`, and `ctx resume` only read. This skill directs cooperative filesystem work in `.context-cache/`; it does not provide an exclusive claim, control another writer, or prove that an agent understood the note. The report shows what the CLI read, and reported checks remain observations until accepted evidence establishes their result.

Keep `.context-cache/` out of Git. It is disposable working material. Deleting it preserves authoritative tickets, decisions, evidence, and acceptance records, but removes retained comparisons and automatic continuation history. Non-Git projects do not need an ignore rule.

Do not retire a note automatically. Age, an absent heartbeat, and a missing `note.md` do not establish that its writer stopped.

## Start or continue work

1. Build the current `ctx` binary in a temporary location. Resolve the selected bundle, checkout, ticket path, and authorized source roots explicitly. Preserve a narrower scope and any limits supplied by the caller. When a current-source limit makes the report partial, inspect its diagnostics, then use only the explicit `--max-files` or `--max-bytes` override justified by that scope; do not present an override as a new default.
2. Capture the exact task-context JSON that this session reads. With direct records access, use:

   ```sh
   <temporary-ctx> context --bundle <records-directory> \
     --ticket <ticket-path> --allow-source <authorized-directory> \
     > <temporary-context-used.json>
   ```

   Repeat `--allow-source` for each authorized root. Keep the output bytes unchanged, read those bytes as the session's task context, and retain the exit status. An incomplete result can be retained, but cannot support a complete comparison.
3. Inspect the current state before acting:

   ```sh
   <temporary-ctx> resume --bundle <records-directory> \
     --checkout <working-directory> --ticket <ticket-path> \
     --allow-source <authorized-directory> --json
   ```

   The direct `--bundle` form always needs `--checkout`. Read current context and orientation, recovery status, candidates, comparisons, and diagnostics. The checkpoint is historical; refresh requirements and current files before continuing.
4. If recovery is absent, reconstruct from the current ticket context, current checkout, and verification results. If it is conflicting, inspect every candidate's reported work, the current code, requirements, and evidence before choosing a next action. Never choose by timestamp or treat a reported check as current evidence. If recovery is unknown or partial, keep the uncertainty visible and continue only work supported by current facts.

Keep exploratory questions in the note. For a blocking unresolved question, use the [issue-tracker profile](../../../docs/agents/issue-tracker.md): create an open Decision record, link it from the affected WorkItem's `Blocked by decisions` section, and select it through Spec or Context when its answer imposes requirements. This keeps the blocker visible after cache disposal. Move settled decisions to that record. Retain accepted evidence as an immutable document outside the cache, then link it from an Acceptance record using [Record acceptance](../../../docs/agents/acceptance.md); link those records from the next note.

## Publish a checkpoint

Checkpoint after a meaningful decision, completed step, verification result, failed approach, or clear next action. Retain the context file from step 2, rather than collecting newer requirements just before publication. Refresh and assess changed requirements first; a later checkpoint can use that new collection.

Read the [RecoveryNote profile](PROFILE.md) before creating a note. It defines the namespace, typed fields, and template. Create a new random lowercase UUID directory under that task namespace:

```text
<checkout>/.context-cache/resume-v1/<sha256-project-id>/<sha256-task-id>/
  observations/<new-observation-uuid>/
    context.json
    note.md
```

Generate another UUID when directory creation reports that it already exists. Copy the exact `temporary-context-used.json` bytes to `context.json`, calculate its lowercase SHA-256, and verify the copied bytes before publishing the note.

Write a version-1 `RecoveryNote` with the exact required fields from the selected task context: `type`, `version`, `id`, `projectId`, `taskId`, `observedAt`, `actor`, `predecessors`, `ticketPath`, `checkoutRevision`, `contextFile`, and `contextSHA256`. Use the required body sections exactly once: `Approach`, `Completed`, `Remaining`, `Checks`, `Questions`, `Failed approaches`, and `Next step`.

State what happened and what remains. Under `Checks`, record the command or procedure, result, tested code identity, environment, and evidence reference when available. Keep failed approaches and unanswered questions concrete. Put only candidate IDs whose accounts this session actually read and incorporated in `predecessors`; discovery alone does not incorporate a candidate.

Publish `note.md` last. Write it to a unique temporary name in that observation directory, then rename it to `note.md` in the same directory. Refuse to replace an existing finalized `note.md`. A normal checkpoint names the one candidate it supersedes. When two writers publish from one predecessor, preserve both successors as candidates. A reconciliation follows inspection of all competing accounts, current code, requirements, and evidence, then publishes a new checkpoint naming every candidate it incorporated.

Run `ctx resume` again and retain the command, exit status, relevant JSON fields, and the cache paths observed. A checkpoint records continuity; it does not establish readiness, acceptance, or exclusive ownership.

## Handle an unfinished publication

An observation directory without a finalized `note.md` remains uncertain. First run `ctx resume` and preserve the incomplete result. Then establish that the writer stopped from the owning session's completed or terminated state, or from explicit human confirmation. When the state is unknown or the writer may still be live, leave every cache file unchanged, report that the procedure is refused, and reconstruct independently from current records.

After confirmed termination, recheck all of these facts immediately before moving anything:

1. The observation still has no `note.md`.
2. No finalized note names that observation as a predecessor.
3. The quarantine destination has a fresh UUID and does not exist.

Move the whole unfinished observation directory to `quarantine/<fresh-uuid>/` without replacing a destination. Write a no-overwrite `recovery.md` in that archive beside its retained cache files. Record the original observation ID, actor when available, observed time, reason, and the evidence establishing writer termination. Refresh `ctx resume`. If either recheck discovers a finalized or referenced observation, stop and preserve the directory; do not quarantine it through this procedure.

The reader excludes quarantine. A later checkpoint uses only the valid candidates actually incorporated; it never names the discarded partial publication.

## Recover an oversized history

First run `ctx resume` with explicit larger `--max-cache-files` and `--max-cache-bytes` values so the entire graph can be inspected. Limits do not authorize cleanup.

For a deliberate fresh start, establish that every writer for this task has stopped. If that cannot be established, leave the history intact. Recheck the stopped-writer condition, create a fresh nonexisting quarantine UUID, and move the entire `observations/` directory to `quarantine/<fresh-uuid>/` without selecting a winner or deleting individual notes. Write a no-overwrite `recovery.md` in the archive. It identifies the reset, archive path, time, actor, reason, and stopped-writer basis.

Then reconstruct from current requirements and publish a new root checkpoint with `predecessors: []`. State that automatic continuity with the old graph is lost and that the old notes remain in the quarantined archive. Reset never removes authoritative project records or accepted evidence.
