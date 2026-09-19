# Continue an interrupted task

Use `ctx resume` when you have a project and ticket reference but do not have
the earlier session's conversation. It refreshes current task context and
project orientation, then inspects the selected checkout's recovery notes. The
command only reads files; it does not claim work, change a record, or select a
winner among conflicting notes.

Build the development executable from the repository root:

```sh
go build -o /tmp/ctx ./cmd/ctx
```

Start by orienting yourself. The default report is the compact project view.
Use `--detail` for every evaluated work item and `--json` when an agent or a
script needs the structured report. The compact view shows its record-store
root once. Paths beneath that root are relative; full provenance is available
in the detailed and JSON reports.

```sh
/tmp/ctx orient --max-files 500 --max-bytes 8388608
/tmp/ctx orient --detail --max-files 500 --max-bytes 8388608
/tmp/ctx orient --json --max-files 500 --max-bytes 8388608
```

This repository has discovery configured. For another project, use `--project`
to select its directory or registered alias. See [Discovery reference](discovery.md)
for selection rules. When discovery is not configured, select the record store
directly. The ticket path is relative to that store. A direct `--bundle` invocation also needs the
checkout whose cache will be inspected. Repeat `--allow-source` for each
directory that contains an authored linked document outside the record store.
Authorization permits a linked document to be read; it does not cause the
reader to include every file in that directory.

```sh
/tmp/ctx resume \
	--bundle /work/records \
	--checkout /work/application \
	--ticket feature/issues/07-verify-workflow.md \
	--allow-source /work/application/docs
```

For this repository, run the following command from its root. The linked
documents live under that root, and accumulated acceptance evidence requires
more than the default source limits. Recovery status depends on the notes
present in your checkout.

```sh
/tmp/ctx resume \
	--bundle .scratch/records \
	--checkout . \
	--ticket cli-wayfinding/issues/07-verify-workflow.md \
	--allow-source . \
	--max-files 500 \
	--max-bytes 8388608 \
	--json
```

The defaults are 100 current-source files, 1 MiB of current-source bytes, 200
cache entries, and 4 MiB of cache bytes. Use the four limit flags only for the
scope you are inspecting:

```sh
/tmp/ctx resume --ticket feature/issues/07-verify-workflow.md \
	--max-files 500 --max-bytes 8388608 \
	--max-cache-files 400 --max-cache-bytes 8388608
```

Inspect the diagnostics before raising a limit. A partial inventory does not
show a complete set of notes or work records. Do not treat an override used
for one large project as a new default.

## Read the report before continuing

`resume` returns four distinct sets of facts: current task context, current
project orientation, recovery observations, and comparisons with each selected
checkpoint. A complete task context does not make the project orientation or
recovery comparison complete. Likewise, an absent recovery state means that no
note was found in this checkout's task namespace; it does not mean the task is
finished or unchanged.

The readable report is for a person continuing the task. `--json` includes the
same facts and makes the boundaries explicit: `context`, `orientation`,
`recovery`, `comparison`, and top-level `complete`. Use JSON for an agent that
needs source paths, digests, candidate IDs, and diagnostics without scraping
the text report.

Exit status `0` means the report is complete, including known blockers, absent
notes, or fully inspected conflicts. Status `1` means some current source,
note, comparison, identity, or bounded inventory is incomplete. Status `2`
means the invocation or selected scope could not be used. Neither status `0`
nor `complete: true` establishes work readiness, acceptance, or permission to
change a task.

When a checkpoint is available, read its `Completed`, `Remaining`, `Checks`,
`Questions`, and `Next step` sections together with the current comparison.
Treat reported checks as historical observations. Check the tested source
identity and retained evidence before relying on them. Re-run a check when the
current sources or the relevant environment have changed. Keep missing
device or runtime evidence explicit. A backend check does not establish a
parent device outcome.

State the completed, remaining, and changed work before continuing. Resolve
current blockers using authoritative records, then name the next action. When
notes are absent or incomplete, reconstruct only what current requirements,
files, and actual verification support.

## Publish an accurate checkpoint

Recovery notes retain the task context that governed the work, rather than a
newly collected snapshot substituted at the end of the session. Capture the
exact JSON bytes before making the documented progress:

```sh
work_dir=$(mktemp -d)
go build -o "$work_dir/ctx" ./cmd/ctx
"$work_dir/ctx" context \
	--bundle /work/records \
	--ticket feature/issues/07-verify-workflow.md \
	--allow-source /work/application/docs \
	> "$work_dir/context-used.json"
```

Use the same authorized roots as the work. Preserve `context-used.json`
unchanged, record the command's exit status, and do not replace it with a
later collection merely because requirements changed. Refresh first when
requirements may have changed; a later checkpoint can then describe work done
under that refreshed context.

The checkpoint writer creates a new observation under the selected checkout's
`.context-cache/resume-v1/` namespace. Its `context.json` is an exact copy of
the captured output and its note records the context file's SHA-256, task and
project IDs, checkout provenance, completed and remaining work, actual checks,
gaps, and one next action. Publish `note.md` last by renaming a unique
same-directory temporary file. The repository's
[recovery-note workflow](../.agents/skills/recovery-notes/SKILL.md) and
[RecoveryNote profile](../.agents/skills/recovery-notes/PROFILE.md) define the
required fields and the safe publication procedure.

Do not put accepted evidence or a blocking decision only in the cache. Record
those in their authoritative project records and link them from the checkpoint.
Keep `.context-cache/` out of Git. Non-Git projects need no ignore rule. The
cache is disposable working material, not the project record.

## Handle conflicts and deliberate recovery

`resume` never picks the newest checkpoint by timestamp. If it reports more
than one candidate, read every candidate and the current sources, then publish
a reconciliation note whose `predecessors` names every account actually
incorporated. Until then, the competing leaves remain valid alternatives.

An observation directory without a final `note.md` is an interrupted or active
publication. Do not retire it because it is old. First establish that its
writer stopped, re-run `resume`, and recheck that no finalized note refers to
that observation. Only then may the recovery workflow move the whole directory
to a fresh `quarantine/` location and write its retained recovery record.

When recovery history exceeds its bounded inspection limits, inspect the full
history with explicit cache-limit overrides. A deliberate reset requires all
writers for that task to have stopped. It moves the entire `observations/`
directory to a fresh quarantine location, preserves a recovery record, and
then starts a new root checkpoint. Removing `.context-cache/` also preserves
authoritative requirements, blockers, decisions, and accepted evidence, but it
removes the convenient continuation history for that checkout.

For the exact non-overwrite checks and stopped-writer requirements, follow the
[recovery-note workflow](../.agents/skills/recovery-notes/SKILL.md). The CLI
reports these states; it does not quarantine observations, reset history, or
write a checkpoint itself.
