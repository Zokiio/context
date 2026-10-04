---
type: WorkItem
id: 559f0bd2-4293-4775-a340-0ddf6c3d2a04
title: Review a work item's sessions through a retrospective
triage: ready-for-agent
execution: unstarted
---

# Review a work item's sessions through a retrospective

Proposed on 2026-10-04 in the workflow adaptation specification. The measures and findings for the Axpilot runs came from a manual inspection of their transcripts.

## Scope

Add an explicitly invoked `retro` skill that measures named sessions of one work item and proposes environment improvements, each with a named home. It writes nothing before the user approves. Automatic invocation, scheduled retrospectives, and changes to orchestrate are outside scope.

## Acceptance criteria

- R1: The skill reviews named sessions of one work item. It reads Claude Code or Codex transcripts and asks for their location when it cannot identify them. It reads the work item's records through the installed task-context procedure, plus any orchestration plan, checkpoint files, progress, decisions, and retained reports.
- R2: The report gives wall time, subagent count, token use, production and test line changes, review findings that led to repairs, and user interventions, each with its source. Unavailable measures appear as gaps.
- R3: Each proposal names its home: project guidance, an automated project check, a tracker follow-up, or Waymark feedback. Mechanical mistakes are proposed as checks rather than written rules.
- R4: The skill changes no project file, record, or tracker before the user approves a proposal. Approved guidance records its source and scope. The report stays local and out of Git.
- R5: Installation follows the linked decision. The canonical template and generated copies agree, links resolve in custom directories, and `go test ./internal/initialization ./internal/cli` passes.
- R6: An independent run on a retained 2026-10-04 Axpilot session reports measures consistent with the specification's Problem table.

## Spec

- [Workflow adaptation](../../../workflow-adaptation/spec.md)

## Blocked by

None

## Blocked by decisions

- [Choose where the retrospective skill is installed](../decisions/02-retrospective-distribution.md)

## Context

- [Choose where the retrospective skill is installed](../decisions/02-retrospective-distribution.md)
- [Domain glossary](../../../../CONTEXT.md)
