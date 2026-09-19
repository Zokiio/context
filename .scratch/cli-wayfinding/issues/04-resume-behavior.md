# What should resumption show when requirements or readiness changed?

Type: prototype
Labels: wayfinder:prototype
Status: resolved
Blocked by: 03
Parent: [Direct and resume agent work with the local CLI](../map.md)

## Question

Given the agreed handoff contract, what CLI interaction lets a fresh session resume without treating old observations as current permission? Prototype unchanged context, changed requirements, missing baseline sources, a newly unresolved decision, and incomplete verification.

Use the [agreed finish line and continuation defaults](01-usable-local-loop.md#answer). Include the basic view of verification gaps and unsupported test claims here. The richer criterion-by-criterion evidence presentation is outside this milestone.

The [resolved handoff contract](03-handoff-contract.md#answer) establishes ticket-based lookup within the current checkout, retained previous requirement text, reconstruction when no note exists, and preservation of conflicting notes. Include these cases in the prototype. Use earlier text to explain changes where available; make missing or incomplete observations explicit rather than inferring a complete comparison.

Decide which differences the CLI can establish, which remain unknown, what refreshes from current orientation/context, and how the user finds the relevant evidence and next action. Specify when continuation needs a new human decision about intent or design, keeping any such decision distinct from completion Acceptance. Preserve independently valid work and avoid certifying that a model understood or followed context.

Inputs: [research findings](../research.md), [reader reference](../../../docs/readers.md), [acceptance decision](../../../docs/adr/0003-recorded-acceptance-for-readiness.md).

## Prototype

- Archived primary source: branch `feat/prototype-cli-resumption`, commit `96f8afb6c86cb8de96ae1015ecbdf1b8fccf5fd9`, file `internal/cli/resume.prototype.html`. Recover with `git show 96f8afb6c86cb8de96ae1015ecbdf1b8fccf5fd9:internal/cli/resume.prototype.html`. This self-contained demo has free-play controls and guided cases for unchanged or changed requirements, absent or conflicting notes, missing sources, a recorded blocking decision, and partial verification.
- A disposable preview copy remains at `.cache/cli-wayfinding/resume.prototype.html`. The branch retains the primary source; the prototype was removed from the production source directory in the current checkout.
- Synthetic observations are separate from proposed skill behavior. Requirement diffs do not imply automatic semantic impact analysis or authorization. Retained historical test evidence does not certify current code or device behavior.
- The user responded “ok” when asked whether the suggested responses matched the desired continuation behavior, with attention directed to changed requirements and conflicting notes. This is agreement on behavior, not evidence that every demo control was exercised. This is not an implementation of context comparison, note persistence, or an agent runtime.

## Answer

The resumption presentation separates observed CLI facts from skill-guided action. It applies the previously agreed [finish line](01-usable-local-loop.md#answer) and [handoff contract](03-handoff-contract.md#answer), illustrated by the archived demo.

Refresh current requirements, task records, and files before choosing the next step. Report the current checkout, note availability or conflicts, observed requirement differences, recorded blockers, and basic verification gaps. A complete report of those facts is not permission to implement or an assertion of completion.

| Observed condition | Report and continuation behavior |
| --- | --- |
| Requirements match the retained text | Say that the observed requirement text agrees. Inspect current code, state the next action briefly, and continue within the agreement. Do not imply that matching requirements prove unchanged code. |
| Requirements changed | Show previous and current wording with the source reference. The agent assesses implementation impact, adapts within agreed scope, and asks for unsettled scope or architecture decisions. A textual difference alone is not an automatic stop or a semantic judgment by the CLI. |
| Recovery note absent | State that earlier approach and next step are unknown. Reconstruct from records and code; ask only if missing information prevents a sound decision. |
| Notes conflict | Preserve and expose both accounts. Inspect code and evidence to reconcile them; do not choose by timestamp. Ask only if the unresolved conflict prevents a sound decision. |
| Previous source text absent | Show current requirements and identify the incomplete comparison. Do not invent a change history. |
| Current required source unavailable | Mark the report incomplete and recover the source. Independent investigation may continue; do not infer the missing requirement. |
| Recorded blocking decision open | Show the decision and affected work. Wait on implementation dependent on that answer while allowing independent investigation. |
| Test claim lacks result or tested revision | Report that the claim does not establish completion. Rerun relevant checks before relying on it for completion. |
| Historical local result retained | Preserve its origin and tested revision. Keep missing device/environment verification visible; the old result does not certify current changes or the whole task. |

Skills guide investigation, questions, verification, and updates to working notes. The CLI supplies facts and comparisons. These responsibilities do not add an agent runtime, an automatic semantic review, or a criterion-coverage engine.

The prototype settles behavior and presentation. Exact command names, output contracts, snapshot limits, source authorization, and safe skill-authoring conventions still need a concrete specification. Those questions are now precise enough for the next contract decision.
