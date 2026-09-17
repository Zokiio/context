# Independent review of local orientation and resumption

Reviewed revision `e1c5a344f6afe913ae16f27f701cd0dfbc9f6589` on 2026-09-17. Scope: the specification, resolved wayfinding decisions, repository vision and ADRs, reader/discovery documentation, and current CLI scope, task-context, and orientation contracts. This review used the technical-writing and unslop skills. No product code was changed and no implementation tests were run.

The product direction matches the agreed decisions. Three contract gaps should be fixed before implementing recovery discovery and the public resumption JSON. Compact orientation can proceed independently.

## Required contract fixes

### P2: Reconcile bounded directory enumeration with lexical discovery

[Specification lines 151–153](spec.md#bounded-and-authorized-reading) require both stopping directory enumeration at the first entry-limit breach and discovering observation directories in lexical UUID order. A filesystem directory does not promise that order. Finding its first lexical entries requires enumerating all names before sorting. With a million observation directories, sorting first violates the stated bound; stopping after 200 entries cannot guarantee the prescribed lexical prefix.

The current orientation inventory is not a bounded-enumeration solution to reuse: [inventory.go lines 18–46](../../internal/orientation/inventory.go) walks and collects all paths before sorting. The handoff decision explicitly requires bounded snapshot reads and concrete discovery behavior [lines 57–59](issues/03-handoff-contract.md#responsibility-and-limits).

Fix: separate bounded discovery from presentation ordering. Enumerate in bounded batches, stop once the budget is exceeded, and sort only the observed entries. Guarantee lexical processing when discovery completes; define partial inventory ordering without claiming that it is the global lexical prefix. Add an oversized-directory fixture that checks enumeration work as well as the number of reported files.

### P2: Define recovery from an interrupted publication

[Specification lines 133–143](spec.md#skill-publication-and-competing-notes) create the observation directory before publication and make an unfinished directory leave candidate selection incomplete. [Lines 180 and 196](spec.md#resumption-result) consequently keep recovery unknown and the command partial. There is no convention for retiring an abandoned write.

For example, observation A is complete, a session dies after creating directory B, and a successor reconstructs the task and publishes C. B still makes every later inspection incomplete. Referencing B as a predecessor cannot reconcile it because B has no finalized note. The safe recovery workflow is unspecified beyond deleting cache material, despite interruption recovery being the [agreed handoff purpose](issues/03-handoff-contract.md#missing-or-conflicting-notes).

Fix: specify how a skill explicitly abandons or quarantines an unfinished write after establishing that its writer has stopped, or define a documented task-cache reset procedure with its information-loss consequences. Keep this outside the read-only command. Do not infer abandonment from age alone. Test that interruption, deliberate recovery, and another resume can return to a complete report without silently discarding competing finalized notes.

### P2: Finish the nested JSON contract before agent consumers depend on it

[Specification lines 167–186](spec.md#resumption-result) name top-level fields but leave the actual shapes of `scope`, candidate observations, earlier-observation references, source differences, and diagnostic attribution unspecified. `comparison` is described both as one comparison per candidate and as something that directly has `baselineAvailable: false` when there is no candidate. It is unclear whether it is an object containing an array, an array, or a value whose shape changes by state.

That ambiguity affects the primary agent interface: an implementation and a workflow skill can reasonably choose incompatible paths for candidate IDs, note bodies, completeness, or missing-baseline detection. Existing [task-context result types](../../internal/taskcontext/context.go) and [orientation result types](../../internal/orientation/types.go) provide concrete field contracts; they do not define these new structures. The [contract decision](issues/06-cli-and-note-contract.md#question) specifically requires a concrete output contract.

Fix: add a compact field/type table or JSON schema for the new nested objects, including nullability, required fields, unknown metadata encoding, diagnostic attribution, and candidate snapshot validity versus graph validity. Supply examples for absent, available, conflicting, and incomplete recovery. Use those examples as contract fixtures during implementation.

## Optional operational improvement

Document the cost of append-only checkpoint history and the supported reset or limit-override procedure. [Lines 139 and 151–153](spec.md#skill-publication-and-competing-notes) retain all older observations and inspect every note on each resume. Even when old snapshots are not loaded, a 200-entry limit is eventually exhausted by ordinary successful writes, not only malformed data. The CLI correctly becomes partial, so this is not silent corruption, but the skills should explain what the user can do without accidentally breaking predecessor links. Automatic pruning can remain deferred.

## Review limits

This is a design review, not acceptance of a working implementation. It does not establish CLI performance, correct filesystem race handling, or successful recovery in the real-task trial. Source-link integrity was checked separately by the parent agent.
