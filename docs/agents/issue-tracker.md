# Issue tracker: Local Markdown

Issues and specs for this repo live as Markdown files under `.scratch/`. Track these authoritative records in Git, despite the directory name. GitHub hosts the repository, but using this tracker requires no external account or service.

This is the bootstrap workflow for developing the product. The product's eventual record format is a separate design choice. See [Product vision](../vision.md).

## Conventions

- Use one directory per feature: `.scratch/<feature-slug>/`.
- Write the spec at `.scratch/<feature-slug>/spec.md`.
- Write one file per implementation ticket at `.scratch/<feature-slug>/issues/<NN>-<slug>.md`. Start numbering at `01` within each feature.
- Record the triage role in a `Status:` line near the top of each ticket. Use the strings in [Triage labels](triage-labels.md).
- Append comments and conversation history under a `## Comments` heading.
- Link each ticket to its spec and any relevant decisions, evidence, or blocking tickets. Use file paths to identify tickets in other features because ticket numbers are local to a feature.

Each record has one authoritative home. A ticket owns its status and acceptance criteria. An ADR owns a decision and its rationale. Handoffs, summaries, and workspace views link to these records.

Include spec and ticket changes in the repository's Git history through the normal development workflow. Keep disposable output separate from these records.

## When a skill says "publish to the issue tracker"

Create the spec or ticket at its path above. Create the containing directory if needed.

## When a skill says "fetch the relevant ticket"

Read the referenced file. If the user provides only a number, resolve it within the named feature. If several features match, ask which feature they mean.

## Wayfinding operations

The `/wayfinder` workflow uses a map file with one child file per ticket.

- **Map**: `.scratch/<effort>/map.md`, containing Notes, Decisions-so-far, and Fog.
- **Child ticket**: `.scratch/<effort>/issues/<NN>-<slug>.md`, numbered from `01`, with the question in the body. A `Type:` line records `research`, `prototype`, `grilling`, or `task`.
- **Status**: wayfinding tickets use `Status: claimed` and `Status: resolved` for their execution state, as defined by the local tracker template. These values are separate from the five triage roles.
- **Blocking**: a `Blocked by: NN, NN` line near the top lists blockers within the same effort. A ticket is unblocked when every listed ticket is `resolved`.
- **Frontier**: scan the effort's issue files for unresolved, unblocked, unclaimed tickets. Choose the first by number.
- **Claim**: set `Status: claimed` and save before starting work.
- **Resolve**: append the answer under `## Answer`, set `Status: resolved`, then append a short summary and link to the map's Decisions-so-far.
