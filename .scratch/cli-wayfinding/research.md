# What remains to build the local CLI

Researched 2026-09-17 against local source revision `bde51f370011d1598cfa662e126e3ed6e513b98e`, the supplied conversation, cached research, and first-party articles. This is research and proposed sequencing, not an approved implementation specification.

The user selected this destination: **a usable local CLI for directing and resuming agent work, with Jira as an early follow-up**.

The user also selected skill-authored records and handoffs, CLI assistance for identifying needed actions and resuming work, and human-directed pickup. Managed work-record writes and exclusive implementation claims are outside this milestone. The [wayfinder map](map.md) indexes the remaining decisions.

## Finding

Build on the existing readers. The next useful increment is a short explanation of work needing intervention, followed by a durable way to resume interrupted work with current context and explicit verification gaps. General record editing, implementation claims, and remote integration need separate decisions. A custom agent runtime is unnecessary for this destination. This recommendation combines the observed baseline below with the product's [bootstrap model](../../docs/vision.md#bootstrap-workflow).

The product already separates execution state, work readiness, context completeness, and recorded acceptance. Keep those distinctions. The missing product experience is helping a person or agent act on the facts without reconstructing them from a full inventory. [Domain glossary](../../CONTEXT.md), [reader reference](../../docs/readers.md), [existing workflow proposals](supporting/workflow-improvements.md).

## Evidence and its limits

- Current source and authored ADRs establish implemented behavior and accepted boundaries. Inspection covered CLI commands, readiness checks, setup preparation and writing, and reader documentation. No fresh build, test suite, or acceptance trial was run for this research.
- [WoW with A](chatgpt-conversation://6aab1283-e634-83ed-9d50-361973232ead) was read across both available pages. The user explicitly described the company's Jira/Scrum environment and later clarified that Confluence is optional. Assistant-generated lifecycle diagrams are proposals, not proof that the user approved every gate. Image content was not used as evidence; textual exchanges were sufficient for these findings.
- Cached repository studies (optional local history: `../../.cache/research/ideas-from-repositories.md`) supply earlier proposals and pointers, not current product guarantees. This research does not revalidate all six competing products. Relevant conclusions already have a durable home in [Help people direct and resume project work](supporting/workflow-improvements.md).
- Earlier product conversation (optional local history: `../../.cache/removethis/start02.md`) contains proposed layouts, a placeholder product name, and a CLI-library suggestion. Current source supersedes those details. The adjacent `start01.md` and `jira.md` files are empty.
- Cached build outputs and worktrees were treated as disposable history, not the authority for current capabilities. New research belongs outside `.cache` under the [tracker conventions](../../docs/agents/issue-tracker.md).

## What the external sources establish


### Anthropic's SDLC playbook

Anthropic proposes a versioned sequence of intent, design specification, implementation plan, build evidence, and review. People accept the intent and design before implementation. It distinguishes advisory skills from executable controls and recommends carrying policy versions and verification output into the record. It also discusses record ownership when existing trackers remain authoritative. These are vendor workflow recommendations, not proof that a prescribed file sequence improves every project. The article's proposed metrics do not establish measured results for this CLI. [The AI-Native SDLC playbook](https://claude.com/blog/the-ai-native-sdlc-playbook)

Implication: discuss whether this project needs an explicit decision accepting intent or design. Do not silently reinterpret the existing work-item Acceptance decision as pre-build design approval. Nor should the playbook's filenames become mandatory product entities without a domain decision.

### Tenex's project-record approach

Tenex describes its internal MetaHarness system as a canonical Markdown project record with structured validation, scoped machine writes, session orientation, shared skills, and derived views. Its CLI checks relationships and evidence; external writes have a preview and separate application step. The essay recommends starting with one real project and a small validator, and acknowledges process overhead at small scale. [Building an AI-Native SDLC](https://www.tenex.co/blog/building-an-ai-native-sdlc/), published August 5, 2026.

This is first-party testimony about Tenex's own workflow. The supplied essay provides no public CLI implementation or complete schema that we can evaluate or reuse. Its reliability claims are not guarantees for our design. Its folder names, checkpoint grammar, and command count need not become requirements here.

Implication: give users precise reasons for blocked or unknown work, preserve prose ownership when writes arrive, and validate a cold-start workflow before expanding the command set.

### Alignbase's context boundary

Alignbase treats shared instructions, skills, and memory as owned, versioned resources that can be delivered selectively across agent tools. Its portability discussion asks who owns context, which recipients need it, how it fits their context limits, and what records delivery. [Portable AI Agent Context](https://alignbase.ai/blogs/portable-ai-agent-context/), published July 16, 2026.

Its management-platform article separates context distribution from orchestration, observability, evaluation, and execution controls. It explicitly limits its current conversation evidence to client-reported information and says that a server response does not prove host injection or model consumption. This is a useful limit to retain in our own claims. [AI Agent Management Platform](https://alignbase.ai/blogs/ai-agent-management-platform/), published September 4, 2026.

Its operating-model article distinguishes approved Knowledge and Skills, editable Memory, and explicit Artifacts. Artifacts do not automatically acquire routing or instruction authority. This gives a concrete example of separating records that guide behavior from material that merely documents work. [AI Agent Operating Model for Enterprise Teams](https://alignbase.ai/blogs/ai-agent-operating-model/), published September 8, 2026.

These are vendor descriptions, not an independent evaluation of Alignbase. The supplied `/blog/` index and an attempted `/blogs/` index failed through the browser tool. Individual first-party articles were available through search. This is a targeted sample, not a complete audit of that blog or its product API.

### Check the executable boundary against product documentation

Claude Code's documentation says that `CLAUDE.md` shapes behavior without guaranteeing compliance; client settings provide technical enforcement. It also documents that agent memory is machine-local, so it cannot substitute for a shared project record. [How Claude remembers your project](https://code.claude.com/docs/en/memory)

The hooks reference defines specific lifecycle events, matchers, and decision controls. `SessionStart` can supply context; `PreToolUse` can block a matching tool call. Those are host integration points, not a reason to put host-specific schemas into the reusable project core. [Hooks reference](https://code.claude.com/docs/en/hooks)

Inference: a successful CLI context response proves what the CLI assembled. A host adapter must supply separate evidence about what it loaded. Neither record proves that the model understood or followed it. A CLI validation error can reject a managed operation; it cannot stop every possible file edit elsewhere on the machine.

## What is already built

| Capability | Observed implementation | Consequence for planning |
| --- | --- | --- |
| Scoped task context | `context` follows explicit Spec, Context, and Blocked by links, retains source text and digests, and reports incomplete collection. | Extend selection only for a demonstrated missing relationship. Do not rebuild context assembly. |
| Session orientation | `orient` reports goals, commitments, work, readiness, decisions, and provenance in text or JSON. | A concise intervention view can reuse these results. |
| Readiness and dependency acceptance | Checks cover criteria presence, required sources, dependencies, blocking decisions, and recorded prerequisite acceptance. | Nonempty criteria do not establish that the intent or specification is good enough. |
| Acceptance provenance | Requirement and evidence fingerprints, attribution, and historical tested revision are retained. | New verification views should explain evidence without replacing acceptance semantics. |
| Discovery and setup | Directory selection, aliases, workspace navigation, direct bundle access, and guarded setup configuration writes exist. | Configuration writing is a useful precedent, but general ticket/handoff writing remains separate work. |

Sources: [CLI command definitions](../../internal/cli/cli.go), [task-context assembly](../../internal/taskcontext/context.go), [readiness checks](../../internal/orientation/readiness.go), [reader contracts](../../docs/readers.md), [setup preparation](../../internal/discovery/setup.go), [setup writing](../../internal/discovery/setup_write.go), [discovery reference](../../docs/discovery.md).

The vision's opening statement that the product is not implemented is stale. The cached local baseline also predates discovery integration. Preserve their product direction but use source and current records for implementation status. The discovery closeout records completed trials and subsequent reassessment; this research does not independently certify those results. [Vision](../../docs/vision.md), cached baseline (optional local history: `../../.cache/research/repository-studies/local-baseline.md`), [Document migration and verify discovery end to end](../records/project-discovery/issues/06-document-and-trial.md).

## Translate the lifecycle into responsibilities

Intent, specification, and implementation plan describe different questions. They need not become three mandatory files or three new status fields. This is a proposed interpretation of the supplied conversation, consistent with the current optional Spec relationship. [WoW with A](chatgpt-conversation://6aab1283-e634-83ed-9d50-361973232ead), [record profile](../../docs/agents/issue-tracker.md).

| Concern | CLI/core responsibility | Skill or human responsibility |
| --- | --- | --- |
| Intent and refinement | Retrieve authored outcome, criteria, decisions, and their sources; expose missing declarations. | Judge whether the requested outcome is understood; propose clarifications and accept changed scope. |
| Decomposition | Preserve explicit relationships and each child's own requirements. | Decide which contributions need independent tracking versus plan steps. |
| Starting work | Explain readiness and refresh context; later acquire a claim if included in scope. | Choose work and respect project authorization. |
| Interrupted work | Locate a handoff, identify its observed sources, and explain current differences and gaps. | Record observations and decide how to continue. |
| Verification | Present evidence, criteria, environment, revision, and gaps with provenance. | Run checks and judge whether observations satisfy criteria. |
| Acceptance | Validate the recorded decision and its retained snapshots. | Author acceptance and attribute any human approval separately. |
| Integration and deployment | Retain references and observations if the project needs them. | Existing review, CI, and deployment systems execute their operations. |

These are recommended boundaries, grounded in [the vision](../../docs/vision.md) and [recorded acceptance](../../docs/adr/0003-recorded-acceptance-for-readiness.md). CLI checks cannot establish that an agent understood a document or prevent code edits made outside managed operations.

For a Jira-backed project, Jira can own specified work fields while the repository owns technical contracts and plans. In the selected local milestone, native project records fill the work-record role. Confluence remains an optional documentation destination, not a required stage. Before synchronizing anything, decide ownership per field and distinguish remote snapshots from locally authored requirements. [Integration progression](../../docs/vision.md#integration-progression), [WoW with A](chatgpt-conversation://6aab1283-e634-83ed-9d50-361973232ead).

## Proposed delivery sequence

### Explain what needs attention

Derive a concise view from existing orientation checks. Group repeated causes, link affected commitments, keep work in progress separate from new pickup candidates, and display incomplete inventory and unknown facts prominently. Preserve authored commitment order rather than invent a priority score.

The decision is what a user needs on the first screen and what belongs in expanded detail. The implementation seam already exists between orientation results and CLI rendering. This is the lowest-cost product increment identified in both current workflow proposals and the earlier repository studies. [Workflow proposals](supporting/workflow-improvements.md#make-the-next-intervention-visible), [orientation renderer](../../internal/cli/orientation.go).

### Make one interrupted task resumable

Trial a handoff containing the work-item identity, requirement sources and observed digests, code checkout/revision, completed and unfinished steps, failed attempts worth retaining, evidence, and unresolved questions. Refresh current context before presenting a continuation.

Decide how handoffs are found, whether several may coexist, and what baseline is retained. A hash detects changed content but cannot reconstruct old text, especially for uncommitted files. Start with a supplied handoff path and explicit source observations if automatic discovery and historical comparison are not needed yet. This is a recommendation to test, not an agreed command contract. [Handoff proposals](supporting/workflow-improvements.md#make-interrupted-work-cheap-to-resume), [reader source semantics](../../docs/readers.md).

Keep mutable handoffs outside authoritative Spec and Context relationships by default. Otherwise routine progress updates can change acceptance inputs. Promote durable conclusions deliberately and retain accepted evidence separately. [Record acceptance](../../docs/agents/acceptance.md), [observations and commitments](supporting/workflow-improvements.md#keep-observations-and-commitments-distinct).

### Show what verification establishes

Trial a criterion-by-criterion view with explicit evidence, tested revision, environment, and missing observations. Distinguish authored coverage from agent-inferred matches. A passing backend test must not imply that configuration reached a device, and partial child coverage must not complete the parent outcome.

Choose the minimum useful evidence representation before introducing stable criterion IDs or coverage percentages. The existing acceptance model remains authoritative. [Evidence proposal](supporting/workflow-improvements.md#explain-what-the-evidence-establishes), [acceptance ADR](../../docs/adr/0003-recorded-acceptance-for-readiness.md).

### Add focused writes and claims only where the milestone needs them

If skills can author records for the first usable loop, prioritize read and resume behavior. If record maintenance is the obstacle, select one focused write, such as appending a progress observation or saving a handoff. Define its proposed diff, expected revision, conflict behavior, prose preservation, and failure recovery before implementation. Existing setup preparation/apply code is a precedent to inspect, not a complete work-record writer. [Setup source](../../internal/discovery/setup.go), [setup write source](../../internal/discovery/setup_write.go).

Exclusive implementation pickup requires a separate claim record and an atomic readiness-and-claim operation within its supported coordination boundary. Human assignment, a Git commit, or a stale heartbeat is insufficient. The vision limits the initial guarantee to one machine and requires explicit takeover. Keep claims outside this milestone if the human continues to direct pickup. [Work claims](../../docs/vision.md#work-claims).

### Follow with read-only Jira retrieval

After the local loop, specify a bounded adapter that retrieves an issue and relevant parent requirements with source identity and freshness. Decide supported Jira deployment, authentication, required fields, parent representation, unavailable-source behavior, and snapshot ownership before implementing it. The local reader currently supports local file relationships, so an adapter needs an explicit materialization or provider boundary.

Keep write-back, status synchronization, and publication to Confluence separate. An imported Done status must not bypass local acceptance checks. These are planning requirements inferred from the approved [integration direction](../../docs/vision.md#integration-progression) and [task-context contract](../../docs/readers.md#read-task-context). This research did not validate Jira APIs or tenant permissions; that is a focused follow-up investigation.

## Decisions before implementation planning

The destination is selected. These questions are sharp enough for future decision tickets, with dependencies refined after the scope discussion:

1. **What proves the local directing-and-resuming loop is usable?** Select the end-to-end scenario and observable success criteria.
2. **What belongs in the shortest useful intervention view?** Decide grouping, expansion, and treatment of unknown facts.
3. **What does a handoff retain, and how does a session select it?** Decide source baselines, uncommitted observations, and multiple handoffs.
4. **What should resumption do when requirements or readiness changed?** Distinguish facts the CLI reports from continuation judgment.
5. **How should criteria and evidence gaps be presented?** Decide the initial advisory representation without changing acceptance policy.

Two boundary questions were answered during research: skills continue to author records and handoffs, and the human directs pickup. Therefore general work-record writes and exclusive implementation claims are deferred. Their discussion above describes later dependencies, not work needed to finish this map.

Further fog includes the right long-term retention policy for source baselines and the amount of configurable workflow policy users actually need. Do not invent those requirements before a trial. Jira API mapping, remote freshness, and parent-chain representation belong to the early follow-up. TUI/web interfaces, general two-way synchronization, cross-machine claims, and a custom agent runtime are beyond the selected local destination unless it is explicitly redrawn.

## Suggested usability trial

Use the vision's backend/device scenario. A backend contribution depends on an unresolved success-response decision. After that decision is recorded, an agent implements and records local evidence, then leaves a handoff before device verification. Change an observed requirement and ask a fresh session to resume.

Observe whether the person identifies the required intervention, the agent notices the source change, and missing device evidence remains visible. Record repeated investigation and record-maintenance effort rather than inventing a performance target. Require at least one independent non-Git project to retain the product's location independence. This proposed trial combines the [vision scenarios](../../docs/vision.md) and [existing workflow trial](supporting/workflow-improvements.md#a-small-sequence-to-evaluate).
