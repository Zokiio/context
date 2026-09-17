# Connect projects and group them in a workspace

Use `ctx setup` to connect an existing records directory to a working directory. Replace `ctx` in these examples with the path to your [built binary](../readme.md#build-and-read-this-project).

## Connect existing records

Choose a records directory that already contains `project.md` with `type: Project`, a nonempty `id`, and a nonempty `title` in YAML frontmatter. Setup does not create records.

From the checkout, preview a shared binding:

```sh
ctx setup --records .scratch/records --allow-source docs --dry-run
```

Review the resolved binding directory, records, allowed sources, destination, and exact proposed document. Then apply the binding:

```sh
ctx setup --records .scratch/records --allow-source docs
```

Setup writes `.context/config.md` in the binding directory. The binding directory defaults to cwd. All command paths resolve from cwd, including paths supplied while you are in a nested directory.

To bind a different directory, supply `--directory`:

```sh
ctx setup --directory /work/widget --records /work/records/widget --allow-source /work/widget/docs
```

Choose allowed sources that contain the linked documents outside the record store. Repeat `--allow-source` for separate directories. For a project that links a root-level `CONTEXT.md`, authorize the directory that contains it.

Read the selected project:

```sh
ctx orient
ctx context --ticket issues/task.md
```

From an interactive terminal, run `ctx setup` without `--records` for guided prompts. Supplying `--records` uses the flags and defaults without prompts or another confirmation. Noninteractive input requires `--records`.

## Save a personal registration

To keep the binding in your personal registry, add `--personal`. To select the project by name from any directory, also add an alias:

```sh
ctx setup --personal --directory /work/widget --records /work/records/widget --allow-source /work/widget/docs --alias widget
ctx orient --project @widget
```

Personal setup writes `~/.context/config.md`. Shared setup cannot use that reserved file, so use `--personal` when the binding directory is your home directory.

Shared setup saves relative paths. Personal setup saves absolute paths. Moving a checkout together with its shared records and configuration preserves their relative relationship. After a personal path changes, update the affected registration.

## Replace a binding

If the existing binding differs, review a replacement before applying it:

```sh
ctx setup --personal --directory /work/widget --records /work/new-records/widget --allow-source /work/widget/docs --replace --dry-run
ctx setup --personal --directory /work/widget --records /work/new-records/widget --allow-source /work/widget/docs --replace
```

Supply the complete allowed-source list on replacement. Omitting `--allow-source` replaces that list with an empty list. The summary reports removed roots, replaced records, and removed aliases.

Omit `--alias` to retain the existing personal alias. Supply a different alias to replace it. Setup matches personal entries by canonical binding directory and retains the entry's key. An alias already assigned to another entry cannot retarget that entry.

`--replace` changes only the chosen destination file. If another effective declaration still conflicts, repair the declaration named in the error before rerunning setup.

Identical setup is a no-op. Dry-run and no-op create no directories or lockfiles. Updates preserve unrelated entries, unknown metadata, Markdown body bytes, and existing file permissions. Malformed configuration requires an explicit repair outside setup.

On macOS and Linux, replacement preserves owner, group, mode, and access-control lists. If setup cannot preserve or inspect those permissions, it leaves the configuration intact and reports the failure. Windows supports creation but rejects replacement of existing configuration. Other operating systems reject writes. Dry-run and identical setup remain available.

Setup uses atomic replacement and checks for concurrent file changes. A changed document requires a new setup invocation. Write failures preserve the previous usable configuration.

Changed setup operations retain a lock beside the personal registry, including shared setup. Shared writes also retain a lock beside their destination configuration. The lock name is `<physical configuration path>.lock`, usually `.context/config.md.lock`. These locks let shared and personal writers using the same home directory check conflicts before either writes. Both locations must be writable. Shared setup creates no personal configuration. Keep the lock files out of Git and leave them in place while setup processes may be active. For a symlinked configuration, setup updates the physical target and leaves the symlink in place.

## Author a workspace

Create a workspace declaration in `/work/team/.context/config.md`. If the file already contains a project binding, retain that binding and add the `workspace` mapping. For a new file, use this document:

```markdown
---
type: ContextConfig
version: 1
workspace:
  id: team-projects
  title: Team projects
  members:
    - key: widget
      title: Widget
      directory: ../widget
      records: ../widget/.scratch/records
      allowSources:
        - ../widget/docs
    - key: research
      title: Research
      records: ../research-records
---

Projects used by the team. Each member keeps its own authoritative records.
```

Adjust the paths to your existing locations. Relative paths start at `/work/team/.context`, the configuration file's directory. The `widget` member above points to `/work/team/widget/.scratch/records`.

List the members in text or JSON:

```sh
ctx orient --workspace /work/team
ctx orient --workspace /work/team --json
```

Use `--workspace` explicitly when the same directory also has a project binding. To select a member, copy its `select` command from the text report. The commands use POSIX shell quoting and include that member's allowed sources.

For the `widget` declaration above, the command is:

```sh
ctx orient --bundle /work/team/.context/../widget/.scratch/records --allow-source /work/team/.context/../widget/docs
```

For process execution from JSON, pass a member's `selectionArgs` array directly as arguments. The array starts with `ctx` and `orient`, followed by `--bundle` and any member-specific `--allow-source` flags. Replace the first element with your binary path if needed. No shell parsing is required.

Workspace orientation reports access to each records directory. It does not inspect project manifests, evaluate work readiness, or check the checkout and allowed-source directories. An unavailable member remains in the list. Membership alone creates no cwd binding and grants no document access to another member.

To read a member's ticket, select its records explicitly and keep its allowed-source flags:

```sh
ctx context --bundle /work/team/widget/.scratch/records --allow-source /work/team/widget/docs --ticket issues/task.md
```

## Inspect selection or recover direct access

To see why discovery selected a scope, add `--explain-scope`:

```sh
ctx orient --project /work/widget --json --explain-scope
```

The report stays on stdout. Scope details and errors go to stderr.

If discovery fails because a mapping or configuration is broken, repair the named declaration. To read known records directly while investigating, use `--bundle` and supply the required roots:

```sh
ctx orient --bundle /work/records/widget --allow-source /work/widget/docs
```

For old task-context commands that used `--project` as a direct records selector, replace it with `--bundle`. Direct `context` still accepts a bundle without `project.md`. Direct `orient` retains its manifest validation and can return a partial report.
