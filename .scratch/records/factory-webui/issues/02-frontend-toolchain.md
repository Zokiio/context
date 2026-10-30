---
type: WorkItem
id: 72dcdab5-5ae0-4614-927f-edf631ab6e9a
title: Build the browser workspace from components
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Web workspace rework](../../../factory-webui/spec.md)

## Context

- [Factory HTTP and browser access](../../../../docs/factory-http.md)
- [Choose the browser workspace stack](../decisions/01-frontend-stack.md)

## Blocked by

None.

## Blocked by decisions

- [Choose the browser workspace stack](../decisions/01-frontend-stack.md)

## Acceptance criteria

- A build step produces the workspace assets that the binary embeds. `go build ./...` and the Go tests pass from a clean checkout without Node installed, by the means the stack decision chose, and a check fails when the committed output is stale.
- Sign-in, the top bar and the project switcher render from the new stack with the Gups tokens and the self-hosted fonts, in light and dark, under the existing Content Security Policy and same-origin rules, with no inline scripts or styles.
- The legacy workspace is served under `/legacy/` and the new shell links to it for screens not yet ported. Both share the token in session storage.
- A browser check against the real service signs in with a token, reaches the new shell and follows the link to a legacy screen.
- The contributor documentation names the toolchain, how to run the build and how to run the browser check.
