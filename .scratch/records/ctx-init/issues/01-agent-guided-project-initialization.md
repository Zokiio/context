---
type: WorkItem
id: ctx-init-agent-guided-project-initialization
title: Prepare a project with ctx init
triage: ready-for-agent
status: stable
execution: completed
---

# Prepare a project with ctx init

The Opsbase and Mukabi trials repeated the same project preparation with different agent tools and trackers. Provide a noninteractive `ctx init`, run from the target project root with explicit flags, to perform the mechanical preparation in one PR.

## Current disposition

The user authorized implementation on 2026-09-20 and selected the flag-driven scope below. The linked scope decision is resolved. Human review of the trials' product PRs is separate from this authorization.

## Implementation scope

`ctx init` requires the project title, authoritative tracker description, skills directory, and agent documentation directory. It accepts a records directory and explicit source roots. Paths resolve from the target project root. Required flags replace discovery and questionnaires: the caller establishes the agent tool, its existing instructions and skill layout, and the tracker before invoking init. Ask which tracker is used when it cannot be established. Offer local file tracking only when none exists.

1. Write embedded task-context and recovery-notes skills, the recovery profile, and trimmed acceptance and tracker guidance. Render the supplied executable's absolute path and selected project paths. The target needs no ctx source checkout.
2. Create a Project manifest in the selected records directory with a generated stable UUID, empty current commitments, and no open decisions. Create no WorkItems. Preserve any existing manifest.
3. Bind the records and explicit source roots through the existing setup implementation, without replacing a conflicting binding.
4. Append missing ignore rules for disposable local files, recovery cache, evidence, snapshots, generated acceptance records, and configuration locks. Keep project guidance and authored records durable.
5. Retain the running binary's version output in the local trial directory. Preserve uncertainty for modified or unstamped builds.
6. Print pointers for the caller to add to the existing instructions file. Report created, unchanged, and skipped files, binding conflicts, and the remaining manual work. Existing files with differing content remain unchanged. The command never edits AGENTS.md or CLAUDE.md.

Manual work includes selecting a real task and its source documents, checking tracker freshness, adding an identified local snapshot only when needed, and reviewing instruction or file conflicts. Preparation does not establish task readiness, acceptance, review, hosted CI, or tracker updates.

## Write boundary and template ownership

Setup already writes configuration. Init adds bootstrap file creation, including one new Project manifest, and append-only ignore rules. It does not modify existing records or create a backlog. The context, orientation, and resume readers remain read-only. This command does not authorize general record mutation.

Embedded portable templates are the source of truth for the installed skills and recovery profile. Generate this repository's copies from those same templates with its development paths, and check that they remain current. Keep repository-specific tracker conventions separate from the trimmed guidance installed in target projects.

## Acceptance criteria

- A supplied binary initializes a target project without prompts, network access, a ctx checkout, or changes to its existing instructions file.
- The two skills, recovery profile, and trimmed acceptance and tracker guidance have working relative references and target-specific executable and record paths. The repository's generated copies match their embedded templates, including version provenance and the useful-information checkpoint rule.
- The new Project has a stable ID, the supplied title, empty commitments, and no decisions. Init creates no WorkItems and preserves existing local or remote tracker authority.
- The existing setup path binds the selected records and explicit source roots. Orientation reads the empty project successfully. A manually selected task can subsequently use context and resume.
- Existing differing files and conflicting bindings remain unchanged and are named in the report. An identical rerun preserves file contents and manifest identity. Report partial preparation visibly without claiming success.
- Disposable local files, evidence, snapshots, acceptance records, recovery cache, and configuration locks are ignored in Git, while skills, guidance, and authored records remain trackable. Retain the running binary's exact version output.
- The report supplies instruction pointers and names the manual steps for task selection, tracker freshness, source selection, and conflict resolution. A snapshot retains source identity, retrieval time, raw source, and visible adaptations under the installed guidance.

## Deferred scope

No questionnaire, harness detection, agent mode, hooks, automatic instruction edits, overwrites, tracker connector, work-item generation, or general rerun reconciliation. Claude symlink creation is deferred; callers can use their existing layout or select a directory explicitly. The paired recovery measurement showed continuity but no first-action correctness or speed benefit. Init adds no checkpoint requirement.

## Blocked by

None

## Blocked by decisions

- [Define initial capabilities and supported targets](../decisions/01-initialization-scope.md)

## Acceptance

- [Local review follow-up acceptance](../acceptances/02-review-followup.md)

## Context

- [Define initial capabilities and supported targets](../decisions/01-initialization-scope.md)
- [Opsbase trial findings](../../../ctx-init/opsbase-trial.md)
- [Mukabi trial findings](../../../ctx-init/mukabi-trial.md)
- [Manual preparation checklist](../../../../docs/tracker-snapshots.md)
- [Existing project setup](../../../../docs/connect-projects.md)
- [Product direction](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-09-21: The user approved merge after review and requested the remaining P3 fix and a package-scoped `-update` note. Setup now exposes its coordination paths for init's file report. Keep this review follow-up in its own commit. Future scope revisions should precede implementation in a separate commit or PR. Preserve the [original implementation acceptance](../acceptances/01-initialization-implementation.md) and its tested revision.

2026-09-20: Implemented the flag-driven command and embedded templates. Full Go tests, race tests, vet, build, template-generation checks, and a standalone supplied-binary trial passed. The trial preserved an existing local roadmap and instructions, used an explicit Claude skills directory, exercised context and resume after manual task selection, and retained conflicts. Verification output and the workflow acceptance remain local. Human PR review remains outstanding.

2026-09-20: The user authorized ctx init and then selected a noninteractive, flag-driven command for the repeated mechanical preparation. The scope decision is resolved. Template generation prevents the repository and distributed skills from drifting. Earlier draft dispositions below are historical.

2026-09-20: Initially drafted from two three-agent debates, with three harnesses, two entry modes, shared skills, and copy fallback. The user redirected the sequence to manual bootstrap, real use, scope rewrite, then implementation.

2026-09-20: Rewritten from the Opsbase issue 165 trial. The original debate remains in the discovery notes. The trial exercised one harness, existing-file adaptation, GitHub snapshot boundaries, local reads, and checkpoint/resume. No init implementation has started. No further design debate is needed to establish these observations.

After review of the first external trial, use the [manual checklist](../../../../docs/tracker-snapshots.md#bootstrap-checklist-before-another-trial) before building this command. A second external project should establish which adaptations repeat. The ticket remains draft, needs-info, and unstarted.

2026-09-20: Select Mukabi with Claude Code for the next external trial and compare fresh-session continuation with and without its checkpoint. The snapshot discriminator is explicit in the profile. Initialization remains draft and unstarted.

2026-09-20: The [Mukabi trial](../../../ctx-init/mukabi-trial.md) exercised Claude symlink discovery and an existing local roadmap. Supplied-binary and linked-guidance adaptation repeated. The paired recovery run showed no action-time benefit and missed the plan-before-action protocol in both conditions. Keep this ticket draft, needs-info, and unstarted.
