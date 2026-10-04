---
type: WorkItem
id: 2655e16f-84ba-420a-a054-39eabc5103d5
title: Return short check and review summaries
triage: ready-for-agent
execution: in-progress
---

# Return short check and review summaries

Proposed on 2026-10-04 in the workflow adaptation specification. Runner and reviewer reports in the Axpilot runs returned to the orchestrator in full.

## Scope

Change the verification brief so runners, reviewers, and the combined-result verifier keep full reports in local evidence files and return short summaries. Review axes, gates, and renewal rules are unchanged.

## Acceptance criteria

- S1: The verification brief requires runners, reviewers, and the combined-result verifier to write the full report to a local evidence file and return a summary of about 400 words or fewer.
- S2: Each summary states the verdict and evaluated revision, cites criterion identifiers for its findings, and names the evidence file.
- S3: The orchestrator passes full reports by path to repair dispatches and acceptance. Standards and Specification reports remain separate.
- S4: The canonical templates and generated development copies agree, and `go test ./internal/initialization ./internal/cli` passes.

## Spec

- [Workflow adaptation](../../../workflow-adaptation/spec.md)

## Blocked by

None

## Blocked by decisions

None

## Context

- [Domain glossary](../../../../CONTEXT.md)
