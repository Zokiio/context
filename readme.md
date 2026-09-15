# Project and context management

A project in design for keeping project records in local files tracked by Git and assembling task context for coding agents.

Agent sessions lose context. Developers then repeat requirements, reconstruct decisions, and gather the same documents for each new session. This project aims to preserve that information and select the sources an agent needs for a work item.

## Current status

The product is not implemented yet. This repository contains the product vision, domain vocabulary, architectural decisions, and the first feature specification. The product name remains open.

The planned implementation uses a reusable Go core with a CLI as its first interface. The [product vision](docs/vision.md) describes the scope and adoption stages.

## First milestone

The first milestone is a read-only context reader. Given a project directory and a ticket path, the reader assembles the ticket, its blockers recursively, and each selected ticket's linked specifications and context documents.

The specified result contains whole source files, their paths, and the reasons for inclusion in a versioned JSON object. Sources reflect current files on disk, including uncommitted edits. If selected sources are unavailable or exceed configured limits, the reader returns the available context marked incomplete, with diagnostics and a nonzero exit status.

The [context reader specification](.scratch/context-reader/spec.md) defines selection rules, limits, and acceptance criteria.

## Project records

A project has its own identity, independent of where its records live. A project can span repositories, and a workspace groups projects without copying their authoritative records.

The recorded design uses one Open Knowledge Format (OKF) bundle per project. Markdown links in named sections express relationships between work records. The architectural decisions explain these choices:

- [One OKF bundle per project](docs/adr/0001-one-okf-bundle-per-project.md)
- [Work relationships in Markdown sections](docs/adr/0002-work-relationships-in-markdown-sections.md)

The [domain glossary](CONTEXT.md) defines project, workspace, work item, task context, and related terms.

## Repository conventions

Specs and tickets for developing this product live under `.scratch/` and are tracked in Git. These bootstrap records follow the [local issue tracker conventions](docs/agents/issue-tracker.md).

[Repository guidance](AGENTS.md) lists the instructions for agents working in this repository.
