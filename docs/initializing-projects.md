# Prepare a project for ctx

Run `ctx init` from the target project root with a supplied binary. It creates agent guidance and an empty Project, then binds that Project through the existing setup operation. It never prompts or creates work items.

First establish the project's authoritative tracker and the agent tool's actual skills directory. Read the existing instructions and preserve their conventions. A local roadmap is a tracker even when the repository has no remote issues. If the tracker is unclear, ask which one is used. Offer local file tracking only when none exists.

## Initialize the project

For a project using GitHub Issues and `.agents/skills`, run:

```sh
/absolute/path/to/ctx init \
	--title 'Widget' \
	--tracker 'GitHub Issues in example/widget' \
	--skills-dir .agents/skills \
	--docs-dir docs/agents/ctx \
	--allow-source docs
```

`--title`, `--tracker`, `--skills-dir`, and `--docs-dir` are required. The tracker value records authority in the installed guidance. It does not configure a connector. For a local roadmap, name its path, such as `--tracker 'docs/roadmap.md'`.

Choose the directory your active agent uses. The trials exercised ordinary files under `.agents/skills` with Codex and individual links into that directory from `.claude/skills` with Claude Code. Init writes ordinary files into the supplied directory. It does not detect tools or create symlinks.

All path flags resolve from the current directory. `--records` defaults to `.context/records` and can select a separate record store. `--local-dir` defaults to `.context/trial` and must stay inside the target project. Keep records separate from disposable local files and recovery cache. Init rejects layouts that would ignore its durable guidance or records.

Repeat `--allow-source` for each existing source directory the reader needs. Roots are explicit; init does not authorize the entire project automatically. When tasks need root `CONTEXT.md` or `CONTEXT-MAP.md`, select the directory containing those files. Future task selection may require a reviewed setup change.

## Review the result

Init reports each file it creates, preserves, skips, or appends to, plus the binding result. It installs:

- `task-context/SKILL.md`, `recovery-notes/SKILL.md`, and `recovery-notes/PROFILE.md` under the selected skills directory.
- `orchestrate/SKILL.md` and its required `VERIFICATION.md` reference under that skills directory. Its frontmatter disables Claude Code's automatic invocation, and its `agents/openai.yaml` disables Codex's.
- `retro/SKILL.md` under that skills directory. Its frontmatter disables Claude Code's automatic invocation, and its `agents/openai.yaml` disables Codex's.
- `cli.md`, `acceptance.md`, and `tracker.md` under the selected docs directory.
- `project.md` under the records directory, with a new UUID, the supplied title, empty commitments, and no decisions.
- `ctx-version.txt` under the local directory, containing the running binary's exact version output.

Existing manifests keep their identity and content. The title flag applies only to a new manifest. Other identical files remain unchanged; differing files and file symlinks are skipped for manual review. Init never replaces them. A differing version file is also preserved, so retain the new version separately if you deliberately rerun with another binary.

Init appends missing rules to the project's `.gitignore` for the local directory, `.context-cache/`, and shared configuration locks. It also appends `/acceptances/` to a `.gitignore` in the selected records directory. Store snapshots, raw tracker responses, evidence, and disposable binaries under the local directory. The ignore rules preserve authored records and guidance. Existing Git rules and already tracked files still need the project's own review.

The existing setup implementation writes `.context/config.md` and coordinates writers through retained configuration locks. It preserves an identical binding and rejects a conflicting one. Follow [Connect existing records](connect-projects.md) for a deliberate binding replacement. Init never passes `--replace`.

Exit status `0` means file preparation and binding succeeded. Status `1` means a file or binding conflict needs attention. Status `2` means invalid input or an execution failure. Earlier successful writes remain visible in the report if a later operation fails; init does not roll back the project.

Review the printed pointer block before adding it to the existing instructions file. Init leaves AGENTS.md, CLAUDE.md, existing hooks, and tracker contents unchanged. The generated skills refer to this installation's absolute binary and project paths. If those locations change, review and adapt the guidance manually.

The CLI pointer tells agents to read the installed `cli.md` before selecting work, reporting project status, implementing a known ticket, or continuing work. That guide selects orient, context, or resume and links the specialized procedures. Add the reviewed block to the instructions file your agent loads, such as AGENTS.md or CLAUDE.md, so future sessions discover it. Installing files alone does not establish that instruction entry point.

## Select work after preparation

Inspect the empty project with the supplied binary:

```sh
/absolute/path/to/ctx orient
```

Choose a real task through the existing tracker. Follow the installed `tracker.md` to select source documents and create a local reader record only when needed. For an external tracker, preserve its original response, source identity, retrieval time, and any local adaptations. A captured issue is not a second editable backlog. An existing local plan remains authoritative.

The installed task-context and recovery-notes skills use the saved binding. Verify source selection with context and resume after selecting a task. Their reports do not establish remote freshness, completion, human review, or hosted CI. Publish a checkpoint only when it preserves useful information beyond the ticket and checkout.

## Invoke checkpoint orchestration explicitly

Use `$orchestrate` for a named work item when you want the agent environment to coordinate test-first checkpoint implementation, checks, reviews, and corrections. Installation alone does not invoke this workflow. Init keeps its metadata explicit-only and leaves routing, hooks, and the normal implementation workflow under the project's control.

The orchestrator holds the whole task in its records on disk and continues in a fresh session at each checkpoint boundary. It dispatches fresh implementing and repairing agents by pointer to one checkpoint file, with one writer active at a time. Required testing and Standards and Specification review procedures ship in `orchestrate/VERIFICATION.md`. Target projects need no separate TDD or code-review skill, trial documents, or Waymark source checkout. Links resolve from the configured skills and docs directories, and reader commands use the supplied binary and target project binding.

Apply the target project's human review and publication policy. Reuse existing authorization within its scope, and wait for actual human responses where that policy requires a gate. Independent combined verification establishes evidence for the whole result. Checkpoint progress, acceptance, human review, and publication remain separate decisions.

## Review a work item's sessions explicitly

Use `$retro` with named Claude Code or Codex sessions of one work item when you want to measure a run and improve the environment for the next one. Installation alone does not invoke it; init keeps its metadata explicit-only.

The skill reads the transcripts and the work item's records, reports measures of the run with their sources, and gives each proposed change one home. The installed `retro/SKILL.md` lists the measures and homes.

It writes only its report, beside the project's ignored local evidence, before you approve a proposal. Approved guidance records its source and scope. The report stays out of Git.

## Maintain the embedded guidance

The canonical templates live under `internal/initialization/templates/`. The binary embeds them directly. Edit these templates when changing the distributed skills, recovery profile, or acceptance procedure, then regenerate this repository's development copies:

```sh
go generate ./internal/initialization
go test ./internal/initialization ./internal/cli
```

The `-update` test flag is local to `internal/initialization`. Use it only when testing that package.

The template test checks those copies for drift. This repository retains its own authored issue-tracker conventions; target projects receive the trimmed tracker template. Target initialization needs no source checkout, generation command, or network access.
