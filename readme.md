# Project and context management

`ctx` is the development executable for this project. The product name remains open.

The CLI reads project goals, work items, dependencies, and supporting documents from local Markdown records. It discovers the project from your current directory. Records can live in a checkout or a separate directory, and workspaces can group them without copying them.

## Build and read this project

From the repository root, build with Go 1.25 or later:

```sh
go build -o /tmp/ctx ./cmd/ctx
```

Read the project's orientation:

```sh
/tmp/ctx orient --max-files 500 --max-bytes 8388608
```

The committed [.context/config.md](.context/config.md) selects `.scratch/records` and authorizes supporting documents in this repository. Discovery also works from nested directories.

This repository's accumulated acceptance evidence exceeds the reader defaults of 100 files and 1 MiB. The example raises those limits for this invocation. Defaults remain unchanged.

For structured orientation, add `--json`:

```sh
/tmp/ctx orient --json --max-files 500 --max-bytes 8388608
```

Read one ticket and its linked requirements:

```sh
/tmp/ctx context --ticket project-discovery/issues/06-document-and-trial.md
```

Ticket paths are relative to the selected records directory. `context` always writes JSON. Both reader commands leave files unchanged and never prompt.

## Connect other projects

Follow [Connect projects and group them in a workspace](docs/connect-projects.md) to register existing records. Use the [Discovery reference](docs/discovery.md) for configuration fields, selection rules, source access, and exit statuses. The [Reader reference](docs/readers.md) covers project reports, task context, fingerprints, and recorded acceptance.

If an older command used `--project` for a directory without a discoverable Project marker, change that selector to `--bundle`:

```sh
/tmp/ctx context --bundle /path/to/records --ticket issues/task.md
```

`--bundle` bypasses discovery configuration. Supply each required supporting-document directory with `--allow-source`.

## Verify changes

Run the repository checks:

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

The [retained discovery trial](.scratch/project-discovery/evidence/07-trial.json) and [trial driver](.scratch/project-discovery/trial.py) record the end-to-end checks.

The [product vision](docs/vision.md), [domain glossary](CONTEXT.md), and [local issue tracker conventions](docs/agents/issue-tracker.md) describe the project's direction and contribution workflow.
