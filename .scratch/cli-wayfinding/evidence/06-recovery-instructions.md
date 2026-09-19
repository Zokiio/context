# Recovery instruction verification

Actor: Codex /root. Date: 2026-09-19.
Instruction revision: `ce8b183`, incorporating `adf7f83` and `625fe74`.
Reader product revision: `3f966925076e07c2cfe46a63981fb729764e8067`.

## Results by criterion

1. The task-context example uses direct `--bundle` access. The recovery skill supplies the explicit checkout for resume, preserves caller scope and limits, and explains justified limit overrides. AGENTS routes a bare continuation request to the skill. The local PROFILE provides format details without requiring arbitrary tickets to link the CLI implementation spec.
2. The skill retains the exact collection that governed the work. The trial captures an initial root, changes a source without committing, checks the digest difference, then captures and reads reassessed context. The root snapshot remains byte-identical while later successors use the reassessed bytes. Required note sections include failed approaches, questions, checks with identity/evidence, and next action.
3. Publication creates an exclusive observation directory, copies exact context bytes, and publishes note.md last through a unique same-directory temporary file and no-overwrite rename. A deliberate directory collision returns 17 and changes no cache file. Two successors preserve both accounts; explicit reconciliation names both.
4. The skill refreshes facts, reconstructs absent notes, preserves conflicts, separates reported checks from evidence, and states the next action without another approval for settled scope. Blocking questions use an open Decision and the affected WorkItem relationship. Settled decisions and accepted evidence belong outside the cache. Independent review identified and rechecked the A/B context timing correction.
5. The trial preserves an interrupted publication, then quarantines it after its fixture author finishes. An actual owned process remains live during a refusal check and is reaped before retirement. Both live and unknown writer refusals preserve the entire task-cache hash manifest. Recovery records retain ID, actor or explicit unavailable attribution, time, reason, and stopped-writer evidence. A whole-history reset archives the graph and reconstructs a new root with no predecessors.
6. The instructions offer cache-limit overrides before reset, describe continuity loss and retained archives, and preserve authoritative files. The reviewed ignore entry excludes the root .context-cache directory; non-Git use requires no ignore setup. There is no CLI writer command, automatic cleanup, claim, or model-understanding guarantee.
7. The root executed the retained script with the fixed reader. All asserted exits, JSON states, digest checks, and refusal snapshots passed. Absent, available, changed, conflict, reconciliation, quarantine, and reset/reconstruction reads returned 0. Interrupted, unknown-writer, and live-writer reads returned 1. The collision helper returned 17 before writing.

## Evidence

- [Executable filesystem trial](06-filesystem-trial.sh)
- [All retained result digests](06-trial-manifest.json)
- [Initial versus reassessed baseline](06-filesystem-trial/context-baseline.sha256)
- [Live writer state](06-filesystem-trial/live-writer-state.txt)
- [Original instruction review](06-instruction-review.md)
- [Resolved finding](06-instruction-recheck.md)

The trial uses controlled UUIDs in a fresh temporary namespace and exercises exclusive creation and collision refusal. These are cooperative procedural examples, not access-control enforcement. The real fresh-agent continuation is ticket 07 and is not claimed here. Earlier exploratory result folders remain disposable and are not the accepted trial.
