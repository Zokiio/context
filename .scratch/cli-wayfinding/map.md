# Direct and resume agent work with the local CLI

Labels: wayfinder:map
Status: open

## Destination

An implementation-ready specification for a usable local CLI that explains where work needs attention and helps a fresh session resume from a handoff. Jira retrieval is an early follow-up after this local milestone.

## Notes

- Planning only. Finish when the specification has no unresolved decisions needed to implement this milestone; implementation is a separate handoff.
- The user chose this scope on 2026-09-17: skills author records and handoffs; the CLI shows needed actions and helps resume; the human directs pickup.
- Consult `grilling` and `domain-modeling` for decisions, `prototype` for concrete behavior trials, and `technical-writing` for the resulting specification. Use `research` for external facts that cannot be resolved from the repository.
- Start from the [research findings](research.md), [existing workflow proposals](../workflow-improvements/discovery.md), [vision](../../docs/vision.md), and [glossary](../../CONTEXT.md). Research suggestions are not accepted requirements.
- Current context, orientation, acceptance checks, discovery, and setup already exist. Extend them rather than planning them again. The research records the source revision inspected.
- Keep execution, readiness, context completeness, acceptance, and human authorization distinct. Skill-produced advice and mutable handoffs do not automatically become authoritative requirements.
- Preserve one authoritative record per decision and requirement. Child tickets hold their eventual answers; this map only indexes them.
- Use the local [wayfinding operations](../../docs/agents/issue-tracker.md#wayfinding-operations): query child files for the frontier, choose the first by number, and save `Status: claimed` before work. Resolve at most one decision per session.
- Prototype artifacts are disposable decision aids, not delivery of the destination. No research tickets are currently needed; completed background reading is captured in the research file.

## Decisions so far

- [What proves the local directing-and-resuming loop is usable?](issues/01-usable-local-loop.md): a fresh session continues an interrupted real task from its ticket reference, using disposable working notes, current context, and visible blockers and verification gaps.
- [What belongs in the shortest useful intervention view?](issues/02-intervention-view.md): broader orientation with needs attention inside it, compact text by default, expanded detail on request, and full JSON using the same facts and checks.
- [What does a handoff retain, and how does a session select it?](issues/03-handoff-contract.md): ticket-based lookup of checkout-local working notes and retained requirement text, with reconstruction when absent and explicit conflicts when notes disagree.
- [What should resumption show when requirements or readiness changed?](issues/04-resume-behavior.md): refreshed facts and explicit differences guide continuation, with conflicts, missing sources, blockers, and verification limits kept visible.

## Not yet specified

No additional unformulated questions are identified. The concrete corrections from the [independent review](independent-review.md) reopened the command and note-format contract ticket. The specification remains under review until that ticket is resolved again.

## Out of scope

- Automatic cache cleanup and extended observation history beyond the current recovery baseline. Evaluate later if real use establishes a need.
- [How should criteria and verification gaps appear during resumption?](issues/05-verification-gaps.md): defer the richer criterion-by-criterion evidence presentation; basic verification gaps remain part of the resumption decision.
- General work-record creation and editing through the CLI. Skills retain authorship for this milestone; existing setup configuration writes remain existing behavior.
- Exclusive implementation claims and automatic pickup. The user directs pickup; same-machine and cross-machine claim design are later work.
- Jira retrieval, parent-chain integration, remote synchronization, and Confluence publication. Jira is an early follow-up; Confluence remains optional.
- A custom agent runtime, scheduling, TUI/web interfaces, mandatory enterprise policy machinery, and automated deployment. They are not needed for the selected local loop.
