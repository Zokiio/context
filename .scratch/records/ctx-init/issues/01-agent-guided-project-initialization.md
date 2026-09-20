---
type: WorkItem
id: ctx-init-agent-guided-project-initialization
title: Prepare a project through agent-guided ctx init
triage: needs-info
status: draft
execution: unstarted
---

# Prepare a project through agent-guided ctx init

The manual Opsbase and Mukabi trials used ctx without an init command, with Codex and GitHub Issues in one project and Claude Code and local plans in the other. The six repeated preparation steps below replace the earlier installer proposal as the scope of this draft.

## Current disposition

Keep this ticket draft, needs-info, and unstarted. Review the trial findings and the linked scope decision before implementation. Both trials have local implementation and verification results. Their product PRs remain subject to human review.

## Scope supported by the two trials

Keep initialization limited to the six preparation steps repeated in Opsbase and Mukabi:

- Locate a supplied ctx binary and retain `ctx version` output.
- Adapt task-context, recovery-notes, their required supporting references, and acceptance guidance to the target project.
- Bind the selected record store through existing `ctx setup`.
- Select the task's source documents and authorize explicit source roots.
- Preserve the existing tracker and its authority, whether remote issues or local plans. Use identified local snapshots only when needed.
- Separate disposable binary files, snapshots, evidence, and recovery cache from durable project guidance and records.

Mukabi verified individual Claude skill symlinks on macOS. Opsbase used ordinary skill copies with Codex. These observations guide manual placement; they do not establish a general installer, cross-platform link fallback, or rerun reconciliation. Keep the checklist until a command is explicitly chosen. This ticket remains draft, needs-info, and unstarted.

## Acceptance criteria

- A target project can follow the supplied workflow without containing ctx's source code or adopting this repository's backlog layout.
- Required skills and references resolve, and existing project guidance and tracker authority remain intact.
- A GitHub-backed task retains its source identity and retrieval time. Local format adaptations and local execution observations are visible; the original source remains available for comparison.
- Task context includes the selected task and relevant project documents, including documents reached through a multi-context layout.
- Existing setup and reader commands can inspect the prepared project, and a published recovery checkpoint can be read back without inventing completion or acceptance.
- Generated local inputs and verification artifacts stay out of Git. The resulting report distinguishes local checks from tracker updates, review, and hosted CI.

These draft criteria are grounded in the two trials. They remain subject to review before implementation.

## Deferred scope

The trials did not exercise a standalone human questionnaire, automatic harness detection, Copilot installation, harness combinations, cross-platform links, copy fallback, hooks added for ctx, or general rerun reconciliation. Preserve those ideas in discovery notes rather than making them first-release requirements. Do not build a tracker connector or synchronization framework as part of this draft by implication.

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

2026-09-20: The [Mukabi trial](../../../ctx-init/mukabi-trial.md) exercised Claude symlink discovery and an existing local roadmap. Supplied-binary and linked-guidance adaptation repeated. The paired recovery run showed no action-time benefit and missed the plan-before-action protocol in both conditions. Keep this ticket draft, needs-info, and unstarted.
