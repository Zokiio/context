---
type: WorkItem
id: ctx-init-agent-guided-project-initialization
title: Prepare a project through agent-guided ctx init
triage: needs-info
status: draft
execution: unstarted
---

# Prepare a project through agent-guided ctx init

The manual Opsbase trial used ctx on a real GitHub issue without an init command. It required a supplied binary, adapted skills and supporting references, a project binding, and a labeled local issue snapshot. Those observations replace the earlier multi-harness proposal as the basis for this draft.

## Current disposition

Keep this ticket draft, needs-info, and unstarted. Review the trial findings and the linked scope decision before implementation. The trial's local development and verification are complete; the Opsbase code remains uncommitted, with review and hosted CI outstanding.

## Scope proposed from the trial

Prepare an existing project for one coding agent using the workflow exercised with Codex on macOS. Reproduce the useful manual setup before adding broader installation behavior.

- Locate a supplied ctx binary. Installed guidance must not tell the target project to build ctx from its own repository.
- Discover and reuse the project's instructions, context layout, relevant documents, and tracker. Opsbase already had AGENTS.md, a context map, domain documents, and GitHub tracker conventions.
- Prepare the task-context and recovery-notes skills, the required recovery profile, and an adapted acceptance procedure. Include every supporting reference that the installed workflow actually uses.
- Remove assumptions about ctx's source checkout, local backlog layout, bootstrap history, and locally authored decisions.
- Preserve the authoritative tracker. For GitHub-backed work, clearly distinguish the local reader snapshot, source freshness, and observed local execution from live tracker state. The trial used manual retrieval, not a connector or synchronization service.
- Bind the selected record store through existing `ctx setup`, with explicit source roots. Keep generated snapshots, supplied binaries, evidence, and recovery observations out of Git.
- Add scoped pointers to existing instructions and verify task context, orientation, and checkpoint/resume using a real task.

The trial establishes the workflow content and its portability problems. It does not validate a particular human questionnaire, command-flag grammar, or unattended installer. Choose the smallest command entry that delivers this workflow when reviewing the scope decision.

## Acceptance criteria

- A target project can follow the supplied workflow without containing ctx's source code or adopting this repository's backlog layout.
- Required skills and references resolve, and existing project guidance and tracker authority remain intact.
- A GitHub-backed task retains its source identity and retrieval time. Local format adaptations and local execution observations are visible; the original source remains available for comparison.
- Task context includes the selected task and relevant project documents, including documents reached through a multi-context layout.
- Existing setup and reader commands can inspect the prepared project, and a published recovery checkpoint can be read back without inventing completion or acceptance.
- Generated local inputs and verification artifacts stay out of Git. The resulting report distinguishes local checks from tracker updates, review, and hosted CI.

These draft criteria are grounded in the Opsbase trial. They remain subject to review before implementation.

## Deferred scope

The trial did not exercise a standalone human questionnaire, automatic harness detection, Claude Code or Copilot installation, harness combinations, shared-directory links, copy fallback, hooks, or general rerun reconciliation. Preserve those ideas in discovery notes rather than making them first-release requirements. Do not build a tracker connector or synchronization framework as part of this draft by implication.

## Blocked by

None

## Blocked by decisions

- [Define initial capabilities and supported targets](../decisions/01-initialization-scope.md)

## Context

- [Define initial capabilities and supported targets](../decisions/01-initialization-scope.md)
- [Opsbase trial findings](../../../ctx-init/opsbase-trial.md)
- [Existing project setup](../../../../docs/connect-projects.md)
- [Product direction](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-09-20: Initially drafted from two three-agent debates, with three harnesses, two entry modes, shared skills, and copy fallback. The user redirected the sequence to manual bootstrap, real use, scope rewrite, then implementation.

2026-09-20: Rewritten from the Opsbase issue 165 trial. The original debate remains in the discovery notes. The trial exercised one harness, existing-file adaptation, GitHub snapshot boundaries, local reads, and checkpoint/resume. No init implementation has started. No further design debate is needed to establish these observations.

After review of the first external trial, use the [manual checklist](../../../../docs/tracker-snapshots.md#bootstrap-checklist-before-another-trial) before building this command. A second external project should establish which adaptations repeat. The ticket remains draft, needs-info, and unstarted.

2026-09-20: Select Mukabi with Claude Code for the next external trial and compare fresh-session continuation with and without its checkpoint. The snapshot discriminator is explicit in the profile. Initialization remains draft and unstarted.
