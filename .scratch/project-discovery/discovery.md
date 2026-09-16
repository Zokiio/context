# Discover the project from the invocation directory

## User intent

The user wants `ctx` to use the project at the invocation location by default. An explicit `--project` overrides that default. Repeating the bundle path for routine commands is cumbersome.

This record preserves the grill-with-docs design interview. The user approved the final package; the [specification](spec.md) owns the implementation contract. Earlier proposals below are historical where superseded by that contract.

## Existing contract

The CLI currently requires `--project` for both `context` and `orient`. The argument identifies the record bundle directory. Relative paths resolve from the caller's working directory. Linked documents outside the bundle require explicit `--allow-source` directories.

The session-orientation specification explicitly deferred automatic project discovery. This request proposes a follow-up to that scope. Application operations can continue receiving resolved project scope even when the CLI discovers it.

## Questions explored during the interview

- Which markers discovery checks in the current directory and its ancestors, and where searching stops.
- Whether an explicit `--project` accepts a containing project directory as well as a bundle directory.
- Behavior when discovery finds no project.
- Which authored marker or conventional bundle location establishes the selected project.
- How discovery handles multiple candidates, filesystem boundaries, and symlinks.
- How the default project selection interacts with authorized document directories.

## Confirmed interaction requirements

The user clarified that location selection applies to `ctx` across subcommands, including `ctx orient`. Invocation starts from the caller's current directory and looks for a project or workspace marker or identifier. The command must not navigate the user's shell elsewhere.

Mappings apply to descendant directories. The user accepted more-specific directory mappings overriding broader ones. Inspecting enclosing configuration does not change the shell's current directory. The specification now defines marker format and search boundaries.

## Repository findings

`project.md` with Project metadata is the existing bundle identity marker. `.scratch/records` is this repository's storage convention, not a product-wide discovery rule. There is no implemented workspace marker or repository-binding file. The discovery design must establish how the invocation location points to an existing bundle without copying records.

## Preferred direction after agent exploration

Three agents independently explored local-first, personal-registry-first, and hybrid discovery, then exchanged critiques. The user responded positively to the hybrid recommendation. That initial response established a preferred direction; later rounds confirmed the resolution rules.

The hybrid allows shared local configuration or personal registration under `~/.context` to connect a directory to existing project records. Users would not need to maintain both for each project. Project identity remains in the authoritative record bundle. Workspaces group projects without requiring their directories to be adjacent.

The agents recommended project orientation when a location selects a project and a labeled workspace overview when it selects a workspace. They also recommended failing same-scope conflicts and broken selected mappings, while disagreeing initially about narrower mappings overriding broader ones. The user subsequently confirmed the three resolution rules below. The specification now defines workspace member navigation.

## Confirmed resolution rules

The user accepted all three recommendations in the next interview round:

- A narrower directory mapping wins over a broader one, regardless of whether either is shared local configuration or personal registration. Conflicting project mappings at the same directory produce an error.
- When the same directory identifies both a project and a workspace, project scope takes precedence. Workspace selection remains available explicitly.
- A broken selected mapping produces an identifying error. Discovery does not fall back to an enclosing project's valid mapping.

The [scope resolution decision](../../docs/adr/0004-directory-specific-project-selection.md) preserves the trade-off.

## Final agent debate

The user asked the three agents to debate the remaining questions together, then present final conclusions for a single decision. Their [consolidated recommendation](recommendation.md) covers selectors, remaining search boundaries, invalid configuration, checkout identity, symlinks, workspace output, and setup. The user approved it, including the deliberate change to the existing `--project` contract and the direct `--bundle` selector. No implementation has begun for this change.

## Record locations, document roots, and selectors

The user agreed with the next round's recommendations:

- Configuration points to existing records; projects need not move records into a fixed folder.
- Saved project configuration supplies allowed document directories. Explicit `--allow-source` arguments add directories for that invocation; workspace membership alone does not grant document access.
- Explicit project selection supports paths and personal aliases with distinguishable syntax. `--project @backend` was the proposed alias spelling, contrasted with `--project /work/backend`.

The user then supplied a screenshot of a separate project-management repository's conventions index for comparison. Its visible document metadata and links do not establish how that system discovers projects, maps checkouts to records, or resolves configuration. Those mechanisms cannot be inferred from the screenshot alone.

## Workspace and setup direction

After a second three-agent debate, the user agreed with the recommendations:

- Support personal and shared workspace declarations and many-to-many project membership. Do not silently merge workspaces by matching names or directories.
- Keep `orient` read-only. When discovery finds no scope, explain where it looked and provide setup guidance. Explicit setup may guide the user; registration of existing records is separate from creation of new records.
- Begin workspace support with lightweight member navigation, explicitly stating that work status is unevaluated. A computed cross-project overview remains the desired later experience and requires its own evaluation-budget and partial-result design.
- Report unavailable member record locations. Missing code checkouts do not independently prevent orientation when the required records remain available elsewhere.

Membership navigation must not claim to provide project readiness. Its output and exit behavior need to reflect that narrower promise.

## Configuration format constraint

The user rejected TOML. Use the project's existing formats: YAML, JSON, JSONL, Markdown, or Markdown with frontmatter. Do not introduce another serialization format.

The proposed `.context/config.toml` and `~/.context/config.toml` paths are withdrawn. The user subsequently chose Markdown with YAML frontmatter and approved the search and selector rules recorded below.

## OKF alignment

The user affirmed that the format should fit OKF. OKF v0.2, as pinned by this repository, defines concept documents as Markdown with YAML frontmatter, including a required `type`. Plain YAML alone is not an OKF concept document.

Use Markdown with YAML frontmatter as the configuration direction. The proposed filenames therefore become `.context/config.md` and `~/.context/config.md`, rather than `config.yaml`. The specification defines the application-specific ContextConfig profile; OKF itself does not define CLI discovery or configuration precedence.

Reference: https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md#4-concept-documents

## Git repository boundaries

The user confirmed that ancestor discovery crosses Git repository boundaries. A workspace mapping at `/work/` can apply inside `/work/backend/`, even when backend is a Git repository, unless a more-specific project mapping takes precedence. A Git root alone neither selects scope nor stops discovery. The specification defines the remaining stopping rules.

## Explicit workspace selection

The user approved `--workspace` with either a directory path or a personal alias, for example `ctx orient --workspace /work/team` or `ctx orient --workspace @work`. These are proposed commands, not implemented behavior. Supplying both `--project` and `--workspace` produces an error because they select different scopes.

## Relative-path debate

At the user's request, three agents debated developer convenience, effective agent use, and portability. The user approved their converged recommendation:

- Resolve relative filesystem paths in configuration from the directory containing that configuration file, consistently for shared and personal configuration. Resolve command-line paths from invocation cwd.
- Let explicit setup resolve user input from cwd, then write the appropriate paths. Prefer portable relative paths in generated shared configuration and absolute paths in generated personal registrations. These are authoring defaults, not different resolution rules.
- Offer an explanation of the selected scope, declaring configuration, and resolved locations. Identify the source field and resolved target in path errors.

The developer-focused agent initially preferred paths relative to the directory containing `.context`, allowing `.scratch/records` instead of `../.scratch/records`. After peer critique, the agent favored consistent file-relative semantics with setup handling conversion. Routine use should remain `ctx orient` without path arithmetic.

Filesystem configuration fields must remain distinct from record links. Existing bundle-relative link semantics do not define filesystem configuration paths. Moving a whole clone preserves its internal relative paths; moving external records or copying configuration alone may require repair. The specification defines setup syntax and configuration fields.

## Approved implementation handoff

The user accepted the consolidated recommendation. The [specification](spec.md) translates it into an implementation contract and links the dependency-ordered tickets. This approval closes the design interview; implementation has not started.
