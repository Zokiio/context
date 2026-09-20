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
- `acceptance.md` and `tracker.md` under the selected docs directory.
- `project.md` under the records directory, with a new UUID, the supplied title, empty commitments, and no decisions.
- `ctx-version.txt` under the local directory, containing the running binary's exact version output.

Existing manifests keep their identity and content. The title flag applies only to a new manifest. Other identical files remain unchanged; differing files and file symlinks are skipped for manual review. Init never replaces them. A differing version file is also preserved, so retain the new version separately if you deliberately rerun with another binary.

Init appends missing rules to the project's `.gitignore` for the local directory, `.context-cache/`, and shared configuration locks. It also appends `/acceptances/` to a `.gitignore` in the selected records directory. Store snapshots, raw tracker responses, evidence, and disposable binaries under the local directory. The ignore rules preserve authored records and guidance. Existing Git rules and already tracked files still need the project's own review.

The existing setup implementation writes `.context/config.md` and coordinates writers through retained configuration locks. It preserves an identical binding and rejects a conflicting one. Follow [Connect existing records](connect-projects.md) for a deliberate binding replacement. Init never passes `--replace`.

Exit status `0` means file preparation and binding succeeded. Status `1` means a file or binding conflict needs attention. Status `2` means invalid input or an execution failure. Earlier successful writes remain visible in the report if a later operation fails; init does not roll back the project.

Review the printed pointer block before adding it to the existing instructions file. Init leaves AGENTS.md, CLAUDE.md, existing hooks, and tracker contents unchanged. The generated skills refer to this installation's absolute binary and project paths. If those locations change, review and adapt the guidance manually.

## Select work after preparation

Inspect the empty project with the supplied binary:

```sh
/absolute/path/to/ctx orient
```

Choose a real task through the existing tracker. Follow the installed `tracker.md` to select source documents and create a local reader record only when needed. For an external tracker, preserve its original response, source identity, retrieval time, and any local adaptations. A captured issue is not a second editable backlog. An existing local plan remains authoritative.

The installed task-context and recovery-notes skills use the saved binding. Verify source selection with context and resume after selecting a task. Their reports do not establish remote freshness, completion, human review, or hosted CI. Publish a checkpoint only when it preserves useful information beyond the ticket and checkout.

## Maintain the embedded guidance

The canonical templates live under `internal/initialization/templates/`. The binary embeds them directly. Edit these templates when changing the distributed skills, recovery profile, or acceptance procedure, then regenerate this repository's development copies:

```sh
go generate ./internal/initialization
go test ./internal/initialization ./internal/cli
```

The template test checks those copies for drift. This repository retains its own authored issue-tracker conventions; target projects receive the trimmed tracker template. Target initialization needs no source checkout, generation command, or network access.
