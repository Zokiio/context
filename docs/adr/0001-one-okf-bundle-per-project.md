# One OKF bundle per project

A project's authoritative record is one OKF bundle, and workspaces reference bundle locations. Bundle boundaries follow projects independently of Git repository boundaries. This keeps each project movable and lets several workspaces group it without copying its records.

Each bundle contains a root `project.md` with `type: Project`, `id`, and `title` in its YAML frontmatter. Each work-item record carries its own stable `id`. These domain IDs survive moves and renames and remain distinct from OKF's path-based concept IDs.
