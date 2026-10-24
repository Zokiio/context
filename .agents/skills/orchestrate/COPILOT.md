<!-- Generated from internal/initialization/templates/orchestrate-copilot.md. Run go generate ./internal/initialization after editing the template. -->

# Run checkpoints with Copilot

Use this reference with [orchestrate](SKILL.md) in GitHub Copilot CLI or the Copilot harness in an Agent Host. Invoke the installed skill as `/orchestrate` with the work item. Keep the shared workflow's scoped dispatches, single writer, independent verification, and human gates.

## Identify the runtime

Record the Copilot version, host, checkout, and available delegation and session tools. Run `copilot --version` and inspect `copilot --help` on the host that will run CLI workers. An Agent Host can run remotely, so its filesystem and installed CLI can differ from the editor machine's.

The Copilot harness uses the Copilot SDK and its runtime process. Agent Host Protocol connects clients to that host. AHP does not define worker context isolation, and `copilot --acp` serves the separate Agent Client Protocol. Use the environment's exposed tools or CLI instead of implementing either protocol for this skill. See [Agent Host architecture](https://code.visualstudio.com/docs/agents/concepts/agent-host).

## Dispatch fresh workers

Use native delegation when it provides a fresh context without the parent conversation. Inspect the tool's actual schema. Copilot's `task` tool and the older VS Code Local harness's `runSubagent` have different interfaces. Pass the retained dispatch by pointer, including the checkpoint and verification instructions. Custom agents do not all inherit project instructions or skills, so name required guidance explicitly. See [subagents](https://code.visualstudio.com/docs/agents/run/subagents).

When that isolation is unavailable or uncertain, start a new CLI session for each worker. Generate an unused UUID for `--session-id`. Reusing an existing ID resumes its history. Leave `--resume`, `--continue`, and `--enable-memory` out of fresh worker invocations. Keep the assigned role in the dispatch. Automatic fleet decomposition does not preserve this workflow's assignment boundaries.

Before launch, retain the exact dispatch and its digest as the main skill requires. Record the new session ID, role, checkout, CLI version, starting revision, and output paths in the dispatch log. Put output files in the project's ignored evidence directory. Use distinct absolute paths for each worker.

For a CLI worker, set the following variables to the selected checkout, fresh UUID, retained dispatch file, and output files. The dispatch file contains the scoped pointer prompt. Add only the tool permissions already authorized for that assignment:

```sh
copilot -C "$ctx_copilot_checkout" \
	--session-id "$ctx_copilot_worker_id" \
	--output-format json \
	--usage-output-file "$ctx_copilot_usage" \
	< "$ctx_copilot_dispatch" \
	> "$ctx_copilot_events" 2> "$ctx_copilot_stderr"
```

Piped stdin selects programmatic mode. Alternatively, pass the literal dispatch text as one `-p` argument through an argument-array process API. Avoid combining these forms, because `-p` overrides piped input. Preserve the process exit status and inspect the worker's evidence and final revision before treating its assignment as complete. See [programmatic CLI use](https://docs.github.com/en/copilot/how-tos/copilot-cli/automate-copilot-cli/run-cli-programmatically).

Use supported, scoped `--allow-tool` permissions for the assignment. A permission failure is an unavailable operation, not a reason to retry with `--allow-all` or `--yolo`. When the installed CLI cannot run within the existing permissions, use native delegation or an interactive session with the same dispatch. Keep required questions visible to the orchestrator.

Retain the JSONL output and final usage JSON when the installed version supports them. If usage capture is unavailable, retain the session ID and transcript location and record the measurement gap. A worker launched as a separate CLI session has no automatic parent relationship. The dispatch log supplies that relationship for a later retrospective.

## Continue in a new session

At a verified checkpoint boundary, save the next step and the records named by the main skill. Use an exposed session-creation tool that starts a new conversation, or give the user a command like this with the selected checkout and concrete work-item pointer:

```sh
copilot -C "$ctx_copilot_checkout" \
	-i '/orchestrate Continue the work item identified in /absolute/path/to/progress.md from its saved records.'
```

Use a new session ID if supplying one. Resuming, reconnecting to an AHP session, or compacting its conversation does not create the fresh checkpoint context this workflow requires. Transfer orchestration ownership to the successor and stop dispatching from the previous session. Keep the handoff prompt and successor identity in the progress records when available.
