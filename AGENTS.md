# Repository guidance

Before planning product changes, read `docs/vision.md` for the product direction and bootstrap milestone.

## Change scope

Keep features and pull requests small. Aim for one independently testable outcome per PR, with its necessary tests and documentation. Split independent refactors and follow-up behavior into separate PRs. If scope grows during implementation, finish the current outcome and record the rest as follow-up work.

## Agent skills

### Issue tracker

Specs and tickets live in local Markdown files under `.scratch/` and belong in Git. Before reading or changing them, read `docs/agents/issue-tracker.md`.

Keep verification output and generated acceptance records out of Git. Commit reusable tests and fixtures, and summarize checks in the PR description. The issue-tracker guidance defines local storage and fresh-checkout behavior.

### Triage labels

This repo uses the five default triage roles. Before assigning a triage role, read `docs/agents/triage-labels.md`.

### Domain docs

This repo uses a single-context layout with root `CONTEXT.md` and `docs/adr/`. Before exploring the codebase, read `docs/agents/domain.md`.

### Recovery notes

Before continuing a task, publishing a checkpoint, or recovering interrupted working notes, read `.agents/skills/recovery-notes/SKILL.md`.
