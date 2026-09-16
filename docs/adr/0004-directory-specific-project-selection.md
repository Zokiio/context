# Select projects by directory specificity

Implicit project selection uses the most specific applicable directory mapping, whether declared in shared local configuration or personal registration. Conflicting project mappings at the same directory fail; when that directory also identifies a workspace, project scope takes precedence and the workspace remains explicitly selectable. A broken selected mapping fails without falling back to an enclosing project, so stale configuration cannot silently redirect a command to different work.

This gives directory scope priority over configuration storage location. It permits a personal mapping for a nested project to override a broader shared mapping while preserving explicit conflicts and failures. The [discovery interview](../../.scratch/project-discovery/discovery.md) records the user's approval; the [specification](../../.scratch/project-discovery/spec.md) defines the approved configuration, search, and selector contract.

Explicit project and workspace paths use the same discovery rules as invocation from that directory. A separate `--bundle` selector provides direct record access without discovery. This intentionally migrates manifestless direct `--project` callers to `--bundle`, favoring consistent selection at any directory depth over preserving the old selector meaning.
