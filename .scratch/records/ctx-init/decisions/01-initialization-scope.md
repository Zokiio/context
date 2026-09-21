---
type: Decision
id: ctx-init-initialization-scope
title: Define initial ctx init capabilities and supported targets
decisionState: resolved
---

# Define initial ctx init capabilities and supported targets

The user authorized implementation after the Opsbase and Mukabi trials, then selected a noninteractive command for their repeated mechanical preparation.

## Resolution

Implement the [initialization ticket](../issues/01-agent-guided-project-initialization.md) in one PR. Run `ctx init` from the target project root with explicit title, tracker, skills, documentation, records, and source-root choices. Embed portable guidance, create only a Project manifest with empty commitments and decisions, call existing setup, append ignore rules, retain the running binary's version, and print instruction pointers and a complete file report.

Preserve differing files and existing bindings. Leave tracker discovery, real-task selection, instruction-file edits, snapshot retrieval, and conflict resolution with the caller. Require an explicit tracker description rather than assuming that absent remote issues mean no tracker. Offer local files only when no tracker exists.

The embedded templates own the portable skills and recovery profile. Generate the repository's development copies from them and test for drift. Target projects need only the supplied binary. Preserve the merged version-provenance and useful-information checkpoint rules.

Setup already writes configuration, and init creates a Project record. This is a bounded bootstrap write operation, not permission to modify existing records or invent a backlog. Readers remain read-only.

Defer human questionnaires, agent mode, harness detection, Copilot, multiple tools, hooks, symlink creation, link fallbacks, and general rerun reconciliation. Both measured recovery first edits were correct. Continuity worked, but the pair establishes neither a correctness advantage nor a speed benefit.

## Basis

[Opsbase](../../../ctx-init/opsbase-trial.md) used Codex, ordinary skill copies, GitHub Issues, and an identified local snapshot. [Mukabi](../../../ctx-init/mukabi-trial.md) used Claude Code, individual skill symlinks, and an existing local roadmap. Both required a supplied binary, adapted linked guidance, an explicit binding and source selection, preservation of tracker authority, and separation of disposable local files from durable records.

The measured first edits were correct both without and with a checkpoint, at 91.32 and 121.71 seconds respectively. Both sessions missed the requested pre-action plan. Keep the rule that a checkpoint must preserve useful information beyond the ticket and checkout.

The broader proposals remain in [discovery](../../../ctx-init/discovery.md). The [manual checklist](../../../../docs/tracker-snapshots.md) records the remaining task selection and snapshot steps. Neither the trials' pending human reviews nor Mukabi's unrelated deployment blocker changes this initialization scope.
