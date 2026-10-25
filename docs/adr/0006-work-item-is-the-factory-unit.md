# The work item is the factory's unit of planning and parallelism

## Context

The factory turns one work item into one job, run by one native agent session with bounded repairs. The autopilot builds one change per project at a time. The unadopted [software factory proposal](../software-factory-plan.md) suggested porting the orchestrate skill's checkpoint protocol into the engine. Other factories split one change into a task graph and run several implementers on it ([research note](../research/software-factory-2026-10-22.md)):

- CodeAF orders tasks whose write scopes overlap and lands them under a repository-wide lock.
- SWE-AF merges parallel branches level by level, hands conflicts to an LLM merger, and can drop criteria as accepted debt.
- Fabro runs parallel branches in one shared checkout.

None of these projects publishes repeated, controlled evidence that parallel implementers within one change beat one implementer. In this project's own evidence, the Axpilot #264 orchestrate run used 52 subagents over about four hours. It produced 1,201 production lines and 5,311 test lines. The person's first demo, after 15 commits, changed the UI that every reviewer had verified. The vision already directs grooming to propose ticket splits around independently verifiable outcomes, with the person deciding.

## Decision

The factory plans a change by splitting it into work items. The refiner proposes a split, and a person accepts it as the decision that agrees what done means. The autopilot never accepts a split. Each child runs the normal line, gets its own approval from the person or the ask policy, and merges by itself.

A job keeps one implementation writer and one active implementation claim per work item. The engine has no checkpoints or parallel implementers inside a job. A builder may use its harness's own subagents inside its one session; the factory records one attempt.

Parallelism comes from building independent ready work items at once, each with its own job, claim, worktree, verification and approval. Merges into one branch run one at a time on a verified merge commit. A change that no longer merges cleanly is rebuilt from the branch tip, with its requirements unchanged.

The orchestrate skill remains the explicit, hand-driven checkpoint workflow. The engine does not port it.

## Alternatives considered

Checkpoints inside a job would keep one card and one approval for a large change. They would need an engine-side planner, checkpoint packets, integration of checkpoint results, and per-checkpoint verification. The person would see the change only at the end, which is the failure the Axpilot run showed.

Parallel implementers with write scopes would shorten a large change's build. Scopes are predictions, and none of the examined projects enforces them across shell commands. A missed overlap still needs a merge step, and a false one serialises work for nothing.

One approval for a whole split would save clicks. It would hold every child until the last is done, and dependent children would have to build on unmerged candidates. That breaks the rule that admission requires merged prerequisites.

## Consequences

A split change produces several cards and several approvals. The ask policy can delegate approvals once a project has a record of trouble-free merges. Children must be independently mergeable, and dependent children wait for their prerequisites to merge.

An item that is tightly coupled and too large for one builder session has no answer beyond bounded repair. If a trial shows such an item, this decision should be reconsidered.

Parallel builds can conflict. A conflict costs a rebuild, and a conflict found after approval also costs a second approval. Trials count conflicts before any prediction of write scopes is added.

The vision's single active implementation claim per work item and [ADR 0005](0005-factory-controller-owns-operational-state.md) remain unchanged.
