---
type: WorkItem
id: e1f2dcee-d054-4311-9719-a55e29f576f5
title: "Show project orientation and work inventory"
triage: ready-for-agent
execution: completed
---

# 01: Show project orientation and work inventory

## What to build

A caller can inspect one explicitly selected local project through `ctx orient`. The report shows authored goals, current commitments, recorded progress, discovered work, and source references in text or JSON. This slice also establishes the authoring rules and completion procedure used throughout the milestone.

## Scope

Start with the shared-reader changes needed by both application operations, then deliver the overview through the CLI. Preserve the existing task-context contract. Shortlisting and semantic decision, acceptance, and dependency checks arrive in later tickets. Report every applicable unsupported check as unknown, mark the report partial, and keep affected work off the shortlist.

Publish the complete agreed record profiles and completion guidance in this slice, before taking acceptance snapshots of those conventions. Retain verification evidence after every slice. Structured acceptance authoring becomes practical with fingerprint output in ticket 04, and validation arrives in ticket 05.

## Acceptance criteria

- [x] One Go orientation operation accepts explicit project scope, allowed document roots, collection limits, and cancellation. It returns structured data and a separate operation error without printing, exiting, writing records, or invoking a model. The CLI receives both application operations as explicit dependencies.
- [x] Shared parsing and authorized source reading support section-presence information and consistent source snapshots without changing the task-context result or selection contract. Existing reader tests remain green, including cycle behavior, external roots, limits, and exit statuses.
- [x] Read the reserved project manifest first and validate its Project type, ID, and title. A missing or invalid manifest in a usable directory produces a partial report. An unusable configured root is an operation failure.
- [x] Discover Markdown records in deterministic relative-path order, including untracked and ignored files. Recognize WorkItem, Decision, and Acceptance identities without treating plain documents or other types as work items. Preserve unknown metadata and identify malformed candidates, missing profile fields, and duplicate typed-record IDs without choosing an arbitrary winner.
- [x] Return authored Goals material and source references, direct Current commitments links, and Open decisions references. Preserve repeated sections and authored order. Missing sections remain unknown. Empty sections and literal None declare none where allowed; malformed prose-only link declarations produce diagnostics.
- [x] Show work-item identity, title, triage, execution, commitment membership, specification references, and recorded progress as distinct facts. Missing or invalid execution remains unknown. Document lifecycle does not substitute for execution or triage.
- [x] Each resolved source is captured once and supplies both parsing bytes and its whole-file digest. Retain distinct inclusion reasons and authored links. Preserve UTF-8 validation, file-alias deduplication, authorized external document roots, and bundle-only record targets. Do not traverse directory symlink aliases or expand ordinary document prose.
- [x] Apply the shared default limits of 100 files and 1,048,576 source bytes to the manifest, inventory, and linked documents. Admit whole files in the specified order and stop at the first breach. Identify the first omission and known pending sources without claiming an exhaustive undiscovered inventory.
- [x] Expose the specified version-1 JSON envelope and a compact default text report with the same known facts. Lists remain arrays and unavailable values are explicit. Include source paths and whole-file digests without returning every full source body. Unsupported readiness checks and unavailable fingerprints must not look successful.
- [x] Accept required project scope, repeated allowed-source roots, positive limit overrides, and the JSON flag without requiring a ticket. Reject unexpected positionals and invalid flags. Return 0 for complete supported evaluations, 1 for partial reports, and 2 for invocation, root, cancellation, application, or output failures. Keep data diagnostics in the report and operation errors on stderr.
- [x] Update tracker conventions and relevant authoring workflows with the agreed Project, WorkItem, Decision, and Acceptance profiles, fingerprint rules, and staged completion procedure. Explicit file edits remain the authoring mechanism. Distinguish workflow decisions, human approvals, execution state, and acceptance evidence.
- [x] Adopt explicit execution metadata and empty blocking-decision declarations for these eight orientation tickets as their actual state permits. Preserve existing reader records for the later migration. During bootstrap, retain the approved blocker graph and existing task-context workflow without claiming that unsupported orientation checks have passed.
- [x] The completion procedure retains each slice's actual criteria results, evidence, and tested revision. It computes acceptance fingerprints after final requirement and checklist edits, preserves old decisions, and requires reassessment when linked context changes. Describe how to derive the specified encoding reproducibly and how later tickets introduce acceptance authoring and validation.
- [x] Tests through the public Go operation use real temporary projects, including an independent project outside Git and an unrelated working directory. Cover malformed inventory, duplicate IDs, exact-fit and overflowing budgets, symlink boundaries, encoding failures, and preserved partial facts. A small CLI suite covers both formats and failures, including injected operation and writer errors. Both commands leave source files and Git state unchanged.

## Blocked by

None

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
- [Acceptance authoring and fingerprint reference](../../../../docs/agents/acceptance.md)
- [Existing reader contract](../../../context-reader/spec.md)

## Comments

Previous acceptance decision, preserved during reassessment:
- [Current acceptance](../acceptances/01-overview-20260916.md)

- Implementation started on branch feat/session-orientation. The coordinating agent owns the ticket; orientation_core handles the Go operation and shared reader, and orientation_cli handles command output and CLI tests. Baseline tests passed before implementation.

- Verified all 14 criteria. [Immutable verification evidence](../../../session-orientation/evidence/01-overview.md) and its source manifest retain the actual tested working tree. Structured acceptance awaits tickets 04 and 05; unsupported readiness remains unknown.

- A separate [shared-reader compatibility review](../../../session-orientation/evidence/01-context-compatibility-review.md) found no regressions in 11 targeted comparisons between 5742063 and 9666b19.

## Acceptance

- [Current acceptance](../acceptances/01-authoring-reassessment-20260916.md)
