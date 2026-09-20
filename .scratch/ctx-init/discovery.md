# Project initialization discovery

The requested `ctx init` command prepares a project for ctx from the project root. An agent inspects existing project files, asks about missing choices, and installs or adapts the skills, instructions, prompts, and hooks needed for the chosen workflow.

This document retains the design interview and its proposals. The current direction is to trial ctx on a second project before defining the first `ctx init` release. The debate history below does not establish that release's scope.

## Confirmed requirements

- Ask which tracker the project uses and preserve that tracker as authoritative.
- If the project has no tracker, offer local file tracking.
- Account for both a human running the command and an agent running it inside its existing tool or harness.
- Discover existing suitable files and their locations before deciding what to create.
- Give human callers a concise interactive entry flow rather than printing the full onboarding prompt in the terminal. Provide an explicit agent mode, proposed as `ctx init --agent`, for the agent to obtain the workflow.
- Establish which harnesses the project uses before installing harness-specific files. Support projects that use multiple harnesses.
- Make installed skills available to each selected harness. Copies in native directories and one shared location with references are alternatives still under discussion.

## Existing behavior and constraints

- [Connect projects](../../docs/connect-projects.md) documents `ctx setup`. It binds an existing project record store to a directory. It does not create records or install agent guidance.
- [Product vision](../../docs/vision.md) puts workflow skills inside an existing agent environment. Launching and managing agents remain later capabilities.
- The [glossary](../../CONTEXT.md) distinguishes a project from its repositories and record store. Running at a repository root does not establish that a new project is needed.
- [ADR 0001](../../docs/adr/0001-one-okf-bundle-per-project.md) requires one authoritative OKF bundle per project. Preserving an external tracker must not silently create a duplicate local backlog.
- Existing repository skills include instructions for building this repository's CLI. Installing those files unchanged in another project would carry incorrect assumptions.

## Three-agent debate

The user requested three independent proposals and peer critique.

| Proposal | Strength | Objection raised by peers | Revised position |
| --- | --- | --- | --- |
| Return a self-contained onboarding prompt for the existing agent | Works immediately in a running agent conversation and needs no installation first | A human must transfer the prompt, and a printed prompt alone does not prepare future sessions | Keep the shared workflow, then install durable guidance after discovery |
| Install a harness-specific onboarding entry point first | Gives a human a concrete way to start setup in their chosen tool | Adds file ownership and reload concerns before discovery; an active agent needs no initial installation | Return the workflow directly, with optional delivery through a supported tool |
| Run a terminal wizard with structured operations for agents | Makes simple configuration choices explicit and repeatable | Cannot establish semantic equivalence of arbitrary existing documents and duplicates the agent interview | Limit terminal interaction to entering the agent workflow; keep narrow deterministic checks |

The agents converged on one agent-led onboarding workflow. A running agent consumes it immediately. A human receives a concrete instruction to continue in their chosen agent. The agent discovers existing material, asks unresolved questions, and then prepares the project.

The recommendation distinguishes making onboarding available from completing project setup. It also separates file and configuration checks from the agent's judgment that existing documents serve the intended purpose.

The user refined this recommendation: the normal terminal flow should be concise and interactive, with instructions to use an explicit agent mode such as `--agent`. Installing skills requires knowing the selected harnesses. Multiple harnesses must receive usable skills, through copies or a shared location with references. The storage strategy remains undecided.

## Open questions

- What does the human flow install before the agent begins semantic project discovery, if anything?
- Should multiple harnesses share one skill directory where supported, or receive managed copies? How are edits and updates kept consistent?
- Which tools must have tested installation support in the first issue? A generic prompt does not establish that every tool's hooks and skills are supported.
- Which installed capabilities make a project ready, and which remain optional?
- How should setup adapt existing guidance, handle conflicting instructions, and behave on repeated runs?
- What usable outcome can setup promise for an external tracker before a matching ctx connector exists?

## Harness documentation checked on 2026-09-20

- [Codex skill documentation](https://learn.chatgpt.com/docs/build-skills) documents project skills under `.agents/skills` and support for symlinked skill folders.
- [GitHub Copilot skill documentation](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills) lists `.github/skills`, `.claude/skills`, and `.agents/skills` as project skill locations.
- [Claude Code skill documentation](https://code.claude.com/docs/en/skills#choose-where-skills-load) documents `.claude/skills` and individual skill-folder symlinks to other directories.

A candidate shared layout stores skill files in `.agents/skills`, which Codex and Copilot read directly, and creates individual links under `.claude/skills` for Claude Code. Existing project conventions and operating-system support need consideration before selecting this layout. This is a proposal, not a confirmed requirement. Shared skill discovery does not establish shared formats for hooks or instruction files.

## Next step

Review the [Opsbase findings](opsbase-trial.md) and the rewritten [implementation issue](../records/ctx-init/issues/01-agent-guided-project-initialization.md). Keep the issue draft, needs-info, and unstarted until its [scope decision](../records/ctx-init/decisions/01-initialization-scope.md) is resolved from those observations. The local trial has run; further debate is not a substitute for any untested case.

## Trial before initialization development

On 2026-09-20, the user replaced the debate-led implementation sequence with manual bootstrap, real use, scope rewrite, then implementation. Recovery work is limited to fixes until this trial is done. The rule is to resolve questions through use before establishing new schemas or mandatory stages.

The manual bootstrap starts with existing materials:

1. Prepare a record store and a Project manifest with identity, title, goals, current commitments, and open decisions, while preserving any existing tracker authority.
2. Bind the records with `ctx setup` and authorize only the source roots the project needs.
3. Copy and trim the task-context and recovery-notes skills and the acceptance procedure for the harness actually used. Follow their required supporting references and remove assumptions about building ctx in the target project.
4. Add concise pointers to the adopted conventions in the project's existing agent instructions.
5. Select or author one real unfinished task and start working with ctx.

Record each copied file, adaptation, merge with existing guidance, unnecessary step, and missing capability as it occurs. Keep raw verification output out of tracked records. After the trial, add a findings section here that compares the observed setup work with the earlier recommendations. Do not invent findings in advance.

Only harnesses and workflows exercised by the trial justify initial support claims. If no hook is needed, do not add one to the first release. If no external tracker is exercised, leave its integration question deferred.

The user selected Opsbase and its GitHub issue 165. The [friction log](opsbase-trial.md) records the completed local bootstrap, task work, verification, and comparison with the debate. Opsbase changes await review and hosted CI. The init draft has been narrowed; its scope decision remains open for review of the findings.

## Follow-up debate: installation and first-issue scope

The user requested a second three-agent debate about skill storage, the human entry flow, and supported harnesses. The user subsequently accepted the summarized recommendations, including the primary agent's copy-fallback recommendation. The issue linked above records the accepted requirements.

### Shared recommendations

- Preserve an existing suitable skills layout. For new installations, prefer one shared source where selected harnesses support it. Link individual skills where necessary, rather than replacing whole skill directories.
- Human `ctx init` selects harnesses and returns a short instruction containing an agent-mode command that carries those selections. No bootstrap files need to be installed before project discovery. The exact argument syntax remains unspecified.
- `ctx init --agent` returns the onboarding workflow without terminal prompts. Missing choices become questions in the agent conversation.
- Include Codex, Claude Code, and a specified GitHub Copilot client, including combinations, for a defined common skill workflow. Declare and test the actual clients and platforms. Directory documentation alone does not establish working installation.
- Discover existing equivalents before making changes. Preserve edited files and expose conflicts. Report unfinished setup accurately.
- Define hooks separately. A supported skills directory does not imply compatible hook or instruction-file formats.

### Arguments that changed the proposals

One agent initially favored Codex and Copilot only. The peers argued that two tools reading the same directory would avoid the main multiple-layout requirement. That agent accepted including Claude Code through a bounded installation strategy.

The agents initially excluded copies because synchronizing editable copies would add scope. A further challenge distinguished initial copies and conservative reruns from automatic synchronization. The installation-focused agent revised its recommendation to allow copies when links are unsuitable. Its proposed rule is to skip identical files and require explicit reconciliation for differing existing contents, with no automatic propagation of manual edits.

The other two agents' final recommendations favored shared files and links, reporting an incomplete installation when those arrangements are unavailable. The primary agent recommended the bounded copy fallback because it supports the requested multiple-harness outcome without requiring automatic synchronization. The user accepted that recommendation.
