---
type: WorkItem
id: a1a394f4-7286-4fd4-8a70-b5558d8e157a
title: "Keep compact orientation focused on work"
triage: ready-for-agent
execution: completed
---

# Keep compact orientation focused on work

## What to build

Apply the user's feedback from the live operator-view demonstration: repeated absolute paths and source relationships take too much space. Show the selected record-store root once, keep short actionable paths, and reserve routine provenance for full detail and JSON.

This follow-up revises only the compact presentation rules in the earlier specification. Its instructions supersede the earlier requirement to print the complete project-reference listing in compact text. The final shared specification and reference must describe this behavior before acceptance.

## Acceptance criteria

- [x] The compact header identifies the record-store root once. Authored goals remain verbatim; do not add repeated project.md source/relationship lines or an inventory of routine project references below them.
- [x] Task selectors and actionable finding paths inside the record store are relative to that declared root, retaining directories so equal filenames remain distinct. Keep paths outside the record store explicit rather than suggesting they are inside it. If a reliable root is unavailable, preserve the original path. Do not infer a Git root or use cwd as a substitute.
- [x] Routine commitment, in-progress, shortlist and affected-work rows omit repeated from/link provenance. Findings retain the path needed to locate a problem; unresolved or ambiguous relationships retain enough referring-path/link detail to distinguish and repair them. Do not alter cause grouping, order, warnings, unknown states, readiness or pickup semantics.
- [x] Full detail and JSON retain their existing exact source paths and provenance. Preserve the existing result; compact formatting must not mutate it. Workspace output and resume output remain unchanged.
- [x] Verify internal/external paths, sibling-prefix boundaries, same-filename records, unknown project identity, relationship failures and retained detail/JSON using meaningful existing or focused checks. Rebuild and inspect the live demo, update the reference and existing PR, and retain evidence/acceptance with affected prerequisite decisions reassessed.

## Blocked by

- [Show compact project orientation](02-compact-orientation.md)

## Blocked by decisions

None

## Spec

- [Local orientation and resumption](../../../cli-wayfinding/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Reader contracts](../../../../docs/readers.md)
- [Discovery contracts](../../../../docs/discovery.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)


## Comments

The user requested shorter actionable paths after the live operator-view demonstration. This ticket owns that presentation refinement. Verification output and acceptance history are local files excluded from Git, as described in the [storage guidance](../../../../docs/agents/issue-tracker.md#local-verification-records).

## Acceptance

- [Local acceptance](../acceptances/08-compact-paths-compact-paths-20260919.md)
