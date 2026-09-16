# Discovery reference

`ctx orient` and `ctx context` share one discovery operation. The selected record store and authorized source directories are inputs to the existing readers. Discovery does not change cwd, prompt, or write configuration.

## Selectors and paths

| Selector | Meaning |
| --- | --- |
| None | Discovery starts at cwd. |
| `--project PATH` | Project discovery starts at `PATH` and follows its ancestors. |
| `--project @NAME` | Selection uses the named personal project registration. |
| `--workspace PATH` | Workspace discovery starts at `PATH` and ignores project candidates. |
| `--workspace @NAME` | Selection uses the named personal workspace registration. |
| `--bundle PATH` | Direct records access bypasses all discovery configuration. |

The three selector flags are mutually exclusive. Empty or repeated selectors fail. Filesystem selector paths must name directories. A leading `@` denotes an alias for project and workspace selectors. `./@name` denotes a literal directory.

Command paths resolve from cwd. Saved relative paths resolve from the encountered configuration file's directory, including when the configuration file is a symlink. Absolute saved paths remain absolute. Saved values have no tilde expansion, environment-variable interpolation, or executable expressions.

`--bundle` receives only explicit `--allow-source` roots. It bypasses malformed personal or shared configuration. Direct `context` does not require a Project manifest. Direct `orient` retains its existing manifest validation and partial-report behavior.

The old direct-directory meaning of `--project` has changed. A manifestless records directory now requires `--bundle` for direct task-context access. Failed project discovery never falls back to treating the selector as a raw records directory.

Ticket paths remain relative to the selected records directory. Bare `ctx` displays command guidance instead of performing discovery.

## Configuration documents

Configuration is UTF-8 Markdown with YAML frontmatter. Every document requires `type: ContextConfig` and integer `version: 1`. Unknown metadata and Markdown prose are retained by setup.

Duplicate YAML keys, wrong known-field types, unsupported versions, empty required strings, duplicate entry keys, and duplicate aliases within one selector kind are invalid. Shared and personal profile fields cannot appear in the same file.

### Shared profile

`<directory>/.context/config.md` declares scope at `<directory>`. At least one of `project` and `workspace` is required. Both can be present.

| Field | Type and meaning |
| --- | --- |
| `project` | Optional project binding. |
| `project.records` | Required nonempty path to an existing Project bundle. |
| `project.allowSources` | Optional array of nonempty directory paths. Defaults to empty. |
| `workspace` | Optional workspace declaration with the fields below. |

For example, this shared document maps its containing checkout to `.scratch/records` and authorizes linked documents beneath `docs`:

```markdown
---
type: ContextConfig
version: 1
project:
  records: ../.scratch/records
  allowSources:
    - ../docs
---

Records and supporting documents for this checkout.
```

`project.md` at a records directory's root is also a project marker. A valid marker has `type: Project`, a nonempty `id`, and a nonempty `title`. The marker supplies identity and location, but no external document roots.

### Personal profile

`~/.context/config.md` is the personal registry. Its location does not itself bind the home directory to a project or workspace. The path is reserved and is never also interpreted as a shared marker during ancestor traversal.

The `projects` and `workspaces` arrays are optional and default to empty. Entry keys are unique within each array. Aliases are unique within the project or workspace selector kind, so the same alias can name one project and one workspace.

| Project entry field | Type and meaning |
| --- | --- |
| `key` | Required nonempty registration identity. |
| `alias` | Optional string used by `--project @NAME`. |
| `directory` | Required nonempty path that creates a cwd binding. |
| `records` | Required nonempty path to the Project bundle. |
| `allowSources` | Optional array of nonempty directory paths. Defaults to empty. |

A workspace entry has `key`, optional `alias`, optional `directory`, and the workspace declaration fields below. A workspace entry without `directory` creates no cwd binding. Its alias can still select it.

Example personal registry:

```markdown
---
type: ContextConfig
version: 1
projects:
  - key: widget-checkout
    alias: widget
    directory: /work/widget
    records: /work/records/widget
    allowSources:
      - /work/widget/docs
workspaces:
  - key: team-navigation
    alias: team
    id: team-projects
    title: Team projects
    members: []
---

Local registrations for this machine.
```

Project aliases select their saved entries independently of cwd. If cwd lookup fails, aliases still work; additional `--allow-source` paths must then be absolute. Alias selection requires available records and allowed roots, but an unavailable checkout does not prevent selection. Unrelated missing registered targets do not invalidate the registry's structure.

### Workspace declaration

| Field | Type and meaning |
| --- | --- |
| `id` | Required nonempty stable workspace identity. |
| `title` | Required nonempty display title. |
| `members` | Required array. An empty array is valid. |
| `members[].key` | Required nonempty key, unique within this workspace. |
| `members[].title` | Optional display title. |
| `members[].records` | Required nonempty records path. |
| `members[].directory` | Optional nonempty checkout path. |
| `members[].allowSources` | Optional array of nonempty directory paths. Defaults to empty. |

Member paths resolve from the declaring configuration file. Membership creates no cwd binding. Matching project IDs do not collapse distinct checkouts into one member. Matching workspace IDs do not merge differing declarations.

## Directory selection

Discovery canonicalizes the start directory and follows its physical ancestors. Symlinks do not add a second search through lexical ancestors. Search can cross Git, home, and mount boundaries through filesystem root. It does not search downward for a conventional records directory.

The personal registry is loaded and structurally validated once. Missing personal configuration is normal. A malformed or unreadable registry fails discovery because it could hide a more specific binding.

At each directory depth, discovery compares applicable personal bindings, the local shared configuration, and a Project marker. The nearest applicable depth wins across storage locations. A narrower personal binding can therefore override a broader shared binding.

At equal depth, implicit selection prefers a project to a workspace. Explicit selectors filter candidates by kind. A nearer workspace stops implicit selection from reaching a broader project. `context` then reports that project selection is required, even when the workspace has one member.

Conflicting candidates of the selected kind at equal depth fail with their origins. Project mappings coalesce only when canonical records and effective allowed-source sets agree. An agreeing mapping can supply roots alongside a bare Project marker. Two authored mappings with different roots conflict. Canonically identical workspace declarations can coalesce.

Selected records and allowed roots must be available. A broken selected mapping fails without fallback. A malformed reserved `project.md` marker also fails when project scope can affect selection. Explicit workspace discovery ignores project markers. Broader local files are not read once they cannot win.

## Source authorization

A selected project's configured roots authorize supporting documents outside its record store. Repeated `--allow-source PATH` flags add roots for that invocation only. They do not update configuration or resolve conflicts between different configured root sets.

Reader containment checks continue to enforce the records and authorized-source boundaries, including symlink targets. Authorization permits linked sources to be read. It does not select every file under an allowed directory.

Each workspace member has its own roots. A member command carries those roots explicitly through `--allow-source`. Workspace membership and roots from another member grant no access. `--allow-source` on a workspace navigation invocation does not add roots to the generated member commands.

## Setup flags and writes

| Flag | Meaning |
| --- | --- |
| `--records PATH` | Existing records with a valid Project manifest. Required for noninteractive setup. |
| `--directory PATH` | Binding directory. Defaults to cwd. |
| `--personal` | Destination is the personal registry instead of a shared file. |
| `--alias NAME` | Personal project alias. Requires `--personal`. |
| `--allow-source PATH` | Saved supporting-document root. Repeat for multiple roots. |
| `--dry-run` | Resolved summary and exact proposed document, with no writes. |
| `--replace` | Permission to update the conflicting binding in the chosen destination file. |

Setup resolves and validates inputs before generating shared relative paths or personal absolute paths. Guided prompts occur only on an interactive terminal when `--records` is omitted. Supplied records use defaults for omitted options without another confirmation.

The summary precedes every write. Identical setup preserves the original file and creates no lock. Updates preserve unrelated entries, unknown metadata and YAML tags, Markdown body bytes, and file permissions. An update can expand aliases to keep unrelated values unchanged. If recursive aliases prevent preserving the document, setup rejects the update before writing. Setup never creates records or silently repairs malformed configuration.

Personal entry identity is the canonical binding directory. Ambiguous matches fail. A new entry receives a UUID key. Replacement retains that key and retains the alias when `--alias` is omitted. A supplied alias cannot collide with another project entry. Records and the supplied roots replace those fields under `--replace`. Omitted roots mean an empty list.

Setup validates the proposed binding against other effective declarations. `--replace` cannot override a conflict that remains in another file. Atomic replacement and concurrent-change checks protect the previous usable file from failed writes. Local writers coordinate through the persistent `<physical configuration path>.lock` file. Relative saved values still use the encountered configuration path as their base.

## Reports, limits, and failures

`context` writes project task-context JSON. `orient` writes project orientation text by default and JSON with `--json`. Discovery preserves the existing project report schemas.

`--explain-scope` writes the winning kind, binding directory, records or workspace identity, and declaration origins to stderr. Reports remain on stdout. Discovery failures produce an error on stderr without a successful report.

Default project reader limits are 100 inspected files and 1,048,576 source bytes. `--max-files` and `--max-bytes` accept positive integers. Readers report omitted sources and partial results when limits prevent a complete read. Workspace navigation does not read project inventories.

| Exit status | Meaning |
| --- | --- |
| `0` | Complete reader result, successful workspace enumeration, or successful setup. |
| `1` | Partial project reader report. |
| `2` | Invalid invocation, discovery failure, or operation failure. |

Workspace JSON is a separate version-1 report with `kind: "workspace-navigation"`. It contains `complete`, the workspace `id` and `title`, `workStatus: "unevaluated"`, authored-order `members`, and `diagnostics`.

Each member contains `key`, optional `title` and `directory`, `records`, `availability`, and `selectionArgs`. Availability is `available`, `unavailable`, or `unknown`, based only on shallow records-directory access. It does not validate manifests, check checkout or source-root access, or evaluate readiness. Enumeration returns `0` even when members are unavailable. A failure to enumerate the selected declaration returns `2`.

`selectionArgs` is the complete process argument array, starting with `"ctx", "orient", "--bundle"`. Each member root adds a `"--allow-source", PATH` pair. Paths are absolute and can retain components such as `.context/..`. Text reports render the same array with POSIX shell quoting.

No-scope errors identify the searched ancestry and suggest setup or direct `--bundle` access. Configuration and selection errors identify relevant files, fields, authored values, resolved targets, and corrective actions where applicable. A complete context result establishes source completeness, not work readiness or accepted completion.
