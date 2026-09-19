#!/usr/bin/env bash
set -euo pipefail

# Exercise the recovery-notes procedure using only a disposable fixture cache.
# Usage: 06-recovery-instructions-trial.sh /path/to/ctx /path/to/results

ctx_bin=${1:?pass the built ctx binary}
results=${2:?pass an empty results directory}
mkdir -p "$results"

fixture=$(mktemp -d "${TMPDIR:-/tmp}/context-recovery-06.XXXXXX")
records="$fixture/records"
checkout="$fixture/checkout"
unknown_checkout="$fixture/unknown-checkout"
mkdir -p "$records/issues" "$checkout" "$unknown_checkout"

cat > "$records/project.md" <<'EOF'
---
type: Project
id: recovery-demo-project
title: Recovery demo
---

## Goals

Keep task history recoverable.

## Current commitments

- [Demo task](issues/task.md)

## Open decisions

None
EOF

cat > "$records/issues/task.md" <<'EOF'
---
type: WorkItem
id: recovery-demo-task
title: Recovery demo task
triage: ready-for-agent
execution: in-progress
---

# Recovery demo task

## Acceptance criteria

- [ ] Continue from current facts.

## Blocked by

None

## Blocked by decisions

None
EOF

context_used="$fixture/context-used.json"
"$ctx_bin" context --bundle "$records" --ticket issues/task.md > "$context_used"
initial_context_sha=$(shasum -a 256 "$context_used" | awk '{print $1}')
project_key=$(printf %s recovery-demo-project | shasum -a 256 | awk '{print $1}')
task_key=$(printf %s recovery-demo-task | shasum -a 256 | awk '{print $1}')
namespace="$checkout/.context-cache/resume-v1/$project_key/$task_key"
observations="$namespace/observations"
quarantine="$namespace/quarantine"
unknown_namespace="$unknown_checkout/.context-cache/resume-v1/$project_key/$task_key"
unknown_observations="$unknown_namespace/observations"

resume() {
  local label=$1
  local working_directory=${2:-$checkout}
  set +e
  "$ctx_bin" resume --bundle "$records" --checkout "$working_directory" --ticket issues/task.md --json > "$results/$label.json" 2> "$results/$label.stderr"
  local status=$?
  set -e
  printf '%s\n' "$status" > "$results/$label.exit"
}

expect_exit() {
  local label=$1
  local expected=$2
  test "$(cat "$results/$label.exit")" = "$expected"
}

snapshot_cache() {
  local root=$1
  local destination=$2
  if test ! -e "$root"; then
    printf '<absent>\n' > "$destination"
    return
  fi
  while IFS= read -r -d '' path; do
    shasum -a 256 "$path"
  done < <(find "$root" -type f -print0) | LC_ALL=C sort > "$destination"
}

write_no_overwrite() {
  local destination=$1
  shift
  test ! -e "$destination"
  local temporary
  temporary=$(mktemp "${destination}.XXXXXX")
  printf '%s\n' "$@" > "$temporary"
  ln "$temporary" "$destination"
  rm "$temporary"
}

write_note() {
  local id=$1
  local predecessors=$2
  local actor=$3
  local observation="$observations/$id"
  mkdir -p "$observations"
  mkdir "$observation" || return 17
  cp "$context_used" "$observation/context.json"
  local digest
  digest=$(shasum -a 256 "$observation/context.json" | awk '{print $1}')
  local temporary_note
  temporary_note=$(mktemp "$observation/note.md.XXXXXX")
  cat > "$temporary_note" <<EOF
---
type: RecoveryNote
version: 1
id: $id
projectId: recovery-demo-project
taskId: recovery-demo-task
observedAt: '2026-09-19T12:00:00Z'
actor: $actor
predecessors: $predecessors
ticketPath: issues/task.md
checkoutRevision: null
contextFile: context.json
contextSHA256: $digest
---
## Approach

Inspect current task facts.

## Completed

Published this fixture checkpoint.

## Remaining

Continue from the reader report.

## Checks

Command: ctx context. Result: captured current context. Tested identity: fixture. Environment: temporary fixture. Evidence: $context_used.

## Questions

None

## Failed approaches

None

## Next step

Read ctx resume.
EOF
  test ! -e "$observation/note.md"
  mv -n "$temporary_note" "$observation/note.md"
  test ! -e "$temporary_note"
}

archive_record() {
  local archive_id=$1
  local kind=$2
  local original=$3
  local original_id=$4
  local actor=$5
  local observed_at=$6
  local reason=$7
  local stopped_writer_evidence=$8
  test ! -e "$quarantine/$archive_id"
  mkdir -p "$quarantine"
  mv "$original" "$quarantine/$archive_id"
  write_no_overwrite "$quarantine/$archive_id/recovery.md" \
    '# Recovery record' \
    '' \
    "Kind: $kind" \
    "Original observation ID: $original_id" \
    "Actor: $actor" \
    "Observed at: $observed_at" \
    "Reason: $reason" \
    "Stopped-writer evidence: $stopped_writer_evidence" \
    "Archive: $quarantine/$archive_id"
}

# An absent cache is reconstructed from current facts, without an unchanged claim.
resume absent

root=10000000-0000-4000-8000-000000000001
left=20000000-0000-4000-8000-000000000002
right=30000000-0000-4000-8000-000000000003
reconciled=40000000-0000-4000-8000-000000000004
interrupted=50000000-0000-4000-8000-000000000005
live=60000000-0000-4000-8000-000000000006
new_root=70000000-0000-4000-8000-000000000007
collision=80000000-0000-4000-8000-000000000008

# A finalized root retains the exact initial context bytes.
write_note "$root" '[]' session-root
resume checkpoint

# A changed current source requires a new exact collection before later work.
printf '\n## Comments\n\nUncommitted reassessment fixture.\n' >> "$records/issues/task.md"
resume changed-current
ticket_source_path=$(jq -r '.sources[0].path' "$context_used")
initial_ticket_sha=$(jq -r --arg path "$ticket_source_path" '.sources[] | select(.path == $path) | .sha256' "$context_used")
current_ticket_sha=$(jq -r --arg path "$ticket_source_path" '.context.sources[] | select(.path == $path) | .sha256' "$results/changed-current.json")
test "$initial_ticket_sha" != "$current_ticket_sha"
context_reassessed="$fixture/context-reassessed.json"
"$ctx_bin" context --bundle "$records" --ticket issues/task.md > "$context_reassessed"
jq -e --arg path "$ticket_source_path" --arg digest "$current_ticket_sha" '.sources[] | select(.path == $path and .sha256 == $digest)' "$context_reassessed" >/dev/null
context_used="$context_reassessed"
reassessed_context_sha=$(shasum -a 256 "$context_used" | awk '{print $1}')

# Two independent successors preserve both branches after reassessment.
write_note "$left" "[$root]" session-left
write_note "$right" "[$root]" session-right
root_snapshot_sha=$(shasum -a 256 "$observations/$root/context.json" | awk '{print $1}')
test "$root_snapshot_sha" = "$initial_context_sha"
test "$root_snapshot_sha" != "$reassessed_context_sha"
printf 'initial=%s\nroot-snapshot=%s\nreassessed=%s\n' "$initial_context_sha" "$root_snapshot_sha" "$reassessed_context_sha" > "$results/context-baseline.sha256"
resume competing

# Reconciliation incorporates both candidates and converges the graph.
write_note "$reconciled" "[$left, $right]" session-reconciler
resume reconciled

# Interrupted publication is unknown until confirmed-stop quarantine.
mkdir -p "$observations/$interrupted"
cp "$context_used" "$observations/$interrupted/context.json"
resume interrupted
test ! -e "$observations/$interrupted/note.md"
if find "$observations" -name note.md -type f -exec grep -l "$interrupted" {} + | grep -q .; then
  echo "interrupted observation became referenced" >&2
  exit 1
fi
archive_record 90000000-0000-4000-8000-000000000009 quarantine "$observations/$interrupted" "$interrupted" unavailable 2026-09-19T12:00:00Z 'unfinished publication after confirmed stop' 'fixture authoring step returned; no writer process remains'
resume quarantined

# Collision leaves the existing observation directory untouched and returns 17.
mkdir -p "$observations/$collision"
snapshot_cache "$namespace" "$results/collision-before.sha256"
set +e
write_note "$collision" '[]' session-collision 2> "$results/collision.stderr"
collision_status=$?
set -e
printf '%s\n' "$collision_status" > "$results/collision.exit"
test "$collision_status" = 17
snapshot_cache "$namespace" "$results/collision-after.sha256"
cmp "$results/collision-before.sha256" "$results/collision-after.sha256"
rmdir "$observations/$collision"

# Unknown writer state refuses retirement and leaves its entire task cache unchanged.
unknown=60000000-0000-4000-8000-000000000006
mkdir -p "$unknown_observations/$unknown"
cp "$context_used" "$unknown_observations/$unknown/context.json"
snapshot_cache "$unknown_namespace" "$results/unknown-before.sha256"
resume unknown-writer "$unknown_checkout"
expect_exit unknown-writer 1
printf '%s\n' 'refused: writer state is unknown; no cache mutation performed' > "$results/unknown-writer-refusal.txt"
snapshot_cache "$unknown_namespace" "$results/unknown-after.sha256"
cmp "$results/unknown-before.sha256" "$results/unknown-after.sha256"

# A live writer is refused while its owned process is alive.
mkdir -p "$observations/$live"
cp "$context_used" "$observations/$live/context.json"
(
  printf '%s\n' 'session-live owns this unfinished fixture observation' > "$observations/$live/writer-session"
  exec sleep 2
) &
live_pid=$!
while test ! -f "$observations/$live/writer-session"; do :; done
kill -0 "$live_pid"
printf 'pid=%s state=live\n' "$live_pid" > "$results/live-writer-state.txt"
snapshot_cache "$namespace" "$results/live-before.sha256"
resume live-writer
expect_exit live-writer 1
printf '%s\n' "refused: live fixture writer pid $live_pid is still running; no cache mutation performed" > "$results/live-writer-refusal.txt"
snapshot_cache "$namespace" "$results/live-after-refusal.sha256"
cmp "$results/live-before.sha256" "$results/live-after-refusal.sha256"
wait "$live_pid" || true
if kill -0 "$live_pid" 2>/dev/null; then
  echo 'live fixture writer did not stop' >&2
  exit 1
fi
printf 'pid=%s state=confirmed-stopped\n' "$live_pid" >> "$results/live-writer-state.txt"

# Confirmed stop permits quarantine, then a whole-history reset retains all history.
test ! -e "$observations/$live/note.md"
archive_record a0000000-0000-4000-8000-000000000010 quarantine "$observations/$live" "$live" unavailable 2026-09-19T12:00:00Z 'unfinished publication from live fixture writer' "fixture pid $live_pid exited normally and was reaped"
resume live-quarantined
expect_exit live-quarantined 0
archive_record b0000000-0000-4000-8000-000000000011 reset "$observations" whole-task-history 'multiple completed fixture sessions' 2026-09-19T12:00:00Z 'explicit whole-task fresh start after complete history inspection' "all fixture note authors completed; live pid $live_pid was confirmed stopped"
resume reset-absent
write_note "$new_root" '[]' session-reconstructed
resume reset-reconstructed

jq -e '.recovery.status == "absent" and .comparison.baselineAvailable == false and .complete == true' "$results/absent.json" >/dev/null
jq -e '[.comparison.candidates[].sources[].status] | index("changed") != null' "$results/changed-current.json" >/dev/null
jq -e '.recovery.status == "available" and .recovery.candidates == ["10000000-0000-4000-8000-000000000001"]' "$results/checkpoint.json" >/dev/null
jq -e '.recovery.status == "conflicting" and .recovery.candidates == ["20000000-0000-4000-8000-000000000002", "30000000-0000-4000-8000-000000000003"]' "$results/competing.json" >/dev/null
jq -e '.recovery.status == "available" and .recovery.candidates == ["40000000-0000-4000-8000-000000000004"]' "$results/reconciled.json" >/dev/null
jq -e '.recovery.status == "unknown" and .recovery.graphStatus == "incomplete"' "$results/interrupted.json" >/dev/null
jq -e '.recovery.status == "available" and .recovery.candidates == ["40000000-0000-4000-8000-000000000004"]' "$results/quarantined.json" >/dev/null
jq -e '.recovery.status == "unknown" and .recovery.graphStatus == "incomplete"' "$results/unknown-writer.json" >/dev/null
jq -e '.recovery.status == "unknown" and .recovery.graphStatus == "incomplete"' "$results/live-writer.json" >/dev/null
jq -e '.recovery.status == "available" and .recovery.candidates == ["40000000-0000-4000-8000-000000000004"]' "$results/live-quarantined.json" >/dev/null
jq -e '.recovery.status == "absent" and .comparison.baselineAvailable == false' "$results/reset-absent.json" >/dev/null
jq -e '.recovery.status == "available" and .recovery.candidates == ["70000000-0000-4000-8000-000000000007"]' "$results/reset-reconstructed.json" >/dev/null
expect_exit absent 0
expect_exit checkpoint 0
expect_exit changed-current 0
expect_exit competing 0
expect_exit reconciled 0
expect_exit interrupted 1
expect_exit quarantined 0
expect_exit reset-absent 0
expect_exit reset-reconstructed 0

printf 'fixture=%s\nresults=%s\n' "$fixture" "$results"
