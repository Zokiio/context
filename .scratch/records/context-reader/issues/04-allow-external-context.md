---
type: WorkItem
id: 0981e1d0-7916-45cb-9639-e23ac004f6d7
title: "Allow authorized external context"
triage: ready-for-agent
execution: completed
---

# 04: Allow authorized external context

**What to build:** A caller can explicitly authorize additional directories for linked Spec and Context documents. Existing plain Markdown documentation can then accompany a ticket without moving into the project's OKF bundle.

## Scope

This slice adds scoped document access rather than configuration loading or cross-project blocker traversal. It can proceed independently of recursive-blocker work because it extends the shared document-reading operation.

## Acceptance criteria

- [x] Accept repeatable --allow-source arguments through urfave. Resolve relative allowed directories against the caller's working directory, including when it differs from the selected project.
- [x] Preserve each flag occurrence literally, including commas in directory names. Disable slice-value separator handling and do not supply authorized roots through environment or configuration fallbacks.
- [x] Load explicitly linked Spec and Context documents outside the bundle only when their resolved targets lie inside an authorized source directory. Plain Markdown without frontmatter is accepted and its full source text is preserved.
- [x] Keep document-link interpretation independent of native filesystem paths: relative links resolve from the containing document, and leading-slash links resolve from the selected bundle root. External documents are reached through ordinary relative links.
- [x] Choose the appropriate permitted root and perform constrained reads with os.Root. Test resolved symlink targets and paths crossing directory boundaries; do not rely on a string-prefix check or an unchecked file open.
- [x] An unauthorized, missing, or unreadable selected external document produces identifying diagnostics, incomplete context, and exit code 1 while preserving other available sources. Successful authorized inclusion retains source path, digest, and all distinct reasons.
- [x] Keep starting-ticket and blocker targets inside the bundle even when another directory is authorized for documents. Reuse the shared containment policy rather than adding an external-ticket exception.
- [x] Deduplicate a source reached through multiple allowed roots or symlink aliases by its resolved path. Do not follow ordinary links from the included external prose or fetch remote sources.
- [x] Application and CLI tests exercise multiple allowed roots, relative roots from an unrelated working directory, comma-containing paths, plain Markdown sources, authorized and unauthorized symlinks, and unchanged-source behavior.

## Blocked by

- [02: Include linked project documents](02-include-project-documents.md)

## Blocked by decisions

None

## Spec

- [Context reader specification](../../../context-reader/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)

## Comments

### 2026-09-16 implementation assignment

Agent `/root/ticket01` owns external source authorization, source-reader root selection, CLI flags, and external application and CLI tests. Ticket02 is complete. Ticket03 is being implemented concurrently against the unchanged source-reader interface.


### 2026-09-16 completion evidence

Implemented by `/root/ticket01` in the uncommitted working tree based on commit `0c74f92506a67242247a4e76a40785561a561bb3`, after ticket02 and concurrently with ticket03.

- Added `Request.AllowedSourceDirs` and repeatable `--allow-source`. The CLI resolves relative directories against the caller's working directory and disables urfave slice separator handling. No environment or configuration source supplies authorization.
- The shared reader resolves directories and source paths, selects a permitted `os.Root` using component-aware containment, and reads through that root. Starting tickets and blockers consider only the bundle root; document sources can use caller-authorized roots.
- Added application tests for multiple and overlapping roots, unchanged plain Markdown and matching digests, deduplication and reasons across aliases, bundle-relative leading-slash links, unauthorized neighboring directories and symlinks, missing/unreadable sources, and external ticket/blocker rejection. Missing files below a symlinked authorized directory retain the missing-source diagnostic.
- Added CLI testscript coverage for relative roots from an unrelated working directory, repeated flags, comma-containing names, no environment fallback, invalid root errors, and starting-ticket containment.
- Passed `go test ./...`, `go test -race ./...`, `go vet ./...`, and `git diff --check` locally with Go 1.27.1. These checks included concurrent ticket03 changes present at test time.

Invalid or empty authorized directories fail operation setup with status `2`. Selected unauthorized, missing, and unreadable documents retain existing diagnostic codes and return incomplete context with status `1`. Documentation additions were handed to the coordinating agent, who owns the root readme.

## Acceptance

- [Current acceptance](../acceptances/04-allow-external-context-migration-20260916.md)
