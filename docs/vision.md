# Product vision

A local-first, Git-native project and context management system, built around a reusable Go core and delivered first as a CLI. Jira and GitHub connections are optional. Future TUI and web interfaces use the same application services.

This document records the intended product and bootstrap plan. The product is not implemented yet. Its name, work-item schema, package layout, and interface libraries remain open.

## Independent configuration choices

| Concern | Intended support |
| --- | --- |
| Record location | Embedded in an application repository or stored in a separate repository |
| Project organization | One project, several projects in a monorepo, or projects spanning repositories |
| Issue tracking | Native work items, optional GitHub or Jira connections, and different choices per project |
| Interface | CLI first, then TUI, web, and API access to the same core |

These are configurations of one product. A tracker choice does not determine record location. Adding an interface does not require moving records.

## Record format direction

[Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md) is the preferred format for authored project records. The link pins the specification examined during design.

Work-state conventions remain open. The [bundle-boundary decision](adr/0001-one-okf-bundle-per-project.md) defines project identity and grouping. The [relationship representation](adr/0002-work-relationships-in-markdown-sections.md) defines links between work records.

Managed work records use OKF. The first reader also accepts explicitly linked plain Markdown context sources outside the OKF bundle.

## Product constraints

- A project can span repositories, and a repository can contribute to several projects through explicit path scopes.
- Workspaces locate and group project records. Registration references existing records without copying them. A workspace can contain both embedded and separately stored records.
- Projects and work items have stable identities that survive moves. Checkout and revision information identifies different versions of the same project.
- An embedded project works without an explicit workspace, external tracker, server, or LLM dependency.
- Application operations receive explicit resolved project scope. Noninteractive commands fail clearly when scope is ambiguous.
- The Go core owns context assembly, validation, readiness, and work operations. Interfaces handle input and output. Provider-specific types remain in integration adapters.
- Native work items support dependencies, acceptance criteria, specification and decision links, workflow state, and completion evidence.
- Workflow state and computed readiness are separate. Moving a ticket to a completed state does not establish that its evidence satisfies project policy.
- Completion evidence can include local test results, reviews, commits, manual acceptance, or external CI results. Evidence retains its origin and revision. A pull request is not mandatory.
- Issue tracking and code hosting are separate integration responsibilities. A project can use native work items or Jira while obtaining review evidence from GitHub.
- Context selection is scoped to the project and task. Session orientation and detailed task context are distinct views. Included records carry revisions and an explanation of their inclusion.
- Cross-project dependencies are explicit. A reference to another project does not imply importing all of its context.
- Shared standards are optional and versioned. A project can use local conventions without adopting an organizational process.
- Authored prose, managed metadata, and disposable caches have clear ownership. Removing a cache preserves project knowledge.
- All interfaces use the same persistence layer for writes. It checks expected revisions and coordinates local writers. Git merges across machines remain a separate concern.

## Integration progression

Start with native tracking and links to external issues. Add synchronization only with declared field ownership.

In record-owned mode, the connector publishes designated fields from the project record. In tracker-owned mode, local copies of designated fields are identified as snapshots, and edits go through the connector. Freshness remains visible. A remote completed state does not override failed local readiness checks.

## Bootstrap workflow

Use the installed Matt Pocock skills to plan, specify, implement, review, and hand off work. Keep persistent context in repository files. The conventions for authoritative specs and tickets live in [Issue tracker](agents/issue-tracker.md).

Adopt the CLI in stages:

1. Skills read and write the bootstrap records. Building and testing the Go project work without an installed copy of the product.
2. The CLI reads those records and assembles context. Skills continue to own edits while the read path is exercised.
3. Once validation, safe writes, revision checks, and recovery are tested, the CLI becomes the preferred way to update the same authoritative records. Keep a known-good binary available separately from the development build.

The bootstrap layout is temporary. A later importer can translate it when the product's record format is settled. Adoption must preserve one authoritative backlog.

## First milestone

Given an explicit project directory and ticket path, assemble context containing the ticket, its dependencies, and explicitly linked project documents. The project directory locates records and need not be a Git root.

The result includes the source text, file path, and reason for inclusion for each selected record.

Read records as they currently exist on disk, including uncommitted edits and new files.

The caller supplies allowed source directories for context files outside the OKF bundle.

A work item can contain its own requirements and acceptance criteria. Its Spec section is optional. A broken link in a present Spec section makes the result incomplete.

Follow blockers recursively and include each selected ticket's Spec and Context links. Include each file once. Other links in document prose remain references.

Report dependency cycles as warnings. A result with all selected sources present remains complete and returns success.

If a selected file is missing or unreadable, return the available context marked incomplete, with diagnostics and a nonzero exit status. Context completeness is separate from OKF document validity.

Limits on file count and total source bytes are configurable. Include whole files within the limits. Report omissions explicitly and return an incomplete result with a nonzero exit status.

Use explicit links and predictable selection rules. Keep the first milestone read-only. Include a second fixture project to expose assumptions that the current repository is always the project. Establish structured output and clear noninteractive errors early.

Jira synchronization, a TUI, a web interface, and autonomous agent scheduling are later work. They are not prerequisites for this milestone.

## Broader acceptance scenario

Create a project inside an application repository and manage a complete task natively. Register that project in a workspace alongside another project. Continue using the same records without copying or migrating them. Attach an external issue to one project while the other continues to work independently.

This scenario tests the intended relationship between projects, record locations, workspaces, and optional integrations.
