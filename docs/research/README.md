# Research

- [Software factory research for Waymark, 22 October 2026](software-factory-2026-10-22.md)
  - [Pinned source records](software-factory-2026-10-22.sources.json)
- [Software factory evals and hooks research for Waymark, 29 October 2026](factory-evals-hooks-2026-10-29.md)
  - [Pinned source records](factory-evals-hooks-2026-10-29.sources.json)
- [Loop Engineering research for Waymark, 29 October 2026](loop-engineering-2026-10-29.md)
  - [Pinned source records](loop-engineering-2026-10-29.sources.json)

Waymark is the factory: it coordinates project work, coding agents, verification, evidence, and acceptance across repositories. The software factory research compares possible engines and execution infrastructure; it does not select a dependency or designate an application target. The evals and hooks note reads Matt Pocock's public factory and popular hook and eval projects. It finds three gaps in Waymark. Factory roles load the target repository's and the operator's hooks and MCP servers. Nothing checks that holdout tests fail before the build. Runs don't record which prompt version they were given. A local probe confirmed the first gap and found that only `--restricted` with an explicit tool list isolates a Claude Code run. The Loop Engineering note assesses a popular pattern library for scheduled maintenance loops around a repository. It does a different job from Waymark's. Waymark already enforces most of its controls in code, and there is nothing to adopt. One idea is kept on file: stop a repair early when it fails the same way again.

The comparison records source observations from its research date. Current platform implementation and acceptance belong in their own project records.
