---
type: WorkItem
id: 940635b3-0697-45c6-804b-43f9ec1d3e58
title: "List eligible work with reasons"
triage: ready-for-agent
execution: completed
---

# 02: List eligible work with reasons

## What to build

Orientation identifies suitable self-contained work and explains why each ticket is eligible or excluded. A human or agent can choose among committed, unstarted tickets while seeing active work and the remaining backlog separately.

## Scope

Establish the four readiness checks and the separate eligibility rules. This slice can prove the dependency and decision checks only for explicit declarations of none. Nonempty relationships whose evaluation is not yet supported remain unknown and produce partial reports.

## Acceptance criteria

- [x] Report criteria, selected context, work dependencies, and blocking decisions as separate pass, fail, or unknown checks with reasons. A known failed check makes readiness blocked; otherwise unknown takes precedence over ready. Retain unknown explanations alongside known failures.
- [x] Require at least one nonempty authored acceptance criterion. A missing Acceptance criteria section is unknown, while an explicitly empty section is a known failure. Check structure without judging substantive adequacy.
- [x] Check the availability of explicitly selected Spec and Context documents using the established source rules. Spec remains optional for self-contained work. Missing, unreadable, invalid, or unauthorized selected sources make the affected condition unknown and the report partial.
- [x] Explicitly empty Blocked by and Blocked by decisions sections, including literal None, pass their respective checks. Missing or malformed declarations remain unknown. Nonempty relationships cannot pass merely because their later evaluation is unimplemented.
- [x] A ticket enters the shortlist only when it is unstarted, explicitly committed, ready-for-agent, and ready. Return each condition and any exclusion reasons separately. Missing execution or commitment information excludes work even when its readiness checks pass.
- [x] Keep in-progress work separate. Show uncommitted unfinished work as backlog, including ready work. Retain completed and cancelled records for inventory and progress, and never offer them as new work.
- [x] Preserve known positive commitments when the commitment list is incomplete and leave unproven membership unknown. Suppress the shortlist after incomplete inventory or invalid project identity. Ambiguous records cannot qualify, while unrelated unambiguous work remains inspectable.
- [x] Missing project information that does not affect an item's identity, scope, membership, or checks does not automatically change that item's readiness. Preserve useful established facts in partial reports and explain any global shortlist suppression.
- [x] Present eligible work in authored commitment order and remaining records by stable ID and resolved path. Do not rank or recommend a single task. Include the ticket reference needed to request detailed task context.
- [x] Text and JSON expose the same eligibility, checks, reasons, and work groups. Complete supported evaluations with known blockers or empty shortlists return 0; unavailable or unsupported information returns a partial report and 1.
- [x] Public-operation tests cover mixed execution and triage states, missing and empty sections, unavailable context, malformed commitments, duplicate identities, partial inventory, and a known failure together with an unknown check. Small CLI tests verify useful report facts and exit behavior.
- [x] Retain this slice's actual verification evidence and tested revision under the completion procedure established by ticket 01.

## Blocked by

- [01: Show project orientation and work inventory](01-show-project-orientation.md)

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
- [Acceptance authoring and fingerprint reference](../../../../docs/agents/acceptance.md)

## Comments

- Verified all 12 criteria. [Immutable verification evidence](../../../session-orientation/evidence/02-eligibility.md) retains the actual tested source identity and results.

## Acceptance

- [Current acceptance](../acceptances/02-eligibility-20260916.md)
