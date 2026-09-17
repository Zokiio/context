# What belongs in the shortest useful intervention view?

Type: prototype
Labels: wayfinder:prototype
Status: resolved
Blocked by: 01
Parent: [Direct and resume agent work with the local CLI](../map.md)

## Question

What should the CLI show first so a human can identify the needed intervention and direct work? Prototype a compact presentation from existing orientation facts, then obtain human reactions to grouping repeated causes, affected commitments, in-progress work, links to detail, and incomplete or unknown information.

Decide how the brief relates to full text and JSON output and which behavior should remain in the existing orientation operation. Preserve authored commitment order and source attribution. Avoid automatic prioritization by blocker count or readiness claims unsupported by current checks.

Inputs: [research findings](../research.md), [existing intervention proposal](../../workflow-improvements/discovery.md#make-the-next-intervention-visible), [reader reference](../../../docs/readers.md).

## Prototype

- Archived primary source: branch `feat/prototype-cli-orientation`, commit `90f6e8e363cfd9f3c1fb4f4aa0693ed30aa52088`, file `internal/cli/intervention.prototype.html`. Recover with `git show 90f6e8e363cfd9f3c1fb4f4aa0693ed30aa52088:internal/cli/intervention.prototype.html`. The `variant` query parameter selects A, B, C, or D; D reflects the agreed broader orientation.
- A disposable preview copy remains at `.cache/cli-wayfinding/intervention.prototype.html`. The branch, not this cache copy, preserves the artifact. No prototype HTML remains in the production source directory on the current checkout.
- Variants show grouped problems, a commitment list, and a short briefing using the same fictional orientation facts. The scenario control switches between complete evaluation and incomplete inventory; the latter suppresses pickup eligibility.
- This explores presentation only. It does not implement CLI commands, inspect live project data, write records, or establish a handoff schema. The user reacted to the prototype and confirmed the presentation decisions below.

## Comments

2026-09-17: The user compared the prototype with supplied screenshots of Tenex operator and agent orientation. The user confirmed a broader project orientation first, with the needs-attention view inside it. The reference screenshots show workspace/identity, work and document indexes, and health/freshness information; they do not establish requirements to reproduce Tenex integrations or command names.

2026-09-17: Added variant D to demonstrate the confirmed direction: project scope and goal, current commitments in authored order, shared blockers, continuation pointers, new pickup, project references, and explicit freshness limits. Working-note paths and discovery are illustrative proposed behavior; detailed representation remains with the handoff decision. Incomplete-inventory warnings remain prominent. Exact defaults, detail expansion, and the relation of human and agent output remain open.

2026-09-17: The user confirmed readable text for humans and structured JSON for agents, backed by the same facts and checks. Skills supply the instructions for acting on that information. Do not introduce separate readiness or acceptance logic for different audiences. Default detail level and expansion behavior remain to be settled.

## Answer

Resolved with the user on 2026-09-17 through the prototype and follow-up questions.

### Project orientation comes first

Use a broader project orientation with a needs-attention section inside it, as illustrated by variant D. Start with project identity, scope, and goal. Show current commitments and ongoing work, then shared blockers and missing information, continuation and new-pickup pointers, and relevant project references.

Preserve authored commitment order. Group a shared blocking cause once and name the work it affects. Keep in-progress work separate from new pickup eligibility. Provide source pointers so users and agents can inspect the authoritative detail. The reference screenshots informed the structure; their integration health checks, repository inventories, and command names are not requirements to copy.

### One set of facts, different renderings

Default human-readable text is the compact project overview. Expanded detail is available on request and includes all checks, sources, and diagnostics. JSON retains the full structured result, so the compact text does not discard facts available to agents.

All renderings use the same facts and checks. Keep readiness and acceptance evaluation in the existing application operations, rather than introducing separate rules for humans and agents. Skills supply instructions for acting on the report. Presentation does not authorize pickup or run an agent.

### Unknown facts stay visible

Missing information and warnings remain visible in compact output. An incomplete inventory cannot produce a pickup shortlist. Preserve independently known facts without presenting the whole project as healthy.

Distinguish current source observations from temporary working notes and externally retrieved snapshots. A note is not current verification, and reading it does not establish that a model understood it. Detailed working-note discovery and freshness behavior remain with [What does a handoff retain, and how does a session select it?](03-handoff-contract.md) and [What should resumption show when requirements or readiness changed?](04-resume-behavior.md).

### Implementation handoff

This resolves the presentation direction, not a production implementation. Retain the existing full JSON facts and evaluation semantics when specifying the CLI changes. Choose exact detail-option spelling and any necessary output-version treatment in the implementation specification. Do not copy the prototype's hardcoded paths, synthetic records, or browser controls into the product.
