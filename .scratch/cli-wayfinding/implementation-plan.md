# Implementation tickets for local orientation and resumption

The user approved this seven-ticket breakdown. The [reviewed specification](spec.md) defines the contract; each linked WorkItem owns its acceptance criteria and execution state. This index does not duplicate those records or add them to project commitments.

| Ticket | Blocked by | Delivered behavior |
| --- | --- | --- |
| [Share captured sources between readers](../records/cli-wayfinding/issues/01-share-captured-sources.md) | None | Reusable captured bytes and collection budget while standalone readers remain compatible. |
| [Show compact project orientation](../records/cli-wayfinding/issues/02-compact-orientation.md) | None | Compact operator overview, grouped intervention, expanded detail, and full JSON. |
| [Resume a task without recovery notes](../records/cli-wayfinding/issues/03-resume-without-notes.md) | Share captured sources between readers | Scoped resumption with refreshed task/project facts and explicit absent-note behavior. |
| [Resume from a retained checkpoint](../records/cli-wayfinding/issues/04-retained-checkpoint.md) | Resume a task without recovery notes | Validated retained source text, bounded authorized reads, and source comparisons. |
| [Handle competing and interrupted checkpoints](../records/cli-wayfinding/issues/05-competing-checkpoints.md) | Resume from a retained checkpoint | Complete history evaluation, conflicts, incomplete publication, and final recovery JSON. |
| [Teach skills to maintain and recover working notes](../records/cli-wayfinding/issues/06-recovery-skills.md) | Handle competing and interrupted checkpoints | Skill-guided checkpoint publication, continuation, reconciliation, quarantine, and reset. |
| [Verify the complete interrupted-task workflow](../records/cli-wayfinding/issues/07-verify-workflow.md) | Show compact project orientation; Teach skills to maintain and recover working notes | Final reference, real-task continuation trial, independent project arrangements, and retained acceptance evidence. |

The first two tickets have no work blockers. The recovery chain uses explicit intermediate failures until later slices provide complete history support; an intermediate limitation must never look like successful final-contract behavior. Each slice owns its focused checks. The final trial verifies the assembled workflow.

## Coverage

| Specification area | Owning ticket |
| --- | --- |
| Shared current-source capture and budget; existing-reader compatibility | Share captured sources between readers |
| Compact/detail orientation, grouping, full JSON, visible incomplete information | Show compact project orientation |
| Command selection, checkout scope, identity, absent-cache report and exit behavior | Resume a task without recovery notes |
| Note profile, snapshot validation, source authorization, comparison, bounded file reads | Resume from a retained checkpoint |
| Bounded inventory, graph validity, concurrent leaves, diagnostics, final JSON examples | Handle competing and interrupted checkpoints |
| Publication, actual observation baselines, deliberate recovery, cache lifecycle guidance | Teach skills to maintain and recover working notes |
| User documentation, cross-checkout/non-Git trials, full verification and acceptance | Verify the complete interrupted-task workflow |

Use the repository's task-context and recorded-acceptance procedures when implementing. The illustrative JSON projections require real source fixtures and computed digests before serving as implementation checks. No implementation is claimed by publication of these tickets.

## Publication verification

Built the current reader from revision `4459638` and read all seven tickets through `context` with the repository authorized for supporting sources. Every result was complete with complete traversal. Orientation recognized all seven unambiguous WorkItems with unstarted execution and ready-for-agent triage. The two dependency-free tickets have ready checks; the remaining five are blocked by their declared prerequisites. All seven remain outside current commitments, so they are not yet pickup-shortlist entries. This verifies record publication, not implementation acceptance.

## Implementation verification

All seven tickets are implemented and completed. The [combined evidence](evidence/07-workflow-verification.md) records final source checks, independent reviews, a real fresh-agent continuation and controlled recovery cases. The [final dependency observation](evidence/07-final-dependency-graph.json) verifies all seven current acceptances as valid and all supported checks as passing. Earlier publication and slice evidence retain their original tested identities.

## Operator feedback

[Keep compact orientation focused on work](../records/cli-wayfinding/issues/08-compact-paths.md) follows the live demonstration. It shortens actionable paths and moves repeated provenance to detail and JSON. The original seven-ticket milestone retains its historical verification.
