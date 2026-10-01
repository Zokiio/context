# Helix Loop adoption assessment

Research note, 2026-10-01. This assessment informs the existing [workflow trial design](helix-design.md). It does not authorize installation or change the agreed workflow.

My recommendation is to adapt the checkpoint loop into the existing workflow skills. The earlier design already specifies the context, evidence, and review boundaries that this clone leaves to convention. The clone supplies useful examples, but its installer and gate policy need changes before they fit this project.

The [workflow reference](helix-loop-reference.md) provides a closer source walkthrough of the roles, artifact handoffs, review formats, example checkpoints, and viewer.

## Source and inspection scope

I inspected the clone at commit [`c52ba265a778c17c11757ef86630686728d6aa02`](https://github.com/johnarks/helix-loop/tree/c52ba265a778c17c11757ef86630686728d6aa02), cloned to `/tmp/context-helix-loop-assessment`. I read its skills, schema, hook, installer, routing, viewer, and examples. I did not install it, run its hook, execute a workflow, or observe its runtime behavior. Findings about execution below follow the source directly or identify an inference explicitly.

The repository identifies itself as an unofficial reconstruction. Shopify's article describes small checkpoints, independent test preparation, ordered behavior and UI gates, two isolated reviewers, and engineer approval by default. Engineers can authorize skipped approvals while retaining the other checks. Shopify also describes its application behavior CLI and revision commits with archived evidence. The article does not publish the enforcement code, so it cannot establish that the clone reproduces Shopify's guarantees. [Clone attribution](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/README.md#L5-L9), [Shopify's engineering article](https://shopify.engineering/helix).

## What the clone provides

The skills describe a useful procedure. A planner proposes small checkpoints. Separate agents prepare tests, implement a checkpoint, and execute tests. Two isolated code reviewers examine the changes, a separate builder handles corrections, and affected checks run again. The implementer receives scoped reference material rather than the orchestrator's full conversation. These are instructions to the agent environment. The repository does not implement a scheduler or enforce those role boundaries. [Orchestrator planning and delegation](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/orchestrator/SKILL.md#L20-L78), [review protocol](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/adversarial-review/SKILL.md#L10-L33).

Its plan is one `.helix/checkpoints.json` array. The schema requires an ID, title, description, UI applicability, and at least one done criterion per checkpoint. Gate fields remain optional, all four accept `skipped`, and duplicate IDs are not prohibited. There is no checked-in state schema or runtime schema validator. The viewer parses plan JSON and displays its statuses. It neither records approvals nor reads the separate `state.json`, so its gate display need not reflect the hook's input. [Plan schema](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/schemas/checkpoints.schema.json), [viewer](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/viewer/index.html#L60-L118).

## The hook checks agent-written labels

The hook reads `.helix/state.json` and inspects only the entry named by `current_checkpoint`. It returns exit code 2 when a present gate value is neither `passed` nor `skipped`. Its check has several concrete limits. [Hook source](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/hooks/stop-gate-check.sh#L34-L85).

- Missing state, unreadable state, invalid JSON, missing Python, or an absent current checkpoint permits stopping.
- Empty or missing gates permits stopping. Required gate names are not checked.
- Any gate can be `skipped`. The hook never checks applicability, a reason, or human authorization.
- It never reads tests, evidence files, reviews, approvals, code revisions, or the plan. It does not check gate order or earlier checkpoints.

The agent writes the same labels that the hook trusts. A passed label therefore cannot establish a passing test or an independent approval. A Stop hook also acts when a turn finishes, so this hook does not intercept a commit or advancement to another checkpoint.

Claude Code documents Stop continuation, exceptions for user interruption and API failures, and a default cap of eight consecutive continuations. The clone does not read hook input or account for an unresolved human decision. Its claim that the agent cannot quit early exceeds these limits. [Claude Code Stop documentation](https://code.claude.com/docs/en/hooks#stop-input).

## Approval and repair policy needs correction

The orchestrator promises two human stops, then requires checkpoint approval after every checkpoint. It initializes all gates as pending before its mandatory plan-approval stop. There is no waiting-for-human phase in the hook. By inspection, a normal Stop at plan approval would encounter open gates and request continuation. This interaction was not run. Autonomous mode substitutes recorded self-review for the human gate rather than retaining a distinct approval waiver. [Orchestrator policy](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/orchestrator/SKILL.md#L33-L98).

The failure policy allows unlimited retries and offers no handling for unavailable checks or repair attempts that stop making progress. It also requires every new test to fail before implementation and declares an already passing test wrong. That requirement can reject valid regression tests during refactoring or valid negative cases against existing behavior. Preserve independently derived expectations and require the intended missing behavior to fail for the expected reason. [Failure policy](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/orchestrator/SKILL.md#L102-L108), [test planning](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/test-planner/SKILL.md#L12-L16).

The supplied state example has statuses and gate strings but no tested revision, source digests, actor identity, or evidence reference. Restart guidance says to resume from state without defining freshness checks. The skill requires committing checkpoint evidence, and retained human feedback becomes guidance for later agents without a separate adoption decision. [State example](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/examples/state.example.json), [commit and restart instructions](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/skills/orchestrator/SKILL.md#L81-L99), [feedback template](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/templates/learnings-template.md#L1-L15).

## Installing it would change more than this project

The installer always writes Codex skills to `~/.codex/skills`, including with project scope. A same-name skill with different content is overwritten. Project scope also appends a routing block to `AGENTS.md` that directs future feature work to its orchestrator. The installer does not copy the viewer, schema, or templates that skills reference. Its Claude settings merge replaces unreadable or invalid existing settings with an empty object before adding the hook. These are reasons to review and adapt installation instead of running it here. [Installer](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/install.sh#L61-L174), [routing block](https://github.com/johnarks/helix-loop/blob/c52ba265a778c17c11757ef86630686728d6aa02/templates/agent-routing.md#L1-L23).

The clone's claim that Codex lacks Stop hooks is outdated. Current official documentation supports repository `.codex/hooks.json` and Stop continuation through `decision: "block"` or exit code 2 with feedback on stderr. Non-managed hooks require review and trust, and project hooks require a trusted project configuration. The clone's installer wires no Codex hook. Installed-host compatibility remains untested. [Codex Stop hooks](https://learn.chatgpt.com/docs/hooks#stop), [hook locations and trust](https://learn.chatgpt.com/docs/hooks#review-and-trust-hooks).

The useful borrowing is the checkpoint procedure and independent correction loop. The supplied state strings and Stop hook do not establish verified completion. A local adoption should preserve the existing record, revision, approval, and recovery policies while adapting the procedure.

## Fit with the existing workflow

The [19 confirmed design choices](helix-design.md#confirmed-choices) already cover the main loop. They define fresh implementing agents, independent test preparation, checkpoint review, correction, and combined verification. The remaining design question is the first trial, not whether to adopt these basic roles.

| Concern | Fit and necessary adaptation |
| --- | --- |
| Checkpoint context | Keep the agreed separate checkpoint files and one plan per work item. Give the orchestrator full task context and give each implementer explicitly selected sources without inherited conversation history. The clone's scoped delegation is useful, but its single JSON plan does not meet the agreed document layout. |
| Verification | Retain separate test preparation and execution. Preserve requirements-derived expectations when correcting code. A test that already passes can still protect existing behavior. Include visual or environment checks when the checkpoint outcome requires them. |
| Independent review | Reuse the existing standards and specification review obligations. Two general architecture reviewers do not substitute for those distinct questions. Review identified local code revisions and renew affected checks and reviews after corrections. |
| Evidence and acceptance | Keep checkpoint progress separate from acceptance of the whole work item. Retain actual observations, their actors, and tested revisions. Arrange independent combined verification before applying the existing acceptance procedure. |
| Feedback | Retain feedback with its source and scope. Apply task-local corrections within agreed scope. Broader mandatory guidance still requires explicit adoption. |
| Storage and recovery | Preserve the existing authoritative records. Keep generated evidence and Acceptance records in their designated ignored locations. Inspect current context, code, and still-running work when resuming, rather than trusting persisted gate labels. |
| Human decisions and blocked checks | Preserve initial plan review and checkpoint review for the first trial. Waiting for a decision, an unavailable check, or stalled repairs must remain an explicit unresolved outcome. Continue only independent work supported by current facts. |

These adaptations follow the [workflow design](helix-design.md), [implementation skill](../../.agents/skills/implement/SKILL.md), [review skill](../../.agents/skills/code-review/SKILL.md), [acceptance procedure](../../docs/agents/acceptance.md), [storage conventions](../../docs/agents/issue-tracker.md), and [recovery procedure](../../.agents/skills/recovery-notes/SKILL.md). These documents remain the source for the local workflow.

The product boundary also fits. Under the [agent execution model](../../docs/vision.md#agent-execution-model), skills coordinate the work inside an existing agent environment. `ctx` supplies context, orientation, recovery observations, and record checks. This adoption does not require a new agent runtime or execution responsibility in the Go core.

## Recommended next trial

The existing [isolated fixture proposal](orchestrator-trial.md) is a suitable first test of the adapted loop. It has two checkpoints and an independent check of their combined behavior. Its proposed success measures cover review effort, supplied context, correction integrity, and combined verification. A deliberate failed check would reveal whether the loop preserves failed evidence and verifies the corrected revision before advancing.

For that trial, I recommend one explicitly invoked orchestrator skill that coordinates existing responsibilities. Keep implementation sequential. Defer automatic routing, global skill installation, the browser viewer, enforcement hooks, and new CLI operations until the trial identifies a concrete need. This recommendation follows the confirmed execution boundary and the trial's existing exclusions.

The assessment provides source findings and an adoption recommendation. It does not establish improved review time, successful runtime enforcement, or a completed trial. Q21 remains the choice between this fixture and an approved real work item.
