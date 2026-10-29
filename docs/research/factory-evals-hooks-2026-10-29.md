# Software factory evals and hooks research for Waymark, 29 October 2026

Research date: 29 October 2026, Europe/Stockholm. The starting point was the AI Hero cohort page [Software Factories for Real Engineers](https://www.aihero.dev/cohorts/software-factories-for-real-engineers-xdy1m), Matt Pocock's two-week course starting 9 November 2026. The page has a syllabus but no materials, so this note reads the public code behind it, then the primary sources on hooks and evals, and compares both with Waymark's factory. It is source research. Nothing was installed or run, and no agent, hook or eval was executed. [Pinned source records](factory-evals-hooks-2026-10-29.sources.json)

## Verdict

**The course is Matt Pocock's factory, and its public code shows what that is.** Agent sessions start from GitHub labels, a cron job and closed issues. Each runs Claude Code through his `sandcastle` library. A reviewer agent fixes what it finds, and Matt merges by hand. The brakes are deterministic: typecheck, lint, a complexity ratchet and 20 custom check scripts. Waymark's factory already goes further on verification: holdout tests the builder can't change, a separate verifier on a clean worktree, a dispute judge, and two human decisions. There is nothing to adopt from the course's factory. Its retro rule, that a mechanical mistake gets a check and not a sentence, is already in Waymark's `retro` skill.

**Hooks: fix what Waymark already runs before adding any hook of its own.** Every Claude Code role in the factory starts `claude -p` without an isolating flag. Claude Code's own docs say that this runs the target repository's hooks, `env` block, project skills' `allowed-tools` and `.mcp.json` servers, and the operator's hooks, with the operator's permissions. The refiner and judge, which the isolation plan treats as having no shell, do this in the main checkout. That is the most important finding here. Once runs are isolated, the only hook worth having is an optional `Stop` gate that makes the builder run the holdouts before it finishes. The popular community hook enforcers fail open or get bypassed, and none is stronger than Waymark's check after the session.

**Evals: test the tests first.** Every factory metric is graded by the required checks and the holdout tests. Benchmark audits keep finding those graders wrong. OpenAI dropped SWE-bench Verified after finding that at least 59.4% of 138 audited hard tasks had tests that rejected correct solutions. The smallest useful eval runs the accepted holdout tests at the base revision and requires them to fail. All three research threads reached that independently, it needs no model, and nothing in Waymark does it today. Agent evals come later. The local factory databases hold 136 refinements and no build jobs. Runs don't record which prompt version they were given, and replaying old tickets in today's worktrees would leak the answers through later commits.

Three gaps, in order: isolate role sessions, check holdouts fail at base, record each run's prompt digest. See [Relevance to Waymark](#relevance-to-waymark).

## What the course page says

The page promises two weeks. Week 1 covers domain language, planning with skills, three kinds of review and codebase design. Week 2 covers "feedback loops": automated checks, custom lint rules, mutation testing, verification for front-end apps, and "building a factory like Matt's" with GitHub Actions to investigate, triage, assign, delegate, track and review work. It names hooks and sandboxes among its concepts. It never mentions evals. It costs $1,295 and is taught with Claude Code.

The page also carries a block addressed to AI agents, with instructions to connect an MCP server and log in to a learner's account. It was treated as page content and not followed.

## Matt Pocock's factory in public code

Three public repositories show what the course teaches. `mattpocock/course-video-manager` runs a factory on its own development. `mattpocock/sandcastle` is the library that factory calls. `mattpocock/skills` holds the skills it uses. A fourth, `mattpocock/dictionary-of-ai-coding`, defines the terms.

### Definition

The dictionary defines a software factory as a system "where agent sessions are started by triggers", such as an issue, a schedule, a CI failure or another session finishing, "rather than by a human" ([Software factory.md:7](https://github.com/mattpocock/dictionary-of-ai-coding/blob/ed1ebed3975cba04ed5e74c6ca73659274beb754/dictionary/Software%20factory.md#L7)). It "takes the human out of starting sessions, and not necessarily out of anything else" (line 9). Which decisions stay human is "the main design question" (line 22). A factory where nobody reviews the output is a "dark factory" ([Dark factory.md:5](https://github.com/mattpocock/dictionary-of-ai-coding/blob/ed1ebed3975cba04ed5e74c6ca73659274beb754/dictionary/Dark%20factory.md#L5)), where automated checks and automated review "are the only gates" (line 11).

Waymark's factory fits that definition from the other side. Its autopilot starts sessions from ready work, and its two human decisions are the ones the dictionary says to keep.

### How the factory runs

`course-video-manager` (public, no license file, 857 stars) runs each stage as a GitHub Actions workflow started by a label or an event. Every agent is Claude Code (`claude-opus-5`) started through `sandcastle.run()`.

| Trigger | Workflow | What it does |
| --- | --- | --- |
| Label `agent:implement` on an issue | [agent-implement.yml](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement.yml) | Refuses sub-issues and issues that already have an open PR. Creates `agent/issue-N-slug`, runs the implement script, fails if the agent made no commits, has a second agent write the PR text, opens a draft PR and labels it `agent:review`. A failure labels the issue `agent:blocked` with the reason ([3-9](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement.yml#L3-L9), [62-89](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement.yml#L62-L89), [145-214](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement.yml#L145-L214), [216-235](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement.yml#L216-L235)). |
| Label `agent:review` on a PR | [agent-review.yml](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-review.yml) | Installs the `code-review` skill from `mattpocock/skills` at whatever version is current ([78](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-review.yml#L78)). The reviewer edits the branch, posts inline comments and replies to human threads, then marks the PR ready for review ([125](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-review.yml#L125)). |
| Label `agent:implement` on a PR | [agent-implement-pr.yml](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement-pr.yml) | Addresses the unresolved review conversation on the PR branch. It uses `pull_request_target` and checks out the PR head with the agent token ([6](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement-pr.yml#L6), [76-78](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-implement-pr.yml#L76-L78)), which is safe only because labelling needs write access. |
| Label on an issue with sub-issues | `agent-implement-prd.yml` | Implements a PRD's sub-issues one after another on one branch. |
| Issue closed | [agent-promote-queued.yml](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-promote-queued.yml) | Moves dependents from `agent:queued` to `agent:implement` once their last blocker closes. People apply `agent:queued` ([1-12](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/agent-promote-queued.yml#L1-L12)). |
| Weekdays 09:00 UTC | [architecture-review.yml](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/architecture-review.yml) | An agent files architecture issues, skipped while 10 or more of its issues are open ([4-6](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/architecture-review.yml#L4-L6), [24-35](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.github/workflows/architecture-review.yml#L24-L35)). |

Nothing merges on its own. The review stage ends by marking the PR ready, and Matt merges. The checks that gate a merge are deterministic. `pnpm run check:static` chains typecheck, lint, boundary lint, a complexity ratchet and 20 custom check scripts for repository rules, such as banned imports and file locations ([package.json:59](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/package.json#L59)).

Three details matter for Waymark:

- **The reviewer is also a repairer.** It runs the `code-review` skill (Standards and Spec axes in parallel sub-agents), then fixes what it finds on the branch and commits ([review/prompt.md:43-63](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.sandcastle/review/prompt.md#L43-L63)). It sends spec gaps to the human rather than coding them. Waymark's planned agent reviewer only reports, and its blocking findings go to a bounded repair by the builder.
- **No sandbox in CI.** Every agent call in the CI scripts passes `noSandbox()`, nine call sites in all ([implement.ts:12-27](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.sandcastle/implement/implement.ts#L12-L27)). The Actions runner is the only isolation, and the agent holds a token that can push. Docker is used only by the local planner loop in `main.ts`, which caps itself at 10 iterations and 4 parallel agents ([main.ts:9-29](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.sandcastle/main.ts#L9-L29)).
- **Retries are for output format, not for work.** `runWithRetry` resumes the same session up to three times when the structured output fails schema validation, and feeds back the exact error ([run-with-retry.ts:24-48](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.sandcastle/run-with-retry.ts#L24-L48)). A failed implementation isn't retried. The issue is labelled blocked and waits for a person. Waymark's roles already ask Claude Code for schema-checked output with `--json-schema`, so this wrapper adds nothing.

How much it runs: 175 PRs from `agent/` branches merged between May and September 2026, peaking at 78 in July and falling to 10 in September. None merged in October, while Matt merged 297 PRs from other branches that month. The repository doesn't say what produced those, so this note doesn't guess.

`sandcastle` itself (MIT, version 0.12.0, 8.7k stars) is a TypeScript library: `run()` starts an agent in a sandbox (Docker, Podman, Vercel or none), on a branch strategy, and merges the commits back ([README.md:10-18](https://github.com/mattpocock/sandcastle/blob/a53e227250fb51df0ad54155b85fecc1c090c937/README.md#L10-L18)). It ships templates for a simple loop, a sequential reviewer and a parallel planner. Waymark is Go and already allocates its own worktrees, so it is a reference for the isolation plan at most.

### Where hooks and evals sit in his material

**Hooks are a small tool for him.** The factory's only hook is a `PreToolUse` command on Bash that blocks `npx tsc` and tells the agent to use `pnpm run typecheck` ([settings.json](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.claude/settings.json), [block-npx-tsc.sh:6-13](https://github.com/mattpocock/course-video-manager/blob/ca63ea73c9ac4435ed107a7bc32dc51994ac9b43/.claude/hooks/block-npx-tsc.sh#L6-L13)). It steers a command; it guards nothing. His `git-guardrails-claude-code` skill installs a hook that blocks `git push`, `reset --hard` and similar ([SKILL.md:8-18](https://github.com/mattpocock/skills/blob/a496f5513d4a4cf889bc48e189e579f4b0bdf6aa/skills/misc/git-guardrails-claude-code/SKILL.md#L8-L18)), but it sits in `skills/misc`, which he marks frozen and doesn't run day to day ([frozen-misc-skills.md:3-9](https://github.com/mattpocock/skills/blob/a496f5513d4a4cf889bc48e189e579f4b0bdf6aa/.out-of-scope/frozen-misc-skills.md#L3-L9)).

**His "evals" are retros that turn mistakes into checks.** The `retro` skill reads a finished session and routes each mistake ([retro.md:15-29](https://github.com/mattpocock/skills/blob/a496f5513d4a4cf889bc48e189e579f4b0bdf6aa/docs/engineering/retro.md#L15-L29)). A mistake a tool could catch becomes an automated check: a lint rule, type, test, pre-commit hook or CI job. A judgement call becomes a rule in `CODING_STANDARDS.md` for the reviewer, never for the implementer, which has the least context to spare. A mechanical violation always gets a check, "because a check can fail and a sentence in a standards file can't". It only proposes, and it installs no hook automatically, because a user was "burned by auto-hooks that blocked good changes" ([retro.md:35](https://github.com/mattpocock/skills/blob/a496f5513d4a4cf889bc48e189e579f4b0bdf6aa/docs/engineering/retro.md#L35)). For agent instructions he says plainly that "There is no automated eval" ([writing-for-agents.md:47](https://github.com/mattpocock/skills/blob/a496f5513d4a4cf889bc48e189e579f4b0bdf6aa/docs/productivity/writing-for-agents.md#L47)). His `evalite` library evaluates LLM-powered apps, not coding agents; see [Evals](#evals).

Waymark adapted `retro`, `code-review`, `tdd` and `implement-spec` from this repository on 4 October 2026 ([workflow adaptation spec](https://github.com/Zokiio/context/blob/44e137962ca364bdf215f381509cbb7084b56921/.scratch/workflow-adaptation/spec.md)). The course's feedback-loop week is the part Waymark has least of. Waymark's factory runs whatever required checks a project names. It has no routine that turns an observed agent mistake into a new check.

## Hooks

### What hooks are

A hook is a command, HTTP call or prompt that the agent CLI runs at a lifecycle event. Claude Code has about 30 events, among them `SessionStart`, `PreToolUse`, `PostToolUse`, `Stop` and `SubagentStop` ([hooks reference](https://code.claude.com/docs/en/hooks); docs carry no version, CHANGELOG at 2.1.312). A hook gets the event as JSON on standard input. Exit code 2 blocks, with standard error as the reason. A `PreToolUse` hook can also return `allow`, `deny`, `ask` or `defer`, and a `Stop` hook that blocks keeps the agent working. Codex ([hooks](https://learn.chatgpt.com/docs/hooks), `rust-v0.166.0`) and the Copilot CLI ([hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference)) have the same shape. They differ on trust. Codex runs a non-managed hook only after someone has trusted its exact hash, and the Copilot CLI has a `--trust-folder` flag to trust a working directory's hooks for one non-interactive session ([programmatic reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-programmatic-reference)). Claude Code's `-p` has no such step.

Three limits matter for a factory:

- **A hook is a guardrail, not a boundary.** On `PreToolUse`, a hook that times out lets the call through unless it sets `onFailure: "block"` (2.1.295 and later). Claude Code's docs send hard allow and deny to permission rules instead. Codex's docs say to "Treat tool hooks as a useful guardrail, not a complete enforcement boundary."
- **Hooks don't see subprocesses.** A hook or a `--disallowedTools "Edit(<pattern>)"` rule sees the agent's own file tools and recognized shell commands. It doesn't see what `go test` or a build script writes. Only an operating-system sandbox covers those.
- **Hooks run with the user's full permissions, outside the agent's sandbox.**

### Waymark already runs other people's hooks

This is the finding that matters most, and it is about risk, not features.

Every Claude Code role in Waymark's factory starts `claude -p` with no isolating flag: the builder ([agent.go:172](../../internal/factoryworker/agent.go#L172)), the expectation author, the refiner and the judge. Claude Code never shows a trust dialog in `-p`, and its [permissions page](https://code.claude.com/docs/en/permissions#what-runs-before-you-trust-a-folder) lists what a repository still gets in that case:

- hooks in its settings files, its `env` block and helpers such as `apiKeyHelper`, and its project skills' hooks and `allowed-tools`: "Used";
- servers in its `.mcp.json`: "Connected without asking, approved or not".

Only the repository's `permissions.allow` rules are held back. The same page tells anyone running `claude -p` in a repository they didn't write to pass `--setting-sources user`, `--bare` or `--settings '{"disableAllHooks": true}'` first. Waymark passes none of them. What follows:

- **The refiner and judge run in the project's main checkout** ([refine.go:314](../../internal/factory/refine.go#L314), [dispute_judge.go:199](../../internal/factory/dispute_judge.go#L199)), with `--tools Read Grep Glob`. That flag "doesn't affect MCP tools" ([CLI reference](https://code.claude.com/docs/en/cli-reference)), and hooks are shell commands outside any tool list. The isolation plan leaves both roles unsandboxed because they have "no shell, no edit tool and no network tool" ([isolation spec](https://github.com/Zokiio/context/blob/44e137962ca364bdf215f381509cbb7084b56921/.scratch/factory-isolation/spec.md#L50)). A repository's hooks and `.mcp.json` break that premise.
- **A repository can widen the builder's permissions.** A `PreToolUse` hook that returns `allow` lets a call through without a prompt. Only deny and ask rules override it ([permissions](https://code.claude.com/docs/en/permissions#precedence-among-rules-hooks-and-mods)). Waymark passes an allow list and no deny rules.
- **A candidate can plant a hook for the next agent.** A builder that commits a `.claude/settings.json` hands it to anything that later runs Claude Code in that candidate: a repair, which starts from the failed candidate, and the planned agent reviewer, which works in a worktree at the candidate. That hook could also rewrite what the reviewer reads through `PostToolUse`. This follows from the documented behavior but hasn't been tested.
- **The operator's own hooks run too.** User-level hooks in `~/.claude/settings.json` apply to every factory session. For Codex, Waymark's `--sandbox workspace-write` overrides the user's sandbox mode, but a user `allow` rule still lets a matching command "Run the command outside the sandbox without prompting" ([Codex rules](https://learn.chatgpt.com/docs/agent-configuration/rules)). Codex has `--ignore-user-config` and `--ignore-rules` for this ([cli.rs:44-50](https://github.com/openai/codex/blob/6ff1e1ca3b30fc8b914720681fcf5fe62e95f856/codex-rs/exec/src/cli.rs#L44-L50)).

The flags that isolate a run, per the [CLI reference](https://code.claude.com/docs/en/cli-reference):

| Flag | Effect | Cost |
| --- | --- | --- |
| `--safe-mode` | Loads no CLAUDE.md, skills, plugins, hooks, MCP servers or auto memory. Login, models, built-in tools and permissions work normally. Managed policy still applies. | Also drops the project's CLAUDE.md, which Waymark's roles may rely on today. |
| `--bare` | Skips hooks, skills, plugins, MCP, auto memory and CLAUDE.md. | Needs an API key. The project's `env` block still applies. |
| `--restricted` | Loads only managed settings and `--settings`. Removes command-running tools unless named in `--tools`. Meant for "an evaluation harness" on a shared machine. | The builder needs Bash for its allowed commands. |
| `--setting-sources user` | Ignores the project's settings and `.mcp.json`. | User hooks still run. |
| `--settings '{"disableAllHooks": true}'` | Turns off user, project, local and plugin hooks for the run. | Doesn't stop `.mcp.json`. |

Two behaviors weren't documented and need a probe: whether `--safe-mode` keeps the project's `env` and `apiKeyHelper`, and whether `--restricted` loads CLAUDE.md or `.mcp.json`.

### What hooks could add, once runs are isolated

| Role | Hook option | Better tool |
| --- | --- | --- |
| Builder | Deny edits to protected paths at tool time instead of after the commit. | `--disallowedTools "Edit(<pattern>)"`, which is a permission rule rather than a hook. Neither stops a subprocess. |
| Builder | A `Stop` hook that runs the holdout tests and keeps the agent working while they fail. This is the one thing only a hook can do. Claude Code ends the turn anyway after 8 consecutive blocks by default (`CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`). | None. It stays advisory, because verification still decides acceptance. |
| Expectation author | A `PreToolUse` hook that denies writes outside protected paths. Permission rules can't say "everything except", because deny beats allow. | The service's own check stays authoritative. |
| Refiner, judge, reviewer | Nothing useful. Their tool lists already confine them. | Isolation flags plus `--strict-mcp-config` or `--disallowedTools "mcp__*"`. |
| Any role | Detect hooks Waymark didn't configure: `--include-hook-events` puts hook events in the stream the adapter already parses. | Fail the run when an unexpected hook fires. |

Logging tool calls through a hook adds nothing. Waymark already reads tool events from `stream-json` and `codex --json`.

Matt Pocock's hooks are all of the first kind, `PreToolUse` guardrails that exit 2. None of his public material uses a `Stop` gate or an isolation flag.

## Evals

### What the primary sources agree on

Anthropic's [Demystifying evals for AI agents](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents) (9 January 2026) is the most direct guide. Its points, all confirmed against the page:

- Start with 20 to 50 tasks drawn from real failures, not a large synthetic set.
- Grade what the agent produced, not the path it took. Read transcripts to learn why it failed.
- Prefer code graders. Model graders must be calibrated against human experts and allowed to answer "Unknown".
- Report reliability, not just success. pass@k is the chance that at least one of k tries succeeds; pass^k is the chance that all k succeed. At 75% per trial, pass^3 is about 42%.
- A 0% pass rate usually means a broken task. Anthropic's CORE-Bench score rose from 42% to 95% after it fixed rigid grading, ambiguous specs and irreproducible tasks.

Claude's docs on [building evaluations](https://platform.claude.com/docs/en/test-and-evaluate/develop-tests) rank code grading as fastest and most reliable, and say to grade with a different model from the one that produced the output. OpenAI's [evaluation best practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices) say to scale a model judge only once it consistently agrees with human labels. Anthropic's [statistical approach to model evals](https://www.anthropic.com/research/statistical-approach-to-model-evals) asks for standard errors and paired comparisons; at 20 tasks and a 50% rate the standard error is about ±0.11.

### The grader is usually the weak part

Coding benchmarks keep finding that their tests, not their agents, were the problem:

- OpenAI stopped reporting SWE-bench Verified on 23 February 2026. In an audit of 138 problems that o3 didn't solve consistently, at least 59.4% had tests that rejected functionally correct solutions ([OpenAI](https://openai.com/index/why-we-no-longer-evaluate-swe-bench-verified/); the page refused automated fetches, so this comes from its indexed text).
- In [SWE-Bench+](https://arxiv.org/abs/2410.06992), 32.67% of successful patches had the solution leaked in the issue, and 31.08% passed only because the tests were weak.
- Agents found fixes in the future history of the repository with `git log --all` ([SWE-bench issue #465](https://github.com/SWE-bench/SWE-bench/issues/465), 3 September 2025).
- StrongDM keeps its "scenarios" outside the codebase as a holdout, because tests inside it "can be reward hacked" ([StrongDM Software Factory](https://factory.strongdm.ai/)).

Mutation testing measures a suite's strength directly: seed small faults and count how many the tests catch. Google runs it on changed code only, during review ([Petrović et al.](https://arxiv.org/abs/2102.11378)). For Go, [Gremlins](https://github.com/go-gremlins/gremlins) v0.6.0 (Apache-2.0, December 2025) has a `--diff <ref>` mode and an efficacy threshold.

### Evalite

Matt Pocock's [evalite](https://github.com/mattpocock/evalite) (`0.19.0`, MIT) runs `.eval.ts` files on Vitest. Each eval has data, a task and scorers, results go to SQLite with a local UI, `trialCount` repeats cases, and a threshold sets the exit code. It evaluates LLM features in TypeScript apps. It needs Node and in-process tasks, and it averages scores, which hides pass^k. For Waymark it is an idea at most: data, task and scorer kept apart, variants compared on the same data, a fixed trial count and an exit code.

### What Waymark has

| Role | What an eval would measure | Data Waymark keeps | Today |
| --- | --- | --- | --- |
| Expectation author | Do the protected tests fail before the change and pass after, and how many seeded faults do they catch? | The expectation commit, patch and coverage claim | The coverage map is "a claim a person reads, not evidence" ([expectations.go:61-63](../../internal/factory/expectations.go#L61-L63)). Nothing checks fail-before or test strength. |
| Builder | First-pass acceptance, acceptance within the repair limit, consistency over repeated runs, protected-path rejections, cost | Jobs, attempts, checks, `agent-result.json`, events | Metrics exist. The repair pass count is planned. |
| Refiner | How often a person accepts, edits or rejects the proposal; wrong claims about the code | Refinements | Accept and reject counts. The decision model's scores are advisory and uncalibrated ([factory-platform.md:360](../factory-platform.md)). |
| Judge | Agreement with the person's resolution; never accepting weakened tests | Disputes | Nothing. |
| Agent reviewer | Misses on changes it cleared, false blocks | Reviews | Planned in the review metrics ticket. |

Three facts limit what can be measured now:

- **No build history.** The three local factory databases under `~/.waymark/` (the Opsbase and EGM trials) hold 136 refinements and no jobs or attempts.
- **No prompt version.** A run records harness, model, session, turns and cost ([agent.go:30-39](../../internal/factoryworker/agent.go#L30-L39), [refine.go:69-74](../../internal/factory/refine.go#L69-L74)) but not the version of the built-in rules it was given. Two prompt versions can't be told apart in the records.
- **Replay would leak.** A workspace is `git worktree add --detach` on the project repository ([workers.go:885](../../internal/factory/workers.go#L885)), so every later commit is reachable from inside it. Replaying a finished ticket there is the SWE-bench leak.

Waymark's "holdout" tests are protected from edits, not hidden: the builder can read them ([factory-platform-design.md:73-75](../factory-platform-design.md)). StrongDM's scenarios are hidden. Hidden tests stop a builder from writing code to fit them. Protected tests only stop it from editing them.

## What other projects do

Stars are from 29 October 2026; repositories were read at the commits in the ledger.

**Hook enforcers mostly fail open or get bypassed.**

- [tdd-guard](https://github.com/nizos/tdd-guard) (2.4k stars) has an LLM decide on each edit whether the agent is following test-driven order. Its own issues report agents getting around it through MCP write tools ([#48](https://github.com/nizos/tdd-guard/issues/48), open) and by stashing and resetting changes ([#41](https://github.com/nizos/tdd-guard/issues/41)), plus slow edits and loops. Its successor, Probity, adds deterministic rules and fails closed.
- [destructive_command_guard](https://github.com/Dicklesworthstone/destructive_command_guard) (6.2k stars) blocks dangerous commands across about 20 agents. 101 of its issues have "false negative" or "fail-open" in the title. It allows unparseable input unless told to fail closed.
- [claude-code-hooks-mastery](https://github.com/disler/claude-code-hooks-mastery) (4k stars) is a teaching repository whose blocking hook exits 0 on any exception. [obra/superpowers](https://github.com/obra/superpowers) (308k stars) uses only a `SessionStart` hook and enforces its test-driven workflow through skill text.
- Gemini CLI and Kiro have the same hook events, including a `Stop` equivalent. Across vendors, a hook that fails or returns something unexpected defaults to allowing the call.

None of these is a boundary. Waymark's check after the session, which rejects a candidate that touched a protected path, is stronger than any of them.

**Eval harnesses converge on Waymark's verification design.**

- [Harbor](https://github.com/harbor-framework/harbor), the Terminal-Bench harness (6.6k stars), made separate verifier sandboxes the default for new tasks in its "Cracking down on reward hacking" post of 28 October 2026. It also deletes stale reward files, rejects symlinks and other special files from the agent, and asks an LLM two questions per trial: did the agent game the verifier, and was the task clear enough? Its [issue #3741](https://github.com/harbor-framework/harbor/issues/3741) shows a separate verifier still reading a directory the agent could write.
- [Inspect](https://github.com/UKGovernmentBEIS/inspect_ai) (UK AI Security Institute) separates a score, an error and "unscored", so infrastructure failures don't count against the agent. It has `pass_at`, `pass_k` and `at_least` reducers over repeated runs. Its SWE-bench scorer restores the original test files before applying the hidden tests.
- [Aider's benchmark](https://github.com/Aider-AI/aider/blob/5dc9490bb35f9729ef2c95d00a19ccd30c26339c/benchmark/benchmark.py#L981-L1010) copies test files from a pristine copy before each run, overwriting the agent's edits, and reports pass rates after the first and second try. That is Waymark's first-pass acceptance with one repair, but more lenient: Aider overwrites tampering where Waymark records and rejects it.
- [promptfoo](https://github.com/promptfoo/promptfoo) (26.6k stars) says a workspace "isn't a sandbox". Its red-team guidance on verifier sabotage asks for hashes of protected files, a verifier outside the writable workspace that fails closed when its report is missing, and deterministic evidence before any LLM judgment.

**Factories that evaluate themselves don't publish how.** StrongDM measures "satisfaction" over scenarios kept outside the codebase but publishes no harness. Factory.ai's readiness levels don't say how criteria are scored. GitHub's Copilot coding agent runs tests, CodeQL and secret scanning on its own work before review ([changelog](https://github.blog/changelog/2026-03-18-configure-copilot-coding-agents-validation-tools/)). Cognition validates its evaluator agents by precision and recall against labelled outcomes ([post](https://cognition.com/blog/evaluating-coding-agents), 2024).

## Relevance to Waymark

The course's factory and the community projects add little to Waymark's design. Waymark already keeps the tests out of the builder's reach, verifies in a separate worktree, and has a person approve. What this research changes is three concrete gaps, in order of urgency.

**1. Isolate every role's agent session. This is a defect, not a feature.** Each Claude Code role runs a target repository's hooks, `env`, skills' `allowed-tools` and `.mcp.json` servers, and the operator's own hooks, with the operator's permissions. The refiner and judge do it in the main checkout. The fix is to choose isolating flags per role. Before choosing, run a small local probe: a scratch repository whose hooks, MCP server and CLAUDE.md each leave a marker, run with the exact builder and refiner arguments and then with each candidate flag, and Codex with and without `--ignore-user-config --ignore-rules`. Then:

- pass the chosen flags for every role and record in the run which configuration loaded;
- add `.claude/`, `.mcp.json`, `.codex/` and `.github/hooks/` to the default protected paths, so a candidate can't plant configuration for a repair or the reviewer;
- fail a run when a hook Waymark didn't configure fires (`--include-hook-events`).

This affects three plans in PR #57. The isolation plan's premise that the refiner and judge have "no shell" is false today. The harness plan's role profiles should carry the flags. The agent review plan runs the reviewer in the candidate's worktree, which is where a planted hook would run. Dropping CLAUDE.md changes what the roles know, so it is a decision for the user, not a default.

**2. Check that the holdout tests fail before the build.** All three research threads reached this independently. After the expectation author's tests are accepted, run them at the base revision. They must fail; a compile failure counts only as weak evidence. Verification already shows they pass at the candidate. A holdout that passes at base checks nothing the change adds. This needs no model and no new language support, and it runs on the existing check machinery. It decides whether a person approves tests on evidence rather than on the author's coverage claim, and it is the evidence an `--after-build when_untested` policy should require. Mutation testing of the changed code is the next step, and it is per language: Gremlins for Go.

**3. Record what each run was told.** Add a digest of the built-in rules, the role's instructions and the `ctx` revision to every refiner, author, builder and judge run. Without it, no comparison of two prompt versions is possible. This is cheap and should come before any eval of the agents.

Smaller points:

- **Classify failures.** A provider error on the first attempt shows up as a repair, and on the last as a failed job, the same as a builder defect ([metrics.go:161-176](../../internal/factory/metrics.go#L161-L176)). The first-pass rate divides by accepted jobs only ([metrics.go:187-189](../../internal/factory/metrics.go#L187-L189)), so it is the share of accepted jobs that needed no repair. It isn't pass@1.
- **Harden what verification reads.** Reject candidate symlinks that leave the tree, protect everything a check reads (module files, Makefiles, CI config, test data), and scan the candidate diff for secrets.
- **Give the reviewer Harbor's two questions:** did the builder game the checks, and was the task clear enough to build? The second is what the dispute judge routes on.

Later, once the factory has real job history (the local databases hold none yet):

- Replay 20 to 30 finished tickets in fresh clones with later commits removed, three runs each. Report how often all three succeed, with cost and paired differences between prompt versions.
- Measure the agent reviewer's precision and recall against labelled jobs, with surviving mutants as known misses.
- A `Stop` hook that runs the holdouts before the builder may finish. It is optional and advisory, and only worth it after isolation.

**Don't take.** Hooks as the enforcement layer. LLM-judged test-order hooks. Command denylists. Evalite. Model graders without human calibration. Rates over a handful of jobs with no prompt version and no error bars. Logging through hooks, which duplicates the event stream Waymark already reads.

**The orchestrate skill, bounded repair and agent review.** The orchestrate skill dispatches workers through the host environment's own tools, so the same configuration loading applies to its sessions. That's tolerable when the user is the one running it, but it's worth knowing. Bounded repair gains nothing new beyond failure classification. Agent review gains the most. The reviewer must run isolated, it should be measured before it gates anything, and Matt's design, where the reviewer also fixes, is the one Waymark deliberately didn't choose.

## Evidence and limitations

- **Read, not run.** Repositories were cloned into empty scratch directories and read with `git`, `cat`, `sed` and `grep`. No package, hook, agent, eval or probe was installed or run. Test files show what is asserted, not that it passes.
- **Delegated reading.** Three research agents read the hook documentation, the eval literature and the community projects. Every claim this note relies on for a recommendation was rechecked by hand: the Claude Code permissions, hooks, CLI and environment-variable pages; the Codex rules, hooks page and `cli.rs`; the Copilot programmatic reference; the Anthropic evals post; OpenAI's SWE-bench figure through its indexed text, because the page refused automated fetches; SWE-Bench+ through arXiv; the cited GitHub issues; Gremlins' release; Inspect's reducers; Aider's test reset; Harbor's hardening commit; and every Waymark file and line cited.
- **Docs move.** The Claude Code, Codex and Copilot pages carry no version. They were read on 29 October 2026, when the Claude Code changelog was at 2.1.312 and Codex at `rust-v0.166.0`.
- **Not established.** Whether `--safe-mode` keeps a project's `env` block and `apiKeyHelper`. Whether `--restricted` loads CLAUDE.md or `.mcp.json`. Whether a planted hook would reach the planned reviewer in practice. The probe in recommendation 1 settles all three.
- **Local data.** The factory database counts come from read-only queries of the three trial databases under `~/.waymark/` on this machine. The research agents also read this machine's user-level Claude Code and Codex configuration. Those findings were reported to the user directly and are not recorded in this public note.
- **Not covered.** The course's lessons, which are unreleased. Matt Pocock's `ai-hero-cli`, beyond one hooks reference. OpenAI Codex cloud, Cursor and Cline. Whether any adopter's factory outcome matches its claims.
- **Waymark side.** Statements about Waymark come from `dev` at `afe9989` and the PR #57 plans at `44e1379`. Nothing in Waymark was changed or run for this note.
