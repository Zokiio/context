---
type: WorkItem
id: 583073dd-c167-494a-96d5-5606ef1e988f
title: "Migrate project records and prepare the pickup trial"
triage: ready-for-agent
execution: unstarted
---

# 07: Migrate project records and prepare the pickup trial

## What to build

The real project uses the new authoring conventions with truthful execution states and retained acceptance evidence. Its five completed reader tickets stop looking available for new work, and a verified reader build is ready for a fresh-session trial.

## Scope

Apply the conventions and completion procedure established in earlier tickets. Backfill legacy reader acceptance and finish project-record migration. This ticket owns the final application checks and build handoff before the real trial. Ticket 08 must remain unstarted until a fresh session selects it.

## Acceptance criteria

- [ ] Review each of the five reader tickets against its own recorded criteria and evidence. Add supported execution metadata and explicit relationship declarations without changing triage into execution state or inventing unsupported facts.
- [ ] Author usable current acceptance records for completed reader work with current subject fingerprints and required snapshots. Preserve the original evidence's actor, tested revision, and source manifests. Record the merged revision separately where relevant; do not substitute it for the revision of an earlier uncommitted test run.
- [ ] Each backfilled decision identifies its actual author and decision time. Attribute human approval only when a source establishes it. Preserve historical decisions and evidence, and do not imply that backfill itself reran tests.
- [ ] Populate project goals, explicit commitments, and decision links from their authoritative sources. Include the existing final trial ticket as an unstarted commitment. Do not create a synthetic task merely to produce a shortlist entry.
- [ ] Verify execution, criteria, relationships, evidence, and acceptance for completed orientation slices. Reassess any acceptance made stale by actual requirement or convention changes, preserving prior decisions. Missing evidence remains an explicit gap to resolve before the trial.
- [ ] Before this ticket's closeout, verify that its pending completion is the remaining work prerequisite preventing the trial ticket from being eligible. Resolve other missing records, stale acceptance, ambiguous identities, or decision blockers. Do not require the trial ticket to be eligible before this prerequisite is completed.
- [ ] Run the normal Go tests, race tests, vet, CLI build, and declared minimum-Go checks before the real workflow trial. Record actual commands, outcomes, toolchain versions, and the tested code revision. Preserve a source manifest when a commit alone does not identify the tested working tree.
- [ ] Retain the exact reader build for the fresh session with its digest, revision or source manifest, and the successful preflight evidence. Document the explicit project and allowed-source scope needed to use it.
- [ ] Run both orientation formats on the migrated project and verify that all five completed reader tickets stay off the shortlist. Compare source files and Git state before and after the read-only commands. Keep planned authoring edits separate from those observations.
- [ ] Record this ticket's own evidence and acceptance after its criteria and required documents are settled. Its acceptance must cover the final record changes and successful preflight checks. Do not record the final trial as already performed or change ticket 08 to in-progress.
- [ ] Leave a handoff that lets a fresh session begin with orientation, choose eligible work, and then obtain task context. The fresh-session prompt supplies project and reader scope without preselecting a ticket or substituting a manually assembled task context.

## Blocked by

- [06: Evaluate prerequisite chains and cycles](06-evaluate-prerequisite-chains.md)

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
- [Acceptance authoring and fingerprint reference](../../../../docs/agents/acceptance.md)
- [Reader ticket 01](../../context-reader/issues/01-read-okf-ticket.md)
- [Reader ticket 02](../../context-reader/issues/02-include-project-documents.md)
- [Reader ticket 03](../../context-reader/issues/03-follow-blocker-tickets.md)
- [Reader ticket 04](../../context-reader/issues/04-allow-external-context.md)
- [Reader ticket 05](../../context-reader/issues/05-bound-context-collection.md)
- [Reader acceptance evidence](../../../context-reader/acceptance-trial.md)
- [Original reader trial manifest](../../../context-reader/acceptance-trial-manifest.json)
- [Reader verification after review](../../../context-reader/review-verification-manifest.json)
