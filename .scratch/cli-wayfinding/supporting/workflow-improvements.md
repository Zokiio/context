# Help people direct and resume project work

Retained planning input, copied on 2026-09-17 from the pre-existing local `.scratch/workflow-improvements/discovery.md`. This snapshot preserves the research source; the resolved wayfinder decisions and specification own the selected milestone.

Status: discovery. These ideas are proposals for discussion, not accepted implementation scope.

## Product problem

The project already retains requirements, evaluates work readiness, and checks recorded acceptance. The next opportunity is to help people use those facts without reconstructing the project themselves.

A developer needs to know what can proceed and what remains uncertain. An agent needs the sources and boundaries for its next action. A project owner needs to understand which commitments need intervention. An architect needs to see when implementation would change an agreed contract or decision.

These needs should share the same records and checks. Separate presentations can emphasize different questions without creating separate versions of project truth.

The [vision](../../../docs/vision.md) and [domain glossary](../../../CONTEXT.md) define the existing direction and vocabulary. The proposals below build on context assembly, session orientation, and recorded acceptance already present in the implementation.

## A working example

Consider the vision's Wi-Fi work. The user owns a backend contribution to a broader device feature. For this example, suppose the team has not agreed whether a successful response means that configuration was queued or actually applied to the device.

That question affects the service contract and verification. Backend tests might establish successful queueing while leaving device behavior untested. An interruption might occur after those tests, followed by a change to the agreed contract.

The product should help the next person answer:

- What can we safely continue within the agreed scope?
- Which decision needs an answer, and which commitments depend on it?
- What do the observed results establish?
- What changed since the previous investigation?
- What information would let someone else continue the work?

The example is illustrative. It does not prescribe a device protocol or architecture.

## Make the next intervention visible

Extend session orientation with a concise view of work that needs attention. Group repeated causes and show the affected current commitments. Each entry should explain the condition, identify its source, and suggest a useful preparation action.

For the example, the view might identify the unresolved success-response decision, link the affected backend work, and explain that contract-dependent implementation is waiting. It could separately show missing device verification without implying that all independent work must stop.

Show a responsible person or role only when that information has been authored. Preserve the project's commitment order. A cause affecting several tickets can be useful context without becoming an automatic priority score.

Use the same underlying result for a brief human presentation and structured agent output. Keep work already in progress separate from the shortlist for new implementation. Include unknown conditions and incomplete inventory prominently, even when the rest of the report is short.

The first increment needs no new record type: derive grouped findings and preparation suggestions from existing orientation checks. Evaluate whether a developer and project owner can identify the needed intervention without reading the full inventory.

## Help clarify work without silently expanding it

Grooming should produce focused proposals for actual gaps. A proposal may contain a missing behavior, an unresolved question, a suggested context link, or a task split. It does not need every section for every task.

For a proposed split, identify each child's contribution, its own acceptance criteria, relevant source constraints, blockers, and verification approach. Keep a distinction between a parent's intended outcome and what a child can establish independently. Numerical limits, permitted states, and exception behavior should remain traceable to their source.

In the example, separating backend delivery from device verification must leave the end-to-end outcome visible. Completing the backend contribution must not silently complete the device feature.

A developer can review feasibility, an architect can inspect the contract boundary, and an owner can accept or reject the proposed scope. The agent can continue independent investigation while recording what still depends on an answer.

Begin with a workflow skill that produces one small reviewable document. Adopt useful conclusions individually. Formal proposal validation or managed ticket creation can follow if repeated use exposes enough authoring friction.

## Explain what the evidence establishes

Add an advisory verification view for a work item. For each criterion, show the observation, evidence source, tested revision, relevant environment, and unresolved gap. Distinguish explicit evidence relationships from agent-inferred matches.

A requirements review asks whether a criterion is clear and testable. Implementation verification asks whether the observed behavior satisfies it. The presentation should keep those judgments separate.

For the example, successful backend tests could support the queueing criterion. The view should still show that no observation establishes application on the device. An architect can inspect the contract interpretation, while an owner can see why the overall outcome remains unaccepted.

Start by identifying a criterion through its source path, text, and source hash. Introduce stable criterion identities only if edits make that approach inadequate. Avoid a percentage that treats unlike criteria as interchangeable units of progress.

This view supplies information to the acceptance decision's author. It does not change [the recorded acceptance model](../../../docs/adr/0003-recorded-acceptance-for-readiness.md). Partial coverage cannot satisfy a prerequisite whose acceptance requires the whole work item.

## Make interrupted work cheap to resume

A handoff should retain the observed requirement sources, relevant code revisions, completed steps, attempted checks and results, unfinished work, and unresolved questions. Preserve failed experiments when they would prevent repeated investigation.

On resumption, refresh current context and compare it with the handoff's observations. Explain known source changes and the work they may affect before suggesting continuation. A handoff describes what someone observed; it does not authorize work or override current readiness.

Existing hashes can establish that content differs. They cannot reconstruct previous text. A later comparison view would need retained source content or an explicitly retrievable revision, including a way to describe uncommitted files. Different sessions may have different baselines, so avoid an implicit project-wide "last seen" value.

Start with one continuation note supplied separately to the next session. Test whether a fresh agent can identify the missing device check and changed contract without repeating the earlier investigation. Use that trial to decide whether baseline storage belongs in the product.

## Reduce the cost of maintaining records

When managed writes are introduced, favor focused operations: append a progress observation, add an accepted context link, or revise a specified criterion. Show the proposed change and expected source revision, preserve unrelated prose, and report conflicts explicitly.

All interfaces should use the same validation and persistence operations. A convenient interface must not invent its own readiness or acceptance rules. Changes to agreed goals or architectural boundaries continue to require the human decision described in the vision.

Recurring verification expectations can initially be explicit links to small authored documents. If reusable templates become useful, retain which version supplied an expectation and make changes visible. Avoid silently changing existing work when a template changes.

Implementation claims are a later extension of coordinated writes. The operation should recheck readiness before recording one active local claim, keep it separate from the human assignee, and preserve the previous handoff during an explicit takeover. Their value should be established through an actual overlap between runners before building more coordination infrastructure.

## Use code investigation as supporting information

An optional external analysis provider could suggest relevant files, call paths, affected components, and regression checks during grooming. Keep provider-specific behavior outside the core.

Retain source locations, observed revision or working-tree identity, freshness, and analysis limits beside an observation. A code relationship can suggest where to investigate. It does not establish a requirement, a complete cross-service dependency map, or successful device behavior.

Useful findings can become proposed context links or authored conclusions after review. Generated indexes remain disposable. The core reader must continue to work without an analysis provider, model, or server.

Before adding an adapter, compare ordinary investigation with one manually prepared set of attributed findings. Check missed components and false impact claims as well as effort saved.

## Keep observations and commitments distinct

Generated summaries, review suggestions, and mutable progress notes should remain advisory unless explicitly adopted. Linking a changing report as authoritative Context makes its content an input to acceptance fingerprints. Updating the report could then require reassessment even when the actual agreement has not changed.

Promote a useful conclusion into a focused authored record. Retain accepted evidence as an immutable observation. Use disposable views to explain those records, with links back to their sources. This keeps reporting convenient without creating a second authority over the work.

## A small sequence to evaluate

1. Add a read-only intervention view over existing orientation results.
2. Trial a criterion-to-evidence review and a separate continuation note on one real work item. Add grooming proposals only for gaps the task actually has.
3. Ask a fresh agent to resume after a controlled source change, checking current context before acting.
4. Use the observed friction to choose the next increment: source-baseline comparison, one focused write operation, or optional code investigation.

For each trial, record whether the owner identified the needed intervention, the developer avoided repeated investigation, the architect could trace the contract decision, and the agent kept missing verification visible. Also record the effort spent maintaining the new material.

## Questions still to resolve

- What belongs in the shortest useful intervention view, and what should require expansion?
- Where should supplemental handoffs live so they are discoverable without becoming requirements automatically?
- Which observed sources need retained text, and for how long?
- Which record edit is frequent and error-prone enough to justify the first managed write?
- When would a criterion identity be worth the additional authoring convention?

Resolve these through use before establishing new record schemas or mandatory workflow stages.
