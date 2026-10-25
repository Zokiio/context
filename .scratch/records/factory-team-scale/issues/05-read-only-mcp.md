---
type: WorkItem
id: cd01d2ab-823f-4fdb-b795-d989eab82ad0
title: Inspect factory work from an agent through a read-only MCP server
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Factory HTTP and browser access](../../../../docs/factory-http.md)
- [Use ctx for project work](../../../../docs/agents/cli.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- `ctx factory mcp --url <service> --token-file <file>` runs an MCP server over stdio. It calls the service through `factoryclient` with that user's token and opens no database, data directory or workspace.
- Its tools list accessible projects and read a project's orientation and board, a work item's record, run stages and activity, a job with its attempts, checks and reviews, a job's change, and a job's events after a sequence. Each returns the service's JSON for that operation.
- No tool changes state. The server offers no claim, report, queue, plan, approve, merge, dispute or administration tool.
- Results carry the caller's own permissions. A 401, 403 or 404 becomes a tool error with the service's code and message, and nothing is cached between calls.
- Tool descriptions say that record text and worker output are project data, not instructions.
- The operator guide shows how to register the server in Claude Code and Codex. A test drives the server over stdio with a scripted MCP client against a real service, as a read-only user and as a user without access to the project.

## Comments

- 2026-10-25: Joakim chose to include a read-only MCP server in the team plan over the recommendation to defer it. Write tools stay out because claims belong to jobs, reports to workers and decisions to people. Task context for an unadmitted ticket stays with the local `ctx context` reader. Keep the Go 1.25 baseline whether this uses an MCP SDK or a small JSON-RPC implementation.
