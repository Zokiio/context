# Issue tracker: Local Markdown

Issues and specs for this repo live as Markdown files under `.scratch/`. Track these authoritative records in Git, despite the directory name. GitHub hosts the repository, but using this tracker requires no external account or service.

Skills author these records while the first CLI remains read-only. New implementation tickets use the minimal OKF profile below. Specs and wayfinding records remain plain Markdown outside the bundle. See [Product vision](../vision.md).

## Conventions

- Write the spec at `.scratch/<feature-slug>/spec.md`.
- Use `.scratch/records/` as this project's single OKF bundle. Its root `project.md` carries `type: Project`, a stable `id`, and `title` in YAML frontmatter, as required by [ADR 0001](../adr/0001-one-okf-bundle-per-project.md).
- Write one file per implementation ticket at `.scratch/records/<feature-slug>/issues/<NN>-<slug>.md`. Start numbering at `01` within each feature.
- Record each implementation ticket's triage role in its frontmatter `triage` field. Use the strings in [Triage labels](triage-labels.md).
- Append comments and conversation history under a `## Comments` heading.
- Put authoritative file links under level-two `Spec`, `Blocked by`, and `Context` headings. Use file paths to identify tickets in other features because ticket numbers are local to a feature. The [reader spec](../../.scratch/context-reader/spec.md) defines section parsing and selection.

Each record has one authoritative home. A ticket owns its status and acceptance criteria. An ADR owns a decision and its rationale. Handoffs, summaries, and workspace views link to these records.

Include spec and ticket changes in the repository's Git history through the normal development workflow. Keep disposable output separate from these records.

## Minimal implementation-ticket profile

Require YAML frontmatter with `type: WorkItem`, a nonempty stable `id`, `title`, and `triage`. Keep the ID when moving or renaming a ticket. Each ticket has its own ID, independent of its filename or feature-local number.

Use `triage` as the single home for the triage role. Reserve OKF's optional `status` field for document lifecycle: `draft`, `stable`, or `deprecated`. Omission means `stable`. Record execution separately as `execution: unstarted`, `in-progress`, `completed`, or `cancelled`. Missing or invalid execution is unknown, never implicitly unstarted. Orientation computes readiness separately.

Keep requirements and acceptance criteria in the ticket body. Require at least one nonempty criterion under `## Acceptance criteria`. Its Spec section is optional. Full OKF validation is outside the readers; the authoring profile still applies to new tickets.

Every work item declares `## Blocked by` and `## Blocked by decisions`. Use local Markdown links for relationships. An empty section or the literal word `None` declares none. A missing section means unknown. Do not use explanatory prose as a substitute for an explicit declaration.

After acceptance, use `## Acceptance` for one link to the current Acceptance record. Keep old links under Comments or other history. A completed execution state alone does not satisfy a dependency. Follow [Record acceptance](acceptance.md) when finishing, reassessing, or migrating work.

Keep plain Markdown specs and context documents outside `.scratch/records/`. Link them using paths relative to the ticket, and supply their directories through `--allow-source`. For example, a ticket in `.scratch/records/context-reader/issues/` links to the existing spec at `../../../context-reader/spec.md`. A leading `/` in a document link means the bundle root.

## Project and decision profiles

The project manifest has required level-two `Goals`, `Current commitments`, and `Open decisions` sections. Goals contains authored direction and links to designated documents. Current commitments links directly to chosen WorkItems, including unstarted work. A specification link does not commit every associated ticket.

Open decisions links to Decision records. Each Decision has `type: Decision`, a stable `id`, a `title`, and `decisionState: open` or `resolved`. A resolved decision requires a nonempty `## Resolution`. The record owns its state even when an index still lists it as open.

Only a ticket's explicit Blocked by decisions links make a decision block that ticket. If its answer imposes implementation requirements, also select the decision through Spec or Context so the existing task-context reader supplies it.

An empty link-only section or literal `None` declares no links. A missing section leaves information unknown. Goals may contain prose. Repeated recognized sections combine in document order, with nested headings inside the section. IDs are unique across typed records in a project.

Keep Decision and Acceptance records within the selected bundle, for example under feature-local `decisions/` and `acceptances/` directories. Keep immutable evidence documents outside the bundle and authorize their containing source roots. This avoids treating every historical evidence document as an inventory candidate.

## Completion during orientation adoption

The [orientation tickets](../../.scratch/session-orientation/discovery.md#implementation-handoff) introduce the checks incrementally. Record these tickets' execution states as work progresses. Migrate legacy reader records in their designated migration ticket rather than inferring completion from triage.

Before required orientation checks exist, use the approved blocker graph, recorded verification, and the existing task-context workflow for these bootstrap tickets. An unsupported check stays unknown in command output. Once the checks exist, inspect their result before pickup and acceptance.

Retain criteria results, source identity, and evidence after every slice. Begin structured acceptance authoring when current fingerprint output is available, then validate those records when acceptance checking lands. [Record acceptance](acceptance.md) owns the exact procedure and fingerprint reference.

## When a skill says "publish to the issue tracker"

Create the spec or ticket at its path above. Create the containing directory if needed.

## When a skill says "fetch the relevant ticket"

Read the referenced file. If the user provides only a number, resolve it within the named feature. If several features match, ask which feature they mean.

## Wayfinding operations

The `/wayfinder` workflow uses a map file with one child file per ticket outside the OKF bundle. These discovery records retain their bootstrap syntax. Accepted implementation work has one authoritative ticket in the bundle; wayfinding records link to it rather than duplicating its acceptance criteria or triage role.

- **Map**: `.scratch/<effort>/map.md`, containing Notes, Decisions-so-far, and Fog.
- **Child ticket**: `.scratch/<effort>/issues/<NN>-<slug>.md`, numbered from `01`, with the question in the body. A `Type:` line records `research`, `prototype`, `grilling`, or `task`.
- **Status**: wayfinding tickets use `Status: claimed` and `Status: resolved` for their execution state, as defined by the local tracker template. These values are separate from the five triage roles.
- **Blocking**: a `Blocked by: NN, NN` line near the top lists blockers within the same effort. A ticket is unblocked when every listed ticket is `resolved`.
- **Frontier**: scan the effort's issue files for unresolved, unblocked, unclaimed tickets. Choose the first by number.
- **Claim**: set `Status: claimed` and save before starting work.
- **Resolve**: append the answer under `## Answer`, set `Status: resolved`, then append a short summary and link to the map's Decisions-so-far.
