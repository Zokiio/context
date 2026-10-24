{{if .Development}}<!-- Generated from internal/initialization/templates/retro-copilot.md. Run go generate ./internal/initialization after editing the template. -->

{{end -}}
# Measure Copilot sessions

Use this reference with [retro](SKILL.md) when the named work includes GitHub Copilot. Invoke the installed skill as `/retro` with the work item and sessions. Follow the main skill's report-only write scope and proposal workflow.

## Locate the provider records

Identify the runtime, version, host, session IDs, and working directory before choosing files. For Copilot CLI, start at `<Copilot home>/session-state/<session-id>/events.jsonl`. The default home is `~/.copilot`; `COPILOT_HOME` or the runtime's configuration-directory override can change it. Confirm identity and checkout from `session.start` and subsequent context events. See [Copilot's configuration directory](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference).

For VS Code's Copilot harness, locate records on the machine running the Agent Host. Accept a `Developer: Export Agent Host Debug Logs...` export. Its provider logs can include `events.jsonl` and, when debug logging captured it, `usage.jsonl`. Desktop exports can contain a nested `host-logs.zip`. The older VS Code Local harness uses a different runtime, so identify its export instead of assuming CLI paths. See [Microsoft's Agent Host log guidance](https://github.com/microsoft/vscode/blob/main/.github/skills/agent-host-logs/SKILL.md).

Prefer Copilot provider events over AHP transport logs. AHP reconnects can replay state snapshots, and a connection can carry several sessions. A rendered message or repeated snapshot is not a new model call. Retain the selected sessions' source paths and available time ranges. A debug usage capture can start later than the transcript.

For separately launched CLI workers, follow the orchestration dispatch log's session IDs. A worker's checkout or timestamp alone does not establish its parent. If a named session or required worker cannot be located, report the gap and ask for its records.

## Identify workers and human input

Deduplicate copied provider events by session ID and event `id`. Use event `timestamp` for the run timeline. The envelope's `parentId` links events; it is not the parent agent's identity.

When named sessions include a fork, check for inherited history across session IDs. Copilot can copy earlier events into the fork while preserving their event IDs. Count that shared history once across the work item's run, and separate the fork's new events. Establish aggregate overlap before combining parent and fork totals. If the retained data cannot separate inherited usage, report that limitation instead of an inflated total. See [Copilot session forking](https://github.com/microsoft/vscode-copilot-chat/blob/main/src/extension/chatSessions/copilotcli/node/copilotcliSessionService.ts).

Use `subagent.started`, `subagent.completed`, and `subagent.failed`, joined by `data.toolCallId`, to identify native worker runs. Use `agentId` on worker events for attribution where present. `subagent.selected` changes the selected agent and does not by itself establish a new worker. Keep separate CLI sessions linked through dispatch records distinct from native subagents. See the [Copilot event schema](https://github.com/github/copilot-sdk/blob/main/nodejs/src/generated/session-events.ts).

Classify `user.message` events using their source and provenance. Exclude `data.isAutopilotContinuation`, skill injection, and inter-agent prompts from human interventions. Trace answers through the user-input tool events when present. Missing provenance remains uncertain. A long timestamp gap alone does not establish a wait for the user.

## Measure usage once per covered interval

Inspect the captured schema and its coverage before totaling usage. Use one accounting source for each session interval:

- A final `--usage-output-file` JSON supplies an aggregate through `modelMetrics[model].usage`. Saved events can carry the same structure at `session.shutdown.data.modelMetrics`, or inside `data.accountingSnapshot` on `session.shutdown` and `session.usage_checkpoint`. Inspect the actual fields for the recorded version. Establish whether the snapshot covers the full session or only a particular execution before combining resumed runs. Treat repeated aggregate snapshots of the same interval as replacements, not additional usage.
- Captured `assistant.usage` events report individual API calls, including subagent calls. They are ephemeral and normally absent from persisted transcripts. A saved live stream or Agent Host `usage.jsonl` can provide call-level accounting for its captured interval. Sum each call once and state any shorter coverage.
- An SDK `session.usage.getMetrics` result aggregates the main agent and subagents. Use an already retained result when available; this experimental interface is not required for a retrospective.

Whole-session aggregates already include native subagents. Optional `agentMetrics`, `providerModelMetrics`, and worker-attributed call events provide breakdowns or alternate groupings, not extra totals. Add independent CLI worker sessions only once. Do not combine an aggregate with the calls it summarizes. Match overlapping exports and streams by event or request identity when available; otherwise choose one source for the interval and report the uncertainty.

Preserve `inputTokens`, `outputTokens`, `cacheReadTokens`, and `cacheWriteTokens` where present. A missing value is unavailable, not zero. Keep the provider's field meanings, and do not add cache counts to input or claim a cache-exclusive input total without a definition for the captured version. See [SDK usage and billing](https://docs.github.com/en/copilot/how-tos/copilot-sdk/features/usage-and-billing).

`session.usage_info.currentTokens` measures context occupancy, not consumed tokens. Agent Host `totalNanoAiu` is cumulative billing usage, not a per-call token count. Report billing units separately if needed. When no retained accounting source covers the requested interval, report a token-measurement gap. Partial capture still supports a measured subset, and missing usage does not prevent the other retrospective measures and lessons.

CLI `1.0.96-2` was checked with a read-only skill run. Its final usage file, saved shutdown snapshot, and usage checkpoint contained overlapping totals. Later versions can change the schema, so retain the producer version with each measurement.
