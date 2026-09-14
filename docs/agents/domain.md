# Domain docs

This repo uses a single-context layout. The product's support for multiple projects does not require multiple domain contexts in this repository.

## Before exploring the codebase

Read root `CONTEXT.md` for domain vocabulary when it exists. Read the ADRs under `docs/adr/` that concern the area you are exploring.

If a root `CONTEXT-MAP.md` exists after a future layout change, use its pointers to read the relevant context files. Also check any corresponding `src/<context>/docs/adr/` directories.

If these files or directories are absent, proceed silently. Do not report their absence or propose creating placeholders. The `/domain-modeling` skill creates them when terms or decisions are resolved.

## Document ownership

- `CONTEXT.md` owns the domain glossary.
- `docs/adr/` owns architectural decisions and their rationale.
- `docs/vision.md` owns the product direction and bootstrap plan.
- Specs and tickets follow [Issue tracker](issue-tracker.md).

## Use the glossary's vocabulary

Use the terms defined in `CONTEXT.md` when naming concepts in issues, proposals, hypotheses, and tests. If a needed concept is absent, check whether an existing term fits. Record a genuine gap for `/domain-modeling`.

## Flag ADR conflicts

If a proposal contradicts an ADR, name the ADR and explain why the decision needs to be reconsidered. Preserve the recorded rationale when proposing a change.
