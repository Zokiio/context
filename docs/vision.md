# Product vision

A local-first, Git-native project and context management system, built around a reusable Go core and delivered first as a CLI. Jira and GitHub connections are optional. Future TUI and web interfaces use the same application services.

This document records the intended product and bootstrap plan. The product is not implemented yet. Its name, work-item schema, package layout, and interface libraries remain open.

## Problem and responsibility

Agent sessions lose project context, and their work can diverge from agreed goals and architectural boundaries. Humans compensate by repeating decisions, reconstructing project state, and manually maintaining status reports. The product makes project understanding durable and actionable so humans can direct several agents without reconstructing that understanding for each session.

The product first owns project records, context assembly, and queries about ready work, blockers, and inconsistencies. Agent runners can use those operations. Launching agents, assigning tasks to them, and managing their execution remain a separate, later capability.

The first user is the project's author. A possible next pilot is a small team working in a smaller monorepo within an enterprise. The solo workflow comes first; the pilot can test shared conventions and project boundaries in an existing repository.

Workflows share a small foundation: explicit intent, linked work, acceptance criteria, and completion evidence. Lifecycle stages and additional requirements are configurable. This describes the intended workflow foundation; the first milestone below remains read-only context assembly.

When implementation conflicts with agreed goals or architectural boundaries, an agent records the discovery and proposes a revision. A human decides whether to change those commitments before the agent proceeds on the revised basis. This keeps disagreement visible instead of allowing an agent to rewrite the commitments to match its implementation.

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

After the local reader, add read-only Jira retrieval as an early follow-up for the first user's pilot. Retrieve parent relationships and expose source freshness. Propose ticket edits locally until write-back is explicitly supported.

In record-owned mode, the connector publishes designated fields from the project record. In tracker-owned mode, local copies of designated fields are identified as snapshots, and edits go through the connector. Freshness remains visible. A remote completed state does not override failed local readiness checks.

Projects can configure pickup triggers using tracker labels, keywords, assignment, or other supported signals. Rules support all-condition and any-condition combinations, and selection reports which rule matched. A matching trigger makes a work item a candidate for pickup; it does not establish readiness or permission to start.

Readiness requires explicit acceptance criteria, available required context, satisfied dependencies, and no unresolved decision marked as blocking. Report unmet conditions separately from the matching pickup trigger.

Pickup triggers are entry conditions. If a trigger stops matching during active work, flag the change for review. Pausing or cancelling active work requires an explicit signal.

Projects configure how to handle work that matches a pickup trigger but is not ready: warn only, or allow an agent to investigate and propose missing information. Preparation does not authorize implementation. Implementation waits until readiness conditions are satisfied, and proposed changes to agreed goals or architectural boundaries still require a human decision.

Default to warn only. Projects can enable preparation assistance, and individual work items can override the project setting.

## Work claims

Track one active implementation claim per work item. Agent runners acquire the claim before starting implementation; supporting reviewers and researchers can work under that claim. Keep the claim separate from the human assignee in an external tracker so agent pickup preserves human responsibility for the ticket.

Claiming work is a coordination operation exposed to agent runners. It does not require the product to launch agents or manage their execution, and it is outside the first read-only milestone.

Before implementation, the agent refreshes context, rechecks readiness, and acquires the implementation claim. It continues within the agreed scope and surfaces discoveries that require changing goals or architectural boundaries.

Initially guarantee exclusive claims among agents on one machine. Cross-machine exclusivity requires a shared claim authority; synchronizing Git checkouts alone cannot guarantee it.

Mark a claim stale after missed check-ins. Initially require explicit takeover, preserving the previous handoff and checking unfinished changes before continuing. A stale claim does not prove that the original agent has stopped.

## Verification and development deployment

Associate verification evidence with acceptance criteria and the code revision tested. Distinguish local checks from tests in a development environment, and show untested criteria explicitly. Passing local tests does not establish successful configuration delivery to a device.

Project policy defines whether development deployment is preauthorized or requires a human decision. The workflow skill checks that policy and records the target environment and deployed revision. Where deployment restrictions require enforcement, use deployment tooling or access controls; a skill instruction alone is insufficient.

## Review, merge, and completion

Prepare a review summary linking the intended outcome, code changes, acceptance evidence, unresolved concerns, and accepted design changes. Keep automated review findings distinct from a teammate's approval.

Preserve evidence and approvals against their original revisions. When code changes, mark them as needing reassessment for the new revision. Project policy determines which checks and approvals must be renewed.

Default to the user merging after required checks and teammate approval. Merge authorization is configurable. After merge, record the merged revision and evaluate completion against the subtask's acceptance criteria. Completing the subtask does not complete its parent work item.

## Bootstrap workflow

### Agent execution model

Use workflow skills inside an existing coding-agent environment, backed by the shared CLI. The agent environment runs the model and provides tools for code inspection, edits, and tests. Skills guide grooming, implementation, and handoffs. The Go core owns context assembly, record validation, readiness, and later safe writes and claims.

Skills initially read and update repository files directly. As CLI operations become available, skills use them for consistent selection, validation, and writes. Add optional hooks for specific checks at supported points in the agent environment. Defer a custom agent runtime until the product needs to launch, pause, or directly supervise agents.

Skill instructions guide behavior but cannot guarantee compliance. Managed CLI operations can reject invalid record changes or conflicting claims; they cannot prevent application-code edits outside those operations. Hooks cover only the execution paths they intercept and cannot establish that an agent understood an architectural decision. Initially, waiting for readiness is a workflow rule backed by checked operations. Strict enforcement would require control over the relevant execution path.

### Adoption stages

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

## After the first milestone

The next milestone is session orientation: project goals, current commitments, ready work, and unresolved decisions. It helps a fresh agent identify suitable work before requesting detailed task context.

Alongside read-only Jira retrieval, extend task context to include requirements and acceptance criteria from the work item's parent chain, identifying their sources. Include sibling work only through explicit relevant relationships. Parent-chain selection extends the first reader's blocker and document-link rules; its local representation remains to be specified. Until then, link parent requirements explicitly as context for the local reader.

Parent acceptance criteria provide context; they do not automatically become the subtask's acceptance criteria. Each subtask defines the criteria for its own contribution. Completing a backend subtask does not establish that the entire device feature works.

Grooming produces a reviewable proposal with affected components, investigation findings, open questions, and suggested ticket splits. Unverified design choices remain proposals. The user decides which questions to take to colleagues and which ticket changes to accept.

Grooming also proposes missing context links with reasons. Accepted links become part of the project record for subsequent sessions. Context completeness continues to mean that all selected sources are present, not that every relevant source has been identified.

Continue independent investigation while questions remain unanswered. Record which decisions depend on each answer, and pause only work that would require guessing. Prepare questions for colleagues with relevant findings, options, and what the answer unblocks. The user chooses whether and where to send them, then brings back answers to record with their sources.

Propose ticket splits around independently verifiable outcomes or separate ownership and dependencies. Keep tightly coupled changes together. Each proposed ticket identifies its outcome, acceptance criteria, and blockers.

When requirements change during implementation, flag the older requirement revision used by the active work and reassess the affected work. Retain that revision so the agent can explain the change and propose an adjustment.

Support interrupted work with a durable handoff that records completed work, unfinished work, relevant code revisions, verification results, and unresolved questions. A replacement agent checks current state before relying on the handoff.

Agents save progress after meaningful decisions, completed steps, and verification results so recovery does not depend on a session ending normally. Capture enough to resume the work without preserving the whole conversation.

During discovery, agents propose focused record updates as conclusions emerge. Confirmed goals and decisions become authoritative; unresolved ideas remain discovery material. Retain references to source conversations or documents when available.

Begin inconsistency detection with deterministic checks for broken links, missing required information, and absent or outdated evidence. Later agent-assisted reviews can identify contradictions between code and prose. Present those findings separately from mechanically verified failures.

## Broader acceptance scenario

### First user's working scenario

A product owner assigns work to enable Wi-Fi on a device. The user owns a backend subtask for delivering configuration to that device. The subtask may lack enough detail to implement directly. Grooming can require questions to colleagues and code investigation to establish the scope, including possible changes across services, APIs, and infrastructure such as an event queue. The user may split the work into smaller tickets.

Implementation includes code changes and tests, possibly deployment to a live development environment for testing, followed by a teammate's review before merging to main. This scenario describes the user's workflow; it does not add Jira integration, deployment, or agent execution to the first read-only milestone.

### Project arrangements

Create a project inside an application repository and manage a complete task natively. Register that project in a workspace alongside another project. Continue using the same records without copying or migrating them. Attach an external issue to one project while the other continues to work independently.

This scenario tests the intended relationship between projects, record locations, workspaces, and optional integrations.
