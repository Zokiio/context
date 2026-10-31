---
type: Decision
id: 56fe4d04-cc15-479c-80d7-154afeb86f42
title: Choose the browser workspace stack
decisionState: open
---

# Choose the browser workspace stack

The workspace is one hand-written script and stylesheet with no build step. The [web workspace rework](../../../factory-webui/spec.md) rebuilds it from components, which needs a toolchain.

## Question

Which component library and build step does the browser workspace use, and does the repository commit the built output?

## What the answer unblocks

[Build the browser workspace from components](../issues/02-frontend-toolchain.md) and every screen ticket after it.

## Constraints

- `go build ./...` must work from a clean checkout. Either the built assets are committed, or a documented `go generate` step builds them and continuous integration checks that the committed copy is current.
- No runtime dependency: the binary serves the built files from its embedded assets.
- The Content Security Policy forbids inline scripts and styles, so the build must emit external files only.
- Project content, worker output and errors must stay text; the library must make escaping the default.
- The existing browser checks use Playwright with Node, so a Node toolchain adds no new requirement for contributors who run them.

## Proposed default

Preact with TypeScript, built by Vite into `internal/factoryhttp/assets/dist/`, run from `go generate ./internal/factoryhttp` as the initialization templates are, with the output committed and a test that fails when the committed output is stale. Svelte is the alternative if the person building it prefers it; the constraints above are what matter.
