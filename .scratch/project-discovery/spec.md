# Discover project and workspace scope

Status: ready-for-agent

## Outcome

Run `ctx orient` or `ctx context --ticket PATH` inside a configured project without repeating `--project`. Selection starts from the caller's directory and never changes the shell directory. The [approved recommendation](recommendation.md) records the design rationale. This specification owns the implementation contract below.

## Scope and compatibility

Use one discovery operation for both commands. Keep application readers dependent on resolved record and allowed-source directories, without embedding discovery in document traversal.

Accept at most one of `--project`, `--workspace`, and `--bundle`, rejecting empty or repeated scalar selectors. With no selector, discover from cwd. `--project PATH` discovers project scope from that path, including ancestors. `--workspace PATH` searches workspace scope from that path, ignoring project candidates. Selector paths must name directories. A leading `@` identifies a personal alias; use `./@name` for a literal directory beginning with `@`.

`--bundle PATH` passes a records directory directly to the existing operation, bypassing all discovery configuration and using only explicit `--allow-source` values. Do not require a project manifest for direct task-context assembly. Direct orientation retains its existing manifest validation and partial-report behavior.

There is no raw-directory fallback from failed discovery. Existing `--project` callers reading directories without a discoverable Project marker migrate to `--bundle`. Ticket paths remain relative to the selected record bundle. Existing reader selection, completeness, budgets, and source authorization rules remain unchanged.

A project-only command such as `context` errors when only workspace scope is available and provides project-selection guidance. It never picks the sole member automatically. Bare `ctx` continues to show command guidance; this increment changes scope defaults, not the root command's action.

## Configuration profile

Use UTF-8 Markdown with YAML frontmatter, `type: ContextConfig`, and `version: 1`. Preserve the Markdown body and unknown metadata when updating configuration. Reject duplicate YAML keys, wrong types for known fields, unsupported versions, empty required strings, and duplicate entry keys. Do not interpret settings as executable expressions or interpolate environment variables. Absolute filesystem paths remain absolute; relative filesystem paths resolve from the encountered configuration file's directory. No implicit tilde expansion in saved values.

The local file is `<directory>/.context/config.md`; the personal registry is `~/.context/config.md`, with home supplied through an injectable environment dependency for tests. These settings do not constitute an implicit home-directory scope.

The personal registry path is reserved. Ancestor traversal never interprets it again as a shared marker. Shared setup targeting home fails with guidance to use a personal registration; the two profiles cannot share that file.

Shared configuration supports an optional `project` mapping and an optional `workspace` declaration. Both apply only to the directory containing `.context`. At least one is required. `project` has `records` and optional `allowSources`, an array defaulting to empty. For example:

```markdown
---
type: ContextConfig
version: 1
project:
  records: ../.scratch/records
  allowSources:
    - ../docs
---

Project records and supporting documentation for this checkout.
```

Personal configuration uses `projects` and `workspaces` arrays, defaulting to empty. Each entry has a unique `key` within its array and optional `alias`, unique within that selector kind. A project entry has `directory`, `records`, and optional `allowSources`. A workspace entry has an optional `directory` plus the workspace declaration fields below. Entries without a directory are selectable by alias but create no cwd binding. Project registrations require a directory when authored, but later unavailability of the checkout does not invalidate alias selection if its records remain available. Personal and local profile fields must not be mixed in one file.

A workspace declaration has a stable nonempty `id`, a nonempty `title`, and `members`, an array that can be empty. Each member has a workspace-local unique `key`, optional `title`, required `records`, optional checkout `directory`, and optional `allowSources`. Member paths resolve from the declaring configuration. Membership creates neither cwd bindings nor inherited document access. Member roots apply only when that member is explicitly selected through its declaration, not to other members or arbitrary paths.

For v1, member navigation shows a copyable `--bundle` command with that member's explicit `--allow-source` flags. Quote filesystem values safely for the documented shell; structured output provides argument arrays so agents need not parse shell text. A direct record location with a Project marker remains discoverable through `--project`, but a copied direct command avoids accidentally picking another mapping or losing member-specific roots.

Stable project identity stays in each bundle's `project.md`. A workspace ID establishes identity independently of the config location. Matching names or IDs never authorize merging different declarations or record stores. Canonically identical effective declarations may coalesce; differing selected declarations conflict with both origins reported. Different checkout locations containing the same project ID remain separate members.

## Discovery algorithm

1. Resolve invocation or selector paths from cwd. Canonicalize the start directory and use its physical ancestor chain. Keep entered spelling for diagnostics. Do not also search symlink lexical ancestors.
2. For discovery, load and validate the personal registry structure once. Missing personal configuration is normal. A malformed or unreadable registry fails discovery because it can hide a narrower mapping. Do not eagerly require every registered target to exist.
3. At each relevant directory depth, compare personal bindings, the local configuration at that directory, and an enclosing bundle's reserved `project.md` marker. A valid marker requires Project type, ID, and title. A present malformed reserved marker is an invalid candidate when project scope could affect selection, never a reason to fall through silently. Explicit workspace selection ignores project markers, including malformed ones.
4. The nearest applicable depth wins across storage locations. At equal depth, project wins for implicit selection. Explicit selectors filter by requested kind. A nearer workspace prevents implicit fallback to a broader project. A project-only command then reports that project selection is required.
5. Conflicting candidates of the selected kind at equal depth error. Duplicate project mappings coalesce only when canonical records and effective allowed-source sets agree. A bare bundle marker supplies identity and location; an agreeing mapping to that same bundle may supply roots. The marker does not itself create a conflicting empty-root configuration. Two authored mappings with differing roots still conflict.
6. Stop reading broader local files once they cannot win. Search through Git, home, and mount boundaries to filesystem root when necessary. Do not recurse downward looking for `.scratch/records` or any other conventional folder.
7. Validate selected targets and authorized roots. Missing or unreadable selected locations fail without fallback. Unrelated missing target locations do not block selection. A binding whose scope cannot be resolved and might contain the start directory must produce a diagnostic rather than disappear silently.

Personal aliases select their concrete saved entry independently of cwd. Unknown or duplicate aliases fail. Alias project selection needs the record store, not an available code checkout. Relative config values are resolved before canonicalizing their targets; a symlinked config file does not rebase values to its target file's directory. Existing reader containment checks continue to enforce authorized source boundaries.

Add explicit `--allow-source` roots to the selected configured roots for that invocation only. Do not persist them or union conflicting configuration declarations.

## Output and failures

Preserve existing project report formats and exit statuses: 0 for complete results, 1 for partial reader reports, 2 for invocation, discovery, or operation failure. Discovery failures go to stderr without producing a misleading successful report. Include the start location, relevant configuration file and field, authored value, resolved target, and a corrective action where applicable. A no-scope error explains searched ancestors and suggests setup or `--bundle`.

Add `--explain-scope` to the read commands. It writes the winning kind, binding directory, record location or workspace ID, and configuration origins to stderr. Keep stdout machine-readable. Normal project JSON schemas need not change to add discovery metadata.

Workspace `orient` has a separate version-1 JSON result with `kind: workspace-navigation`, `complete`, `workspace` identity/title, `workStatus: unevaluated`, `members`, and `diagnostics`. Members retain authored order and contain key, optional display title and checkout path, records path, availability, and selection argument array. Availability is only a shallow directory-access check, not manifest or readiness validation. Use `available`, `unavailable`, or `unknown` with explanatory diagnostics. Do not scan work inventories. Return 0 when membership enumeration succeeds, even if a member is unavailable. Failure to enumerate the selected declaration returns 2. Empty membership is valid. Human output conveys the same facts and explicitly states that work status was not evaluated.

## Connect existing records

Provide `ctx setup` to register an existing Project bundle. Guided use prompts only on an interactive terminal. Fully specified flags require no prompts:

- `--records PATH` supplies existing records.
- `--directory PATH` supplies the binding directory, defaulting to cwd.
- `--personal` writes a personal registration instead of shared configuration.
- `--alias NAME` adds a personal project alias and requires `--personal`.
- Repeated `--allow-source PATH` saves authorized supporting-document roots.
- `--dry-run` prints the exact proposed change and resolved locations without writing.
- `--replace` explicitly permits updating a conflicting existing project binding. It never overwrites unrelated entries.

Resolve inputs from invocation cwd, then generate relative paths in shared configuration and absolute paths in personal registration. Validate the bundle manifest, directory, and allowed roots before writing. Identical setup is a no-op. Do not replace malformed configuration, lose its prose/unknown metadata, or silently overwrite a conflicting declaration. Use atomic replacement and detect changes between reading and writing. Preserve existing file permissions and report failures without leaving truncated files.

Before any write, print the binding directory, records location, allowed roots, destination file, and whether the operation creates or updates a binding. Fully specified noninteractive invocations require no additional confirmation. `--dry-run` additionally prints the proposed document and makes no changes.

Personal setup matches an existing entry by canonical binding directory. Ambiguous existing entries error. Generate a UUID key for a new entry and retain the key on replacement. Omitting `--alias` on replacement preserves the existing alias; a supplied alias must not collide with another entry. The records and supplied allowed-root list replace those fields explicitly under `--replace`; an omitted root list means empty. Report removals in the pre-write summary. Never retarget another entry merely to reuse its alias.

Validate the proposed binding against other effective declarations before writing. `--replace` authorizes updates only in the chosen destination file. If a conflicting declaration would remain in another file, fail without writing and identify the declaration requiring repair. Identical declarations elsewhere may coexist under the normal coalescing rule.

This setup command connects one project. Workspace declarations can be authored as documented Markdown in v1; a workspace creation wizard is outside this increment. Record creation, automatic relocation repair, and discovery-triggered configuration writes are excluded. Read commands never prompt or modify configuration.

## Verification

Use real temporary filesystems with injected home and cwd. Cover independent non-Git projects, nested project/workspace mappings, same-depth conflicts, registry errors, aliases, separate records, symlinks, moved clones, and missing checkouts. Test both CLI entry paths and direct application readers. Check read-only commands leave files unchanged and setup preserves unrelated configuration. Exercise the legacy manifestless bundle migration explicitly.

Run repository Go tests, race checks, vet, and build as appropriate to implemented slices. The final trial must use a second independent project and a workspace, with both human output and structured results. Record criterion-specific evidence under the normal acceptance workflow.

## Implementation sequence

- [Read discovery configuration](../records/project-discovery/issues/01-read-configuration.md)
- [Resolve scope from directories and aliases](../records/project-discovery/issues/02-resolve-scope.md)
- [Use discovery in reader commands](../records/project-discovery/issues/03-integrate-cli.md)
- [Show workspace member navigation](../records/project-discovery/issues/04-workspace-navigation.md)
- [Connect existing records through setup](../records/project-discovery/issues/05-connect-records.md)
- [Document migration and verify discovery end to end](../records/project-discovery/issues/06-document-and-trial.md)
