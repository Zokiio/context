# Project discovery recommendation

This proposal consolidates the design interview and the final three-agent debate. The user approved this package, including unified path discovery and the new direct `--bundle` selector. No discovery implementation has begun.

## Everyday use and explicit selection

Run `ctx orient` from a project or any descendant directory. Discover scope from that directory without changing the shell directory. Apply the same scope resolver across commands.

Recommended selector contract:

- `--project PATH` runs project discovery from PATH, just as invocation from that directory would. `--project @alias` selects a personal project registration.
- `--workspace PATH` runs discovery for workspace scope from PATH, even if a project is also present. `--workspace @alias` selects a personal workspace registration.
- Add `--bundle PATH` for a direct record-directory input that bypasses local and personal discovery configuration. It uses explicitly supplied allowed source directories.
- These selectors are mutually exclusive. Relative command-line paths resolve from invocation cwd.
- A workspace cannot satisfy a command requiring one project. Report its members and selection guidance rather than choosing a member automatically.

This deliberately changes the existing `--project` contract. Existing calls targeting a valid project bundle continue to discover its marker, subject to explicit conflict checks. Callers reading a directory without a project marker must use `--bundle` for the old direct-directory behavior. Failed discovery never falls back to treating the input as an arbitrary record directory.

All three agents ranked consistent discovery with a direct bundle escape above preserving bundle-only `--project`. They rejected an exact-directory-only compromise because `/repo` and `/repo/src` would behave differently.

## Configuration and search

Use Markdown with YAML frontmatter in local `.context/config.md` and personal `~/.context/config.md`. Configuration filesystem paths resolve from the directory containing the file. This does not change document-link semantics.

For the first version, a local configuration binds the directory containing `.context`. Nested scope uses a nested configuration. Personal registrations can bind arbitrary directories. Workspace membership references do not independently create directory bindings.

Recognize an enclosing `project.md` with Project metadata as a bundle marker. Do not recursively search for records or assume `.scratch/records` is universal. Inspect ancestor directories up to the filesystem root, crossing Git and mount boundaries. Stop when broader scopes cannot affect selection. Personal settings are loaded once and do not implicitly bind the entire home directory.

Keep the accepted precedence rules: the most specific mapping wins; same-directory conflicts error; project wins over workspace at the same directory for implicit selection; a broken selected mapping never falls back to another project. Explicit workspace selection considers workspace candidates.

Coalesce duplicate mappings only when their effective record location and allowed document roots agree. Do not silently union document permissions. Malformed or unreadable configuration that could change selection produces an error. Broader local configuration that cannot win need not be read. An unparsable personal registry blocks discovery because its mappings cannot be determined. Direct `--bundle` remains available to bypass discovery.

## Paths, checkouts, and identity

Match directory scopes using canonical filesystem paths and search their physical ancestors. Keep the user's original path spelling in diagnostics. Do not search both a symlink's apparent ancestors and its target's ancestors. Resolve authored relative values from the encountered configuration directory before canonicalizing their targets.

Project identity stays in the authoritative bundle. Two checkouts with the same project ID remain distinct locations. Do not select one by recency or merge their records.

Give workspace declarations a stable workspace ID, separate from display names and file locations. Matching IDs do not authorize merging different declarations. Conflicting membership or settings must be reported with both origins.

Personal aliases are unique within their selector kind. An alias resolves without using cwd to break ties. Moving a shared checkout preserves internal relative paths. Stale personal registrations require explicit repair; discovery does not search the disk or rewrite registrations automatically.

## Setup and explanation

Provide an explicit connection workflow for existing records, separate from creating new records. It accepts paths from the caller's cwd, resolves them, and writes the correct values. Generated shared configuration prefers relative paths; personal registrations prefer absolute paths. Show the selected directory, records, allowed source directories, and destination configuration before writing. Agent use must support explicit noninteractive inputs.

Keep ordinary orientation output concise. Provide an optional resolution explanation for successful commands. Errors include the declaring configuration and field, authored path, resolved destination, and a corrective next step.

The agents propose `ctx setup` as the connection entry point, defaulting its binding directory to cwd and offering shared or personal storage. Exact flags belong in the specification. Repeating identical setup is idempotent. Conflicting existing declarations require an explicit update rather than silent replacement. Repair can initially mean editing the identified configuration; a dedicated repair command is unnecessary.

When no scope is found, return an error with the starting directory, searched locations, and connection guidance. Never launch setup from `orient`.

## Workspace output

The first version lists members and how to select them. Label work status as unevaluated. Use a distinct structured result kind for workspace navigation so agents cannot confuse membership with project readiness.

A complete member listing can succeed while marking individual record locations unavailable. Failure to enumerate the workspace itself is an error. Selecting an unavailable project fails. A missing code checkout alone does not prevent reading records that remain available elsewhere.

Computed cross-project readiness, automatic registration repair, and record creation are outside this discovery increment.

## Approval and specification

The user approved this behavior package. The main compatibility trade-off is the `--project` migration and additional `--bundle` selector. The [specification](spec.md) defines configuration fields, setup command syntax, output behavior, and verification requirements under these rules.

The [interview](discovery.md) records the earlier approvals and rationale.
