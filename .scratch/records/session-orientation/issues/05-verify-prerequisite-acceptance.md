---
type: WorkItem
id: f2a2f976-d844-4e27-86fc-a3ec9b0025e3
title: "Satisfy direct prerequisites using recorded acceptance"
triage: ready-for-agent
execution: completed
---

# 05: Satisfy direct prerequisites using recorded acceptance

## What to build

A ticket can become eligible when its direct prerequisites have completed with usable, current acceptance and retained evidence. Orientation explains the basis for acceptance and distinguishes it from an execution flag.

## Scope

Evaluate the full Acceptance profile and direct prerequisite edges. In this slice, an edge can pass only when the completed prerequisite explicitly declares no work dependencies and its blocking decisions are resolved. Nonempty prerequisite chains remain unsupported and unknown until ticket 06. Missing declarations remain unknown.

## Acceptance criteria

- [x] A completed prerequisite must link exactly one current Acceptance record of the correct type and unambiguous identity. Match its projectId and workItemId to the selected project and subject. Missing, multiple, wrong-type, or mismatched current links cannot satisfy the edge. Never choose acceptance by filename or timestamp.
- [x] Validate all required acceptance metadata, including the actor's human or workflow kind and identity, an explicit-offset decision timestamp, tested-revision origin and value, fingerprint version, and both subject fingerprints. Keep optional human approvals separately attributed with their own actor, time, and revision.
- [x] Acceptance covers the entire current criteria section and ticket requirements. Compare the recorded fingerprints with the operation's current values. Missing or malformed acceptance information is unknown; changed requirements are a known failure requiring reassessment.
- [x] Parse Requirements and Evidence snapshots as the specified Markdown list entries, each with one file link and one inline-code SHA-256 digest. Diagnose malformed, ambiguous, conflicting, and missing entries. Require at least one evidence source.
- [x] Require snapshots for the subject's selected Spec and Context documents and linked decisions. Check current membership and whole-file digests. Additional explicit requirements must also remain available and unchanged.
- [x] Resolve evidence and requirement documents through authorized local roots using captured UTF-8 bytes, common budgets, and source provenance. Do not fetch remote observations or read binary evidence. Acceptance and other record targets stay within the project bundle.
- [x] Reject self-snapshots and self-evidence. A subject-ticket snapshot uses its requirement fingerprint rather than the whole-file digest. Preserve prior acceptance decisions when a new current decision is authored.
- [x] Missing or unreadable evidence makes acceptance unknown; changed evidence makes it stale and blocks the edge. A changed current Git HEAD alone does not invalidate historical acceptance. The report retains the actual tested revision without claiming to certify the current checkout or rerun tests.
- [x] A direct edge passes only for a completed prerequisite with valid fresh acceptance, resolved blocking decisions, and an explicitly empty Blocked by declaration. Unstarted, in-progress, and cancelled prerequisites are known blockers. A deeper chain cannot pass through the temporary lack of recursive evaluation.
- [x] Text and JSON expose acceptance status, actor attribution, historical revision, evidence references, and diagnostic reasons separately from execution, readiness, and eligibility. A valid accepted prerequisite can release dependent work when its other conditions pass. Known stale acceptance alone remains a complete evaluation with exit 0; unknown or unsupported required information produces exit 1.
- [x] Public-operation tests cover valid acceptance, subject mismatches, malformed metadata, unsupported fingerprint versions, missing or multiple current links, changed fingerprints, changed source membership, extra requirements, absent or changed evidence, self-reference, and separate human approval. Include a ready dependent, a cancelled prerequisite, and a non-leaf prerequisite that remains unknown.
- [x] Validate current acceptance records for earlier completed orientation slices. Use retained evidence to author missing records and reassess any changed requirements without fabricating new test runs. Record acceptance for this slice after its criteria are verified, and exercise the documented procedure for subsequent slice closeout.

## Blocked by

- [03: Explain blocking decisions](03-explain-blocking-decisions.md)
- [04: Expose reproducible requirement fingerprints](04-expose-requirement-fingerprints.md)

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
- [Acceptance authoring and fingerprint reference](../../../../docs/agents/acceptance.md)

## Comments

Previous acceptance decision, preserved during reassessment:
- [Current acceptance](../acceptances/05-review-reassessment-20260916.md)

Previous acceptance decision, preserved during reassessment:
- [Current acceptance](../acceptances/05-acceptance-20260916.md)

- Verified all 12 criteria. [Immutable verification evidence](../../../session-orientation/evidence/05-acceptance.md) retains the actual tested source identity and results.

## Acceptance

- [Current acceptance](../acceptances/05-evidence-correction-20260916.md)
