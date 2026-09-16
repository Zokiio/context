---
type: WorkItem
id: 9881a140-ff67-47f8-8b61-a7cc173743b2
title: "Verify fresh-session pickup on the real project"
triage: ready-for-agent
execution: completed
---

# 08: Verify fresh-session pickup on the real project

## What to build

A fresh agent session uses orientation to identify this real committed work, explains its eligibility, and obtains its detailed task context. Recorded observations demonstrate the intended workflow on the project's own records.

## Scope

Begin this ticket through orientation while it is still unstarted. The predecessor owns the completed preflight checks and retained reader build. This ticket verifies the pickup workflow and records its result; checking a box or leaving an execution field unchanged cannot substitute for the required order.

## Acceptance criteria

- [x] Use the reader build and explicit project scope handed off by the predecessor, with its retained successful preflight evidence. Orientation establishes the predecessor's completed and accepted state at pickup. Do not perform this ticket's verification work before its orientation-based selection.
- [x] A fresh agent session begins with `ctx orient` on the real project while this ticket remains unstarted and explicitly committed. Supply scope and the workflow objective without naming the ticket to choose or passing it the earlier planning conversation.
- [x] The session selects this ticket from the observed eligible list and explains each required condition: unstarted execution, current commitment, ready-for-agent triage, and passing readiness checks. The command presents an explained list and does not choose or assign the work itself.
- [x] After selection, use the existing task-context workflow to obtain the ticket's full authored requirements, recursive blockers, and explicitly linked documents. Verify that the result is complete and attributable before recording the work as in progress.
- [x] Record exact orientation and context invocations, reader build identity, code revision or source manifest, source paths and digests, the selected ticket, observed eligibility reasons, completeness values, and exit statuses. Preserve the state observed at pickup rather than silently replacing it with later record bytes.
- [x] Compare text and JSON orientation facts and verify that the five completed reader tickets remain absent from the shortlist. Retain the useful inventory and acceptance explanations for completed work.
- [x] Verify that orientation and task-context reads leave project files and Git state unchanged. Separate explicit workflow state and evidence edits from the read-only command observations.
- [x] Record relevant context that lacks an authored link separately from reader defects. Do not claim automatic relevance discovery, independent evidence grading, or acceptance of the current checkout merely from historical revisions.
- [x] Preserve actual failures and subsequent verification if the trial uncovers a defect. Do not mark this ticket or the milestone accepted until the required checks and the real pickup trial have recorded successful outcomes.
- [x] Complete the ticket through the established authoring workflow, retaining trial evidence and current acceptance with truthful actor and revision attribution. This ticket's completion does not automatically close or rewrite the specification.

## Blocked by

- [07: Migrate project records and prepare the pickup trial](07-migrate-records-and-prepare-trial.md)

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
- [Task-context workflow](../../../../.agents/skills/task-context/SKILL.md)

## Comments

The fresh workflow actor selected this ticket from the real orientation report, consumed all 26 context sources before recording in-progress execution, and retained successful observations for criteria 1–9. See the immutable [trial report](../../../session-orientation/evidence/08-pickup-trial.md) and [observation manifest](../../../session-orientation/evidence/08-pickup-trial.json), which preserve the original unstarted pickup state.

The Codex coordinating agent reviewed that evidence, settled this checklist, and completed the acceptance-authoring workflow. The current decision retains the original reader source identity and names this separate closeout judgment. [Post-authoring validation](../../../session-orientation/evidence/08-closeout-observation.json) records current acceptance, readiness, and the empty shortlist. The specification remains unchanged.

## Acceptance

- [Current acceptance](../acceptances/08-pickup-trial-20260916.md)
