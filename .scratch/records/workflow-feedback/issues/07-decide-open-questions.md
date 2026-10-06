---
type: WorkItem
id: 5cf0853a-a513-4a82-ba90-bcf685b4bb75
title: Decide an implementer's open questions before review
triage: needs-triage
execution: unstarted
---

# Decide an implementer's open questions before review

Recorded on 2026-10-06 from the Axpilot #275 retrospective's proposal 5, carried into the batch retrospective of the #266 and #275 runs.

## Evidence

In #275, the checkpoint 2 implementer reported an open question: `AskUserQuestion` would show as a failed tool step. The orchestrator noted it for the pull request and commissioned the runner and both reviews. The Standards review failed on exactly that point, which cost a repair and one more round of three subagents.

The orchestrate skill resolves material choices before implementation starts. It says nothing about questions an implementer raises after implementation.

## Scope

Change orchestrate's "Check and correct an identified revision" section. When an implementing or repairing agent reports an open question about user-visible behavior, the orchestrator decides it before commissioning the runner and reviews. It decides from the requirements, or asks the user under the project's review policy. A change that the decision requires is committed before review. Review axes and gates are unchanged.

## Acceptance criteria

- Q1: When an implementing or repairing agent reports an open question about user-visible behavior, the orchestrate skill requires the orchestrator to decide it before commissioning the runner and reviews on that revision.
- Q2: The orchestrator decides from the requirements or with the user under the project's review policy, and commits any change the decision requires before review.
- Q3: The canonical template and generated development copy agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Orchestrate template](../../../../internal/initialization/templates/orchestrate.md)
- [Domain glossary](../../../../CONTEXT.md)
