# Tracker intake splits ownership at agreement and lets a saved trigger queue work

Draft. The user made these choices on 25 October 2026 while planning [factory intake](../../.scratch/factory-intake/spec.md). The vision and tracker-snapshot text listed under Consequences changes when this decision merges.

## Context

The factory starts only work in Current commitments, and only a person adds work there. The vision describes pickup triggers and two ownership modes for tracker fields: record-owned, where the connector publishes from the record, and tracker-owned, where local copies are snapshots and edits go through the connector. The [tracker snapshot profile](../tracker-snapshots.md) keeps the tracker authoritative for requirements and routes requirement changes through the tracker.

Factory refinement rewrites acceptance criteria, and a person agrees to them before the build. Under the snapshot rule, every refined item would need a manual tracker edit and another refresh before it could build. The Opsbase grooming trial refined imported GitHub issues and planned to propose the changes back by hand.

Text in a tracker comes from people outside the project record. The builder prompt includes ticket text verbatim. The test author and builder can run `go test` and other commands an operator allows. With the ask policy set to never ask before the build, an issue could reach a builder without anyone reading it.

## Decision

Ownership of an imported item is per field and changes once, at agreement. The tracker owns the request: title, description, labels and the triage they carry, state, assignee, source identity and timestamps. Acceptance criteria and blockers follow the tracker until a person agrees what done means. From then on the record owns them. A later tracker change to the request or to those fields holds the item for a person to keep the agreed version or plan again. The record always owns execution, decision blockers, Spec and Context links, Acceptance, Comments and queue membership.

A pickup trigger match makes an item a candidate and nothing more. A human project administrator may save a trigger whose action is to queue its matches. Queueing is then that person's standing decision, carried out on their behalf with their capabilities rechecked each time and recorded as delegated, as the ask policy already works. Readiness and eligibility are computed as before and reported apart from the match.

On an imported item, the first agreement on what done means is always a person's, whatever the ask policy says. Before it, only the read-only refiner runs on the item's text. After it, the ask policy applies as it does to native work.

## Alternatives considered

Keeping the tracker authoritative for every field preserves one source. It requires a tracker edit for each refined item, and write-back is not supported, so the person would copy criteria by hand before every build.

Copying an issue once and never refreshing it is simplest. Tracker edits after import would then be invisible, which the vision's freshness requirement rules out.

Letting imported items follow the ask policy like native ones gives the most autonomy. It lets text written outside the project reach an agent that can run commands with nobody reading it. SuperPlane and Open-Inspect both work this way; Open-Inspect relies on a marked prompt block alone.

A match that only imports a draft, with a person queueing each item, keeps today's guarantee but leaves the gap this decision closes.

## Consequences

The vision's sentence "A matching trigger makes a work item a candidate for pickup; it does not establish readiness or permission to start" still holds for the match. A saved queueing trigger is a separate, attributed permission. The vision's two field-ownership modes gain a third arrangement, where criteria and blockers switch from tracker-owned to record-owned at agreement. The tracker snapshot profile's rule that requirement changes go through the tracker applies only until agreement for factory-imported items. The hand-authored snapshot convention is unchanged.

Current commitments can now contain work no person queued individually. Each such entry traces to a trigger and the person who saved it.

The ask policy no longer governs the first plan decision on imported work. A project that wants imported work to build unattended end to end would need a further decision.

Refreshes must never rewrite an item while work holds it or after it completes, because admission and acceptance fingerprint the ticket.
