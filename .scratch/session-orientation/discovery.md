# Session orientation discovery

This note records the design interview for the next milestone named in the [product vision](../../docs/vision.md#after-the-first-milestone). The [specification](spec.md) defines the implementation contract. Eight approved implementation tickets are linked in the implementation handoff below.

## Settled decisions

- Present eligible work with reasons. The human or agent chooses the next work item. Orientation does not select a single next action.
- Return the known information when project records are incomplete. Identify the gaps and make unknown readiness explicit.
- Scope the first release to one explicitly selected local project.
- Include chosen work that has not started among current commitments. Show execution state separately. [CONTEXT.md](../../CONTEXT.md) owns the definition of current commitment.
- Record execution state explicitly as unstarted, in progress, completed, or cancelled. Missing state means unknown. Keep completion evidence separate from execution state.
- Require unstarted execution state, membership in current commitments, the `ready-for-agent` triage role, and passing readiness checks for the default shortlist. Show work in progress separately. Show ready work outside current commitments separately as backlog.
- Require a completed prerequisite to have a recorded acceptance decision identifying the criteria, supporting evidence, and tested revision before it satisfies a dependency. The CLI checks those records and relies on the acceptance decision. [CONTEXT.md](../../CONTEXT.md) owns the definition of acceptance decision.
- Treat a cancelled prerequisite as an unsatisfied dependency.
- Use explicit sections in the bundle's `project.md` to link project goals, current commitments, and open decisions to their authoritative documents and tickets.
- Keep commitments explicit per implementation ticket. Show associated specifications for navigation. A broader epic hierarchy is a later capability.
- Allow authorized workflow skills to record acceptance. Identify the decision's author, criteria, evidence, and tested revision. Attribute any human approval separately.
- Apply an unresolved blocking decision only to the tickets explicitly linked to it. Evaluate other work against its own readiness conditions.
- Require a new acceptance decision when the ticket's requirements, linked requirement documents, or supporting evidence change. Preserve the previous decision and its tested revision. Unrelated comment edits leave acceptance intact.
- Report readiness as ready, blocked, or unknown, with reasons. Keep blocked and unknown work off the default shortlist.
- Treat an explicitly empty dependency or blocking-decision section as declaring none. A missing required section makes the affected evaluation unknown.
- Discover `WorkItem` records by scanning the explicitly selected project bundle. Use `project.md` commitment links to select current commitments from that inventory.
- Expose the operation as `ctx orient`, with a concise text report by default and versioned structured output through `--json`. Both formats report the same facts.
- Keep orientation compact: project goals, ticket titles and states, readiness reasons, open decisions, gaps, and source references. Use `ctx context` for a selected ticket's full requirements and documents.
- Detect stale acceptance through content fingerprints. The confirmed boundary covers the ticket body with designated Comments and Acceptance exclusions, plus whole linked requirement and evidence files. The user confirmed the expansion from criteria-only coverage during the ticket-breakdown review.
- Preserve the tested revision as historical evidence. It need not equal the current Git HEAD.
- Return exit status `0` when evaluation finishes, even if every ticket is blocked; `1` for partial results caused by missing or unreadable information or limits; and `2` for invalid invocation or operation failure. Return the useful report whenever possible.
- Prove the milestone through a real ticket pickup: a fresh agent uses orientation to identify a committed ticket, explains its eligibility, then obtains full task context through `ctx context`. The five completed reader tickets stay off the shortlist. Tests cover stale acceptance, missing records, cancelled prerequisites, and dependency cycles.

The user accepted these decisions during the design interview on 2026-09-16.

## Existing constraints

- [CONTEXT.md](../../CONTEXT.md) distinguishes session orientation, task context, context completeness, and work readiness.
- The [tracker profile](../../docs/agents/issue-tracker.md#minimal-implementation-ticket-profile) reserves `status` for document lifecycle and `triage` for the triage role. Execution-state fields and computed readiness were deferred for the first reader.
- [ADR 0001](../../docs/adr/0001-one-okf-bundle-per-project.md) makes one OKF bundle the project boundary, independent of Git repository boundaries.
- [ADR 0002](../../docs/adr/0002-work-relationships-in-markdown-sections.md) places authoritative relationships in named Markdown sections.
- [ADR 0003](../../docs/adr/0003-recorded-acceptance-for-readiness.md) records why orientation relies on authored acceptance decisions and retained evidence.

## Confirmed fingerprint correction

The current tickets contain inline requirements outside `Acceptance criteria`. For example, [ticket 03's Scope section](../records/context-reader/issues/03-follow-blocker-tickets.md#scope) excludes automatic parent traversal and requires reuse of the existing reader. Changing that section alone would leave the approved initial fingerprints unchanged.

The proposed correction was to fingerprint the entire ticket body except explicitly designated comments and acceptance bookkeeping, while continuing to fingerprint whole linked requirement and evidence files. The user initially invoked `/to-spec` before separately answering that correction, so the specification recorded assumption A1. During the later `/grill-me` review, the user answered "Confirm whole-body coverage." A1 is now a confirmed decision.

## Specification handoff

The [specification](spec.md) defines the record profiles, fingerprint encoding, readiness and eligibility rules, partial results, output contract, limits, and migration trial. Technical synthesis choices are identified there. The user confirmed Go application-operation tests with real project files and a small CLI suite during specification drafting.

## Implementation handoff

The user approved eight tickets after reviewing the dependency order and two pairs of proposed slices. Keep the overview separate from shortlisting, and keep direct acceptance validation separate from recursive dependency evaluation.

- [01: Show project orientation and work inventory](../records/session-orientation/issues/01-show-project-orientation.md)
- [02: List eligible work with reasons](../records/session-orientation/issues/02-list-eligible-work.md)
- [03: Explain blocking decisions](../records/session-orientation/issues/03-explain-blocking-decisions.md)
- [04: Expose reproducible requirement fingerprints](../records/session-orientation/issues/04-expose-requirement-fingerprints.md)
- [05: Satisfy direct prerequisites using recorded acceptance](../records/session-orientation/issues/05-verify-prerequisite-acceptance.md)
- [06: Evaluate prerequisite chains and cycles](../records/session-orientation/issues/06-evaluate-prerequisite-chains.md)
- [07: Migrate project records and prepare the pickup trial](../records/session-orientation/issues/07-migrate-records-and-prepare-trial.md)
- [08: Verify fresh-session pickup on the real project](../records/session-orientation/issues/08-verify-fresh-session-pickup.md)

Each ticket owns its acceptance criteria and Blocked by links. Ticket 01 can start immediately. Tickets 02 and 04 can proceed independently after it.

The user confirmed that intermediate releases may expose `ctx orient` while unsupported checks remain explicit unknowns, cause partial reports, and exclude affected work from the shortlist. Direct prerequisite acceptance can pass only for completed prerequisites with no further work dependencies until recursive evaluation is implemented.

Authoring conventions and completion guidance belong in ticket 01. Retain evidence for each slice, begin structured acceptance authoring when fingerprints are available, and validate it when acceptance checking lands. This avoids deferring all completion records until migration.

The sequencing review found that editing linked conventions can stale existing acceptance because linked documents use whole-file hashes. Any later requirement changes still require reassessment. Ticket 07 owns remaining migration and final preflight checks; ticket 08 begins with orientation and selection while genuinely unstarted.

Tickets use the current minimal authoring profile at publication, with ready-for-agent triage and unchecked criteria. Ticket 01 adopts execution metadata under the new conventions. No orientation implementation or acceptance is claimed by this planning handoff.

## Grounding

PR [#1](https://github.com/Zokiio/context/pull/1) merged the task-context reader at `ecb8f294144dab8bb36f2e74e6ad174bcce17ef1`. At that revision, all five reader tickets still have `triage: ready-for-agent` alongside completed acceptance checklists and completion notes. Triage alone cannot distinguish outstanding work from finished work.

The existing reader selects an explicit ticket, recursive blockers, and linked Spec and Context sources. It does not interpret completion evidence or compute work readiness. See the [reader specification](../context-reader/spec.md) and [acceptance evidence](../context-reader/acceptance-trial.md).

## User-provided structure references

On 2026-09-16, the user pointed to images in `removethis/` as possible structural references. The observations below are design input.

- The [agent orientation screenshot](<../../removethis/Screenshot 2026-09-14 at 23.28.35.png>) separates project identity from a product and engineering map. It lists epics and specifications with execution labels and open-ticket counts. The visible excerpt does not establish parent relationships between those groups or show individual ticket readiness.
- The [repository and conventions screenshot](<../../removethis/Screenshot 2026-09-14 at 23.26.37.png>) separates documentation, epics, specifications, logs, external material, and scratch work. Its conventions index explains the scope and responsibility of linked documents.
- The [operator screenshot](<../../removethis/Screenshot 2026-09-14 at 23.27.26.png>) groups workspace identity, diagnostic counts, and suggested commands. Its integrations, caches, and runtime checks extend beyond the settled local-project milestone.

The useful design questions are how much work hierarchy to show and how to separate project direction, delivery progress, readiness, and diagnostic information. The existing OKF bundle and linked-document conventions remain the constraints for this milestone.
