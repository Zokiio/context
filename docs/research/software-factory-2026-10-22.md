# Software factory research for Waymark

Research date: 22 October 2026, Europe/Stockholm. This report compares the seven projects and two StrongDM pages supplied for the software-factory research. It is source research and design input, not an engine selection or live evaluation. [Pinned source records](software-factory-2026-10-22.sources.json)

## Product direction

**Waymark becomes the software factory:** it turns project intent into coordinated implementation, independent verification, and an explicit acceptance decision. It can drive development across different repositories using existing coding agents and tools.

Waymark should own requirements, dependency readiness, workflow policy, dispatch, bounded repair, evidence, and acceptance. A selected execution backend can own its native sessions and tools. An existing workflow engine could supply implementation machinery without taking ownership of Waymark's requirements or acceptance model.

The research shortlist is:

- **Fabro:** evaluate for explicit coding workflows and an external-agent integration.
- **CodeAF:** evaluate as a local factory implementation and reference for practical verification and recovery.
- **SuperPlane:** evaluate when shared workflows, operational UI, and integrations are central.

Lightspeed and Open-Inspect offer broader execution infrastructure. SWE-AF provides useful adaptive orchestration patterns but requires careful interpretation of completion and verification. Attractor supplies specifications and methodology rather than a turnkey official engine.

Building the factory platform is the current product direction. These comparisons inform its design and possible component choices; they do not require starting application pilot work first.

## What the projects actually provide

| Project                              | Current shape                                                                                                                        | Fit here                                                                                                 |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| StrongDM Attractor                   | Specifications for graph orchestration, a coding loop, and a unified LLM interface; the official repository is not a turnkey engine. | Reference semantics and factory methodology. Evaluate an implementation separately.                      |
| Fabro                                | Graph-based coding workflows, human gates, checks, repair, Git checkpoints, server/UI, and sandbox backends.                         | First candidate for an explicit workflow controller or a standalone factory.                             |
| CodeAF                               | Local Go product with TUI/headless operation, dynamic task graphs, worktrees, verification, and landing/recovery.                    | First candidate for a small local factory trial.                                                         |
| SuperPlane                           | Factory backlogs, work orders, automation lines, runners, integrations, and PR feedback.                                             | Strong candidate for an operational platform with shared workflows.                                      |
| Lightspeed                           | Managed agent fleets with its own event-sourced agent loop, Temporal workflows, and durable tool execution.                          | Infrastructure alternative when long-lived agents and custom durable workflows are the main requirement. |
| Open-Inspect, from background-agents | Background coding sessions, native harness adapters, sandboxes, child sessions, and event/scheduled automation.                      | Execution-platform candidate; less directly aligned with Waymark's acceptance model.   |
| SWE-AF                               | AgentField-based engineering-team API: planning, parallel coding, review, integration, verification, and PR/CI work.                 | Useful adaptive workflow reference; requires stricter mapping of success and acceptance before adoption. |

### StrongDM: separate the method, the spec, and an implementation

The [factory site](https://factory.strongdm.ai/) describes StrongDM's experience with specifications, scenarios often kept outside the codebase, simulated dependencies, and noninteractive implementation. Its advocacy of removing human code review is a publisher report, not independent evidence that this is reliable for these repositories. Keeping scenarios outside a repository also does not establish access isolation.

The [Attractor product page](https://factory.strongdm.ai/products/attractor) and the [pinned specification](https://github.com/strongdm/attractor/blob/fb57a55ed97372a27ac90102f436947e29f48426/attractor-spec.md) differ in emphasis. The spec defines graph routing through conditions and outcomes; deterministic routing does not make agent-generated work deterministic. It permits external CLI backends, so adopting the workflow model does not require replacing native agents with a custom model loop. Its generic success rules include partial success at goal gates and optional automatic success status. Those success conventions must not become Waymark acceptance rules.

The [official repository](https://github.com/strongdm/attractor/blob/fb57a55ed97372a27ac90102f436947e29f48426/README.md) publishes natural-language specifications. “Install Attractor” therefore requires choosing an implementation or building one.

### Fabro: the first workflow candidate

The current implementation delegates workflow execution to **Petri**, uses **Pebble** for its native agent loop, and supports host, Docker, and Daytona execution. This is more specific than calling it an Attractor wrapper. The inspected [lockfile](https://github.com/fabro-sh/fabro/blob/9d9cbe9f417006603a4bbc4372ffdb2477afa314/Cargo.lock) pins those dependencies; the analysis follows the locked Petri revision.

Useful evidence includes immutable admitted graph snapshots, real workflow validation, and recovery that requires the recorded workspace and commits to survive. Recovery is not a promise to reconstruct a lost machine. [Admission](https://github.com/fabro-sh/fabro/blob/9d9cbe9f417006603a4bbc4372ffdb2477afa314/lib/components/fabro-petri/src/admission.rs), [recovery](https://github.com/fabro-sh/fabro/blob/9d9cbe9f417006603a4bbc4372ffdb2477afa314/lib/components/fabro-petri/src/recovery.rs)

Resume and interview tests cover interruption and gate behavior; they were read, not run. [Runtime](https://github.com/fabro-sh/fabro/blob/9d9cbe9f417006603a4bbc4372ffdb2477afa314/lib/components/fabro-petri/src/runtime.rs), [resume tests](https://github.com/fabro-sh/fabro/blob/9d9cbe9f417006603a4bbc4372ffdb2477afa314/lib/components/fabro-petri/tests/resume.rs)

Its [ACP agent interface](https://github.com/fabro-sh/fabro/blob/9d9cbe9f417006603a4bbc4372ffdb2477afa314/docs/public/core-concepts/agents.mdx) can delegate to an external agent that owns authentication, tools, and its model. An adapter to a separate execution service is plausible, not present in the inspected evidence. It would need exact runtime/account/environment binding, explicit workspace ownership, cancellation forwarding, and reconciliation before retries.

A material default: the locked [Petri ACP client](https://github.com/lithoscomputer/petri/blob/e34d60586a92719ba02a4f1a6c85a723d6376406/crates/attractor/steps/src/acp/mod.rs) automatically allows permission requests unless a configured pre-tool hook blocks them. Hooks depend on what the agent reports. A human workflow gate is therefore distinct from tool authorization. Killing an ACP wrapper also cannot establish that a native execution or its external effects stopped.

### CodeAF: the smallest local experiment

CodeAF is a local Go application, not merely a client requiring a separate AgentField control-plane deployment. Its headless command runs planning, coding, checks, and structured outcome reporting. It has a dynamic graph, write-scope coordination, worktree support, and landing/recovery behavior.

The [headless documentation](https://github.com/Agent-Field/CodeAF/blob/c740a5417230dcabe074c7a7919c3403a44ebc79/docs/HEADLESS.md) matters for a trial: the default command edits the current directory in place, and temporary state needs explicit retention. Start in a disposable checkout and retain its database. The inspected dynamic orchestrator stops new launches at its dollar cap; in-flight work may finish, so it is not a hard billing ceiling. [Budget behavior](https://github.com/Agent-Field/CodeAF/blob/c740a5417230dcabe074c7a7919c3403a44ebc79/internal/orchestrate/run.go#L18)

The [verification implementation](https://github.com/Agent-Field/CodeAF/blob/c740a5417230dcabe074c7a7919c3403a44ebc79/internal/verify/run.go) collects actual process outcomes and preserves original check commands across the change. Tests reject some no-op verification substitutions. This is useful evidence discipline, but does not establish an independent protected verifier. [No-op evidence tests](https://github.com/Agent-Field/CodeAF/blob/c740a5417230dcabe074c7a7919c3403a44ebc79/internal/verify/noop_evidence_test.go)

CodeAF owns its agent loop. An offered Codex subscription route is not proof of compatibility with a separate service's native session bindings. Treat it as a standalone trial with Waymark inputs and outputs, not an assumed interchangeable execution backend. Its default branch is `dev`; the reviewed revision is an early-preview product.

### SuperPlane: more than an integration canvas

Current SuperPlane includes a real factory model: work orders, ordered automation lines, saved execution snapshots, agent runs, and verification feedback. The [line dispatcher](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/models/factory_work_order_line_dispatch.go) persists execution choices and versions. The stack includes Go, PostgreSQL, RabbitMQ, workers, and a React UI.

The [Codex runner](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/components/runner/codex/run.js) uses the native CLI and resume identifiers. Implementation runs bypass CLI approval/sandbox controls, making the actual runner environment the confinement boundary. PR feedback includes bounded repair and revision checks.

Two policies need attention. Its [GitHub check evaluator](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/integrations/github/components/checks/wait_for_pull_request_checks_eval.go) treats neutral, skipped, and cancelled conclusions as nonfailing. Its [terminator](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/workers/execution_terminator.go) can finalize cancellation despite a component cancellation error. Neither policy proves that required acceptance checks ran or that external work stopped.

SuperPlane overlaps with Waymark's proposed scheduling and with a separate execution platform. That makes it a credible replacement option, but an integration must choose one owner for dispatch and claims. Its root Apache-2.0 license has an `ee/` exception with separate terms. The inspected work orders, automation lines, and native runners are community code; current enterprise gates concern features such as custom roles and groups. [Enterprise boundary](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/docs/contributing/enterprise-licensing.md)

### Lightspeed: durable fleet infrastructure

The current repository describes managed agent fleets with Rust, Temporal, and PostgreSQL, rather than establishing itself as a drop-in DOT/Attractor factory. Agents retain logical state separately from machines attached for OS work.

Its [drive reducer](https://github.com/smartcomputer-ai/lightspeed/blob/86f6adf3cd28cde11644403c886ae08324cf21bb/crates/harness/src/core/drive.rs) separates decisions from runtime I/O. Completed parallel tool results are committed individually. [Checkpoints](https://github.com/smartcomputer-ai/lightspeed/blob/86f6adf3cd28cde11644403c886ae08324cf21bb/crates/temporal-runtime/src/checkpoint.rs) are validated accelerators over authoritative history. Its documentation distinguishes replay durability from exactly-once external effects.

Versioned [workflow tools](https://github.com/smartcomputer-ai/lightspeed/blob/86f6adf3cd28cde11644403c886ae08324cf21bb/crates/temporal-runtime/src/worker/activities/workflow_tools.rs) have keyed execution identity, recovery, and cancellation. A custom delegation workflow to an external execution service is conceivable. It would still need implementation and leaves a substantial controller and operations stack. Choose this path for broader fleet requirements, not solely to automate a few coding tasks.

### Open-Inspect: an execution-platform alternative

The background-agents repository now brands the product **Open-Inspect**. It supports Codex, Claude, and OpenCode harnesses, isolated coding sessions, child sessions, and automation. The inspected [Codex adapter](https://github.com/ColeMurray/background-agents/blob/776c771acf9e5fd389722f3506ea8587e22a54de/packages/sandbox-runtime/src/sandbox_runtime/harness/codex/__init__.py) uses app-server and checks saved-state compatibility rather than blindly adopting every previous session.

There are meaningful [process containment tests](https://github.com/ColeMurray/background-agents/blob/776c771acf9e5fd389722f3506ea8587e22a54de/packages/sandbox-runtime/tests/test_codex_process.py), including detached descendants and unrelated processes. Those tests support a narrower claim than universal cancellation safety.

The main deployment uses Cloudflare control-plane services and sandbox providers. A [container alternative](https://github.com/ColeMurray/background-agents/blob/776c771acf9e5fd389722f3506ea8587e22a54de/docs/CONTROL_PLANE_CONTAINER.md) uses a single-process Node/SQLite service, S3-compatible object storage, and global-database replication. That replication excludes per-session state and jobs: complete recovery requires a whole-volume backup. The alternative also has feature gaps and requires one live data owner. The README describes trusted single-tenant assumptions and credential-retention risks in snapshots. A full account/isolation audit was outside this review.

This is a credible alternative to building a separate execution product. It does not remove the need for Waymark's requirement snapshots and acceptance evidence.

### SWE-AF: capable orchestration, permissive completion paths

SWE-AF is a separate Python/AgentField service with a running control plane, rather than another mode of CodeAF. It offers planning, coding/review loops, conditional deeper QA, integration, and final verification, with several runtime options.

The critical finding is in the [coding loop](https://github.com/Agent-Field/SWE-AF/blob/4d3dfe719e5ce9e2e8cd454722295bbe6512542f/swe_af/execution/coding_loop.py#L309): a nonfatal reviewer exception produces an approved, nonblocking review. The advisor can also revise criteria or accept debt. Disabling advisor/replanning features removes some adaptive behavior but does not fix the reviewer-exception path. [DAG executor](https://github.com/Agent-Field/SWE-AF/blob/4d3dfe719e5ce9e2e8cd454722295bbe6512542f/swe_af/execution/dag_executor.py), [configuration](https://github.com/Agent-Field/SWE-AF/blob/4d3dfe719e5ce9e2e8cd454722295bbe6512542f/swe_af/execution/schemas.py)

Final verification can report failure even when the overall service execution succeeds. The factory must interpret those separate results. A same-repository agent with write access is also not an independent verifier. Study this project's adaptive execution, but do not map its execution-success status directly to Waymark acceptance. [Verification agent](https://github.com/Agent-Field/SWE-AF/blob/4d3dfe719e5ce9e2e8cd454722295bbe6512542f/swe_af/reasoners/execution_agents.py#L528), [result tests](https://github.com/Agent-Field/SWE-AF/blob/4d3dfe719e5ce9e2e8cd454722295bbe6512542f/tests/test_empty_build_guard.py)

## Waymark's responsibilities

Waymark's [vision](../vision.md) already describes project goals, dependencies, readiness, evidence, and acceptance. At the research snapshot, its context, orientation, and resumption readers supplied useful inputs, while durable claims, scheduling, and execution supervision remained future work. Subsequent platform development is outside this source comparison; consult the current implementation records for its status.

The [acceptance reader](../readers.md) checks records, attribution fields, fingerprints, and evidence snapshots. It does not execute tests, authenticate the recorded actor, or certify the current checkout. A newer HEAD alone does not invalidate unchanged historical acceptance; changed requirements or relevant evidence can make it stale.

| Responsibility | Owner |
| --- | --- |
| Project intent, requirement versions, dependency readiness, workflow policy and final acceptance | Waymark |
| Durable job stages, dispatch claims, retries, deadlines and recovery decisions | Waymark's factory controller |
| Native agent sessions, tools, execution identity and receipts | Selected worker or execution backend |
| Workspace allocation and cleanup | One designated workspace owner |
| Independent checks and protected result evidence | Verifier outside the implementer's write authority |

Work IDs, requirement fingerprints, candidate commits, and execution IDs should connect these records. Each operational fact needs one authority. A stale claim or lost connection does not establish that the old execution stopped.

## Requirements informed by the research

The factory platform should provide:

1. **Versioned work inputs.** Freeze accepted requirements, repository/base revision, allowed scope, required checks, and stop conditions. Workers may propose requirement changes; they cannot approve them.
2. **Durable coordination.** Persist job stages, dispatch identity, claims, limits, and attempts. Reconcile uncertain outcomes before issuing another action.
3. **Isolated workers.** Allocate explicit workspaces and execution environments. Define cancellation and recovery through the actual worker, including surviving child processes and external effects.
4. **Independent evidence.** Keep verifier policy outside the implementer's write authority. Record executed, failed, skipped, unavailable, and inconclusive checks separately.
5. **Explicit acceptance.** Bind the decision to requirements and the candidate revision. A completed worker or successful service request is not an acceptance decision.
6. **Operational controls.** Make pending reviews, exhausted limits, missing evidence, and uncertain execution visible in the service, CLI, and any web workspace. Use one shared controller behind clients.

Platform verification should cover missing review, skipped required checks, changed requirements after approval, lost workspaces, interrupted dispatch, and uncertain external effects. Offline fixture workers can establish these contracts before evaluating model-driven application work.

If an external engine is adopted, map its statuses to Waymark's model explicitly. Defaults such as automatic approval, partial success, or treating skipped checks as nonfailing must not silently become acceptance policy.

## Evidence and limitations

Three parallel research agents investigated graph/fleet engines, workflow/session platforms, and AgentField projects. They inspected primary documentation, implementation, tests, and licenses at pinned revisions, then reviewed their report sections. The lead researcher independently checked the SWE-AF reviewer-exception path and Fabro/Petri ACP permission defaults.

No candidate was installed, no external implementation was copied, and no provider-backed run or live recovery test was performed. Test source was inspected; that is not a passing test result. Publisher benchmarks were not used to rank reliability.

Observed root licenses: Attractor, Lightspeed, CodeAF, and SWE-AF are Apache-2.0; Fabro and Open-Inspect are MIT; SuperPlane is Apache-2.0 outside its separately licensed enterprise directory. Fabro also carries third-party review rules and notices. The source ledger records these observations, not a complete transitive license review.

The Waymark source snapshot was `abe3da6beac25a42620be363830d03c108ba9d9d`. This saved edition focuses on Waymark as a general factory and does not designate a particular application as its first target.
