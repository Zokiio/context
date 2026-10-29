# Agent harnesses and repair continuity

Status: plan agreed with Joakim on 25 October 2026. No ticket is in Current commitments yet; selecting them is a separate decision.

The factory runs Claude Code and Codex through `ctx factory agent`. This plan adds GitHub Copilot, checks every adapter against a live provider, lets a project choose which harness refines, writes tests and judges, and tells a repair why the previous attempt failed. Native session resume stays behind an open decision until measured repairs show it is needed.

## What exists on `dev` (`afe9989`)

- `internal/factoryworker/agent.go` runs Claude in print mode with `acceptEdits`, `--no-session-persistence`, an optional `--max-budget-usd` and `--allowedTools`. It runs Codex as `codex exec --json --sandbox workspace-write --cd <workspace>`. Any other harness is refused.
- Claude builder, judge and test-author runs happened live on 24 October 2026, as recorded in the [judge ticket](../records/factory-task-workspace/issues/03-judge-disputes.md). `docs/factory-platform.md` still says Claude was exercised once. Codex has only argument-construction tests.
- The Codex adapter reads `thread_id`, tool items, the final message and failure events. It records no model, no usage and no cost, so metrics count Codex attempts as $0. Codex has no spending cap.
- A repair is a new attempt in a new worktree (`<root>/<job>/<attempt>/checkout`) that starts from the failed candidate. `assignmentFor` in `internal/factory/workers.go` gives it the same instructions as the first attempt. The failed check, its log, a protected-path rejection and a person's "changes requested" summary never reach it. Only a resolved judge decision does.
- The refiner, expectation author and judge are Claude-only adapters. The service runs one installation-wide command per role (`--refiner-command-file`, `--expectation-command-file`, `--judge-command-file`). The builder already varies per job: a coordinator picks one of the project's allowed executor profiles, and the board and autopilot take the first.
- Claude records a session ID, turns and cost per run. Role runs record model, session, turns and cost (`RefinerRun`), but not the harness or profile.

## Decisions

Joakim made these choices on 25 October 2026.

| Question | Decision |
| --- | --- |
| Live Codex validation | An opt-in live adapter test, skipped by default and never run in CI, followed by one real Waymark ticket built by Codex in solo mode |
| ACP | Not planned. GitHub Copilot is the required new harness, and its programmatic CLI covers it. The permission rule below binds any future ACP client |
| GitHub Copilot route | The programmatic CLI, driven like Claude and Codex. Not ACP, the Agent Host Protocol or the Copilot Go SDK |
| Repair continuity | A failure brief first. Native resume waits on an [open decision](../records/factory-harnesses/decisions/01-resume-native-sessions.md) |
| Harness per stage | Per-project role profiles for the refiner, test author and judge. Each run records its profile, harness and model. No enforced independence rule |

### Why the Copilot CLI and not ACP, AHP or the SDK

Copilot CLI 1.0.96 already provides everything the adapter needs:

- a programmatic mode, with the prompt on stdin and `--output-format json` events;
- `--session-id <uuid>`, `--usage-output-file` and `--max-ai-credits`;
- `--allow-tool` and `--deny-tool` patterns, where deny always wins;
- file access limited to the working directory by default, and `--sandbox` to confine shell commands;
- `--no-auto-update`, `--no-ask-user` and `--disable-builtin-mcps`.

The [Copilot orchestration reference](../../.agents/skills/orchestrate/COPILOT.md) already records that the Agent Host Protocol connects clients to a host and is not a worker protocol. ACP (`copilot --acp`) would let Waymark answer each permission request. The CLI's deny rules give the same default without a protocol client to maintain. The Go SDK adds a large dependency that changes daily. Its quick-start also approves every permission request, the same default as the pinned Petri ACP client.

### Why a brief before resume

A repair today knows less than the person watching it. Every harness benefits from the brief, deterministic fixtures can test it, and it keeps the dispatch a complete record of what the builder was given. Resume is harness-specific. It depends on provider-local transcripts that the controller does not own, and it adds compatibility risks:

- `codex exec resume` 0.163.0 accepts neither `--sandbox` nor `--cd`, so a naive resume can drop the sandbox or work in the previous attempt's worktree, which still exists.
- Claude runs with `--no-session-persistence` and keeps transcripts under a directory named after the working directory. Whether `--resume` finds a session from another worktree is unverified.

Build resume only if briefed repairs still fail for lost context.

## Harness contract

Every adapter, for every role, follows these rules. The service still enforces its own guarantees: protected paths, clean-worktree verification, and the two human decisions.

- **Prompt.** The adapter passes the role's rendered prompt on stdin, with the rules first and the project's instructions after them.
- **No commits.** The agent never commits. The adapter commits what the builder left, or the service keeps what the test author left.
- **Identity.** Each run records the harness, the CLI version the adapter observed, the model, the native session ID and the profile key. A session ID that Waymark generated (`--session-id`) is preferred to one the agent reports.
- **Usage.** Each run records usage in the harness's own units: USD for Claude, tokens for Codex, AI credits for Copilot. A run whose harness reports no dollars has unknown cost, never $0.
- **Spending cap.** A cap applies where the harness has one (`--max-budget-usd`, `--max-ai-credits`). Codex has none, so only the deadline bounds a Codex run.
- **Fresh sessions.** Every run starts a fresh native session until the resume decision says otherwise.
- **Fail closed.** An unparseable result, a reported error or a run that changes nothing fails the run. A judge's decision must be exactly one JSON object that matches its schema.

### Permission rule

This rule binds every adapter and any future ACP or SDK client.

- Deny by default. Allow only file edits inside the assigned workspace and the shell commands the profile lists. Read-only roles get no edit and no shell tool.
- Never select `allow_always`. Never pass `--allow-all`, `--allow-all-tools`, `--allow-all-paths`, `--yolo`, `--dangerously-skip-permissions`, `bypassPermissions` or `--dangerously-bypass-approvals-and-sandbox`.
- Strip auto-approval variables from the agent's environment. `COPILOT_ALLOW_ALL=true` also trusts the directory's hooks.
- A protocol client answers each permission request itself and records the decision as an attempt event. It rejects a write to a protected path when the request names the path.
- A denied permission is an unavailable operation. The adapter never retries with broader permissions.

Permission prompts cover only the tool calls an agent asks about. They are not confinement. Confinement belongs to the sandbox plan.

### Per harness

| | Claude Code | Codex | GitHub Copilot |
| --- | --- | --- | --- |
| Builder | `-p`, `stream-json`, `acceptEdits`, `--allowedTools` | `exec --json --sandbox workspace-write --cd` | `--output-format json`, `-C`, `--allow-tool write` plus the profile's shell patterns, `--sandbox` |
| Read-only roles | `--tools Read Grep Glob`, `--json-schema` | `exec --sandbox read-only --output-schema <file>` | `--available-tools` limited to read and search tools; final message parsed strictly |
| Spending cap | `--max-budget-usd` | none | `--max-ai-credits` |
| Usage | `total_cost_usd` | token counts from `turn.completed` | `--usage-output-file` |
| Version pin | observed `--version` | observed `--version` | `--no-auto-update` and observed `--version` |
| Resume, if later decided | `--resume`, with `--session-id` set by Waymark | `exec resume <id>` with `-c sandbox_mode=...` | `--resume` or `--session-id` with `-C` |

The live check confirms each cell before an adapter relies on it. Copilot tool names and its JSON event shapes are unverified until the first live transcript.

## Live validation

An opt-in Go test runs the real `ctx factory agent` for each named, installed harness against a disposable Git repository with a small Go task. `go test ./...` and CI skip it. Selecting harnesses through an environment variable runs it on Joakim's machine with his provider logins. Each harness runs four scenarios:

1. Success produces one candidate commit.
2. A run that changes nothing fails without a commit.
3. A write outside the workspace leaves no file behind.
4. A dispute reaches the dispute output without a commit.

The test logs the CLI version and what it observed. A sanitized JSONL transcript per harness is committed as test data, so parser tests catch format drift without a provider.

After the Codex adapter passes, one real Waymark ticket runs through `ctx factory solo` with the Codex builder. Joakim chooses the ticket and makes the approval decision. The ticket comments record the version, outcome per stage, repairs and usage. Each defect becomes its own ticket.

## Repair brief

A repair attempt's assignment carries a brief about the attempt it repairs, built only from retained records:

- each failed or unknown required check on the previous candidate, with its key, command, exit observation and a bounded tail of its `command-N.log` artifact;
- the protected paths the previous candidate changed;
- the previous builder's final message, from its `agent-result.json`;
- the person's summary when they requested changes.

The brief is bounded in size and labelled as observations, not requirements. The admitted requirements and context stay frozen and are supplied again in full. `AgentPrompt` renders the brief after the rules for every harness. The brief is retained with the attempt, so History shows what the repair was told. Agent-review findings, planned in the agent-review plan, add a section to this brief rather than a second channel.

Metrics gain a repair pass count: repair attempts whose candidate passed every required check. Retained attempts supply it, so existing history counts.

## Role profiles

A project administrator registers role profiles the way executor profiles are registered: a key, a role, a display name and an immutable argument array. The roles are `refiner`, `test_author` and `judge`. The agent-review plan can add `reviewer`. A profile never changes, and a different command needs a new key.

Project policy names one profile per role. A project that names none uses the installation's configured command, so current installations keep working. A refinement, test run or judgment keeps the profile it started with. Each records the profile key, harness and model, and Run, History and the Agents page show them. Solo registers built-in profiles for each installed harness that supports a role. Waymark does not enforce a different harness for the judge and the builder. The admin's choice of profiles provides that independence, and the recorded harness shows it.

Codex and Copilot role adapters follow the harness contract. The Codex refiner and judge run with `--sandbox read-only` and `--output-schema`. The Copilot refiner and judge run with read and search tools only. Copilot has no schema flag, so a Copilot judge's final message must be exactly one JSON object that passes the existing strict decision checks. Anything else is an error, and the dispute goes to a person, as it does today.

## Native resume, if decided later

The [resume decision](../records/factory-harnesses/decisions/01-resume-native-sessions.md) holds the [resume ticket](../records/factory-harnesses/issues/10-resume-repair-session.md). If it is built, a repair may resume only when every check passes; otherwise it starts fresh and records why:

- It is an implementation repair of the same job, resuming the session of that job's immediately preceding implementation attempt.
- That attempt ended with a known outcome, not `unknown` or an unacknowledged cancellation.
- The executor profile key, harness CLI version, model and worker identity match.
- The admitted requirement fingerprint, context digest, protected paths and builder instructions match.
- Waymark generated the session ID, and the transcript exists.
- The harness binds the resumed session to the new attempt's workspace and restates its confinement. Codex needs `-c sandbox_mode=workspace-write`, because `exec resume` has no `--sandbox` flag.

A resumed repair still receives the full brief and admitted context. Waymark deletes transcripts it persisted for resume when the job ends. Resume makes repair depend on a provider-local conversation outside the dispatch record. The vision asks for enough to resume work "without preserving the whole conversation", so building resume needs an ADR first.

## Not planned

- **An ACP client.** Revisit when a wanted harness has no usable headless mode. The permission rule above then applies.
- **The Copilot Go SDK and the Agent Host Protocol.** The CLI is enough.
- **OpenCode.** Its CLI has no sandbox of its own. Revisit after the sandbox plan lands.
- **A rule forcing the judge's harness to differ from the builder's.** Revisit if recorded runs show correlated misses.
- **Routing work to harnesses by a decision model.**

## Dependencies on sibling plans

- **Sandbox plan.** It owns confinement, egress, cancellation and budgets. Codex cannot be capped in dollars, so a budget reservation must treat Codex profiles as uncapped or refuse them. The refiner, test author and judge run inside the service process, and their isolation is also the sandbox plan's.
- **Agent review plan.** Its reviewer uses a role profile, and its findings join the repair brief.

## ADR

None now. The plan keeps the vision's adapter model and ADR 0005's policy-owned execution profiles. Role profiles extend that policy to more roles. Native resume would change the dispatch-record and conversation commitments, so its ticket requires an ADR before implementation.

## Tickets

1. [Tell a repair why the previous attempt failed](../records/factory-harnesses/issues/01-repair-brief.md)
2. [Count repairs that pass verification](../records/factory-harnesses/issues/02-repair-pass-count.md)
3. [Check the native adapters against live providers on demand](../records/factory-harnesses/issues/03-live-adapter-check.md)
4. [Build one real Waymark ticket with Codex](../records/factory-harnesses/issues/04-codex-real-run.md)
5. [Report agent usage without inventing a cost](../records/factory-harnesses/issues/05-usage-without-invented-cost.md)
6. [Build with GitHub Copilot](../records/factory-harnesses/issues/06-copilot-builder.md)
7. [Choose the refiner, test author and judge per project](../records/factory-harnesses/issues/07-role-profiles.md)
8. [Refine, write tests and judge with Codex](../records/factory-harnesses/issues/08-codex-roles.md)
9. [Refine, write tests and judge with Copilot](../records/factory-harnesses/issues/09-copilot-roles.md)
10. [Resume the builder's session on repair](../records/factory-harnesses/issues/10-resume-repair-session.md), blocked by the resume decision

Tickets 01, 02, 03 and 07 have no blockers and can run in parallel. 04 and 05 follow 03. 06 follows 05. 08 follows 05 and 07, and 09 follows 06 and 07.
