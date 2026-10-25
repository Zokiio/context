# Planned steps and parallel builds in the factory

Proposed on 25 October 2026 from a planning session on `dev` at `afe9989`. The user made the decisions recorded under [Decisions](#decisions). None of the tickets is a current commitment yet.

## Outcome

A person can have the factory split a change that holds several outcomes into smaller work items, then let it build the independent ones at the same time. Each piece lands on main by itself, after its own approval, and a piece that no longer merges is rebuilt instead of stranded. A person still makes two decisions per change: agree what done means, then approve it.

## Observed need

- **One build per project.** The autopilot starts one build, one test-writing run and one refinement per project at a time (`internal/factory/pilot.go`, `pilotBusy`). `ctx factory solo` runs one implementation worker and one verification worker. Ready work that doesn't depend on anything waits in line.
- **Large changes surface late.** In the Axpilot #264 orchestrate run on 2026-10-04, 52 subagents worked for about four hours on one issue. They produced 1,201 production lines and 5,311 test lines. The user's first demo, after 15 commits, changed the UI that every reviewer had verified ([workflow adaptation spec](../workflow-adaptation/spec.md#problem)). A smaller first piece would have reached the person earlier.
- **Merges race.** Two approved changes build their merge commits from the same tip. The first moves the branch; the second fails with "the branch moved during verification", and the autopilot retries it (`internal/factory/merge.go`). Nothing serialises merges into one branch. A candidate that no longer merges cleanly is treated as a retryable merge failure.

Grooming already proposes "ticket splits around independently verifiable outcomes" and leaves the choice to the person ([vision](../../docs/vision.md#after-the-first-milestone)). The factory has no way to carry that out.

## Decisions

- **The work item is the factory's unit of planning and parallelism.** The refiner may propose splitting an item. A person accepts the split as the plan decision. Each child runs through the normal line on its own. A job keeps one implementation writer. The engine doesn't port the orchestrate skill's checkpoints. [ADR 0008](../../docs/adr/0008-work-item-is-the-factory-unit.md) records the reasons and the alternatives.
- **Parallel builds rely on a verified merge queue.** Each build keeps its own claim, worktree and frozen protected paths, as today. Merges into one branch run one at a time. A change that no longer merges cleanly is rebuilt from the branch tip. Nobody predicts write scopes up front; the trial counts conflicts first.
- **Each split child gets its own approval.** A split always waits for a person, even on autopilot. Accepting it is the plan decision for every child. Each child is then approved by the person or by the ask policy, and merges by itself.
- **Pipeline shape stays open.** [Decision 01](../records/factory-parallel-runs/decisions/01-pipeline-shape.md) holds the options and the recommendation, a fixed line with policy switches. No ticket here depends on it.

## Required behavior

### Merge queue

- One merge per target branch verifies at a time. Further approved merges wait in approval order and start by themselves when the one ahead finishes, whether it merged or failed. The queue survives a restart.
- A queued merge builds its merge commit from the branch tip when its turn comes, so the required checks run on top of everything merged before it.
- A candidate that doesn't merge cleanly is recorded as a conflict, distinct from a failed check and from a moved branch. The autopilot doesn't retry a conflict. A branch moved from outside the factory keeps today's bounded retry.
- The board shows a waiting merge and a conflict in the Merge column. Metrics count conflicts.

### Parallel builds

- A project setting limits how many of its builds run at once. It defaults to 1, today's behavior. A project administrator changes it through the CLI, HTTP and the settings page.
- The autopilot starts builds for ready, queued items in commitment order until the limit is reached. Readiness and admission are unchanged. A dependent item still waits until its prerequisite is merged into its base.
- `ctx factory solo --builders N` runs N implementation workers and N verification workers, each with its own identity. The default is one of each.
- Refinement and test writing stay at one run each per project. The trial measures whether builds wait on them.

### Rebuilding a change that no longer merges

- When a candidate's checks pass, the factory checks that it still merges cleanly into the current branch tip before the card asks for approval. If it doesn't, the factory starts a linked replacement run from the tip with the same admitted requirements and accepted locked tests. No person decides this. After two automatic rebuilds of one item, the card asks the person instead.
- Approval repeats the check before publishing anything. A candidate that stopped merging cleanly is not accepted, and its rebuild starts.
- A conflict that appears after approval, while the merge waited in the queue, offers a rebuild on the card, and the autopilot takes it. The rebuild supersedes the unmerged acceptance, returning the ticket's Acceptance link and execution state through an expected-revision write while retaining the old decision and evidence. The new candidate is checked and approved again, by the person or by the ask policy.
- A rebuild never relaxes requirements. Changed requirements send the item back to Plan, and locked tests that no longer apply to the tip ask the person.
- History shows each rebuild, the original candidate and the reason.

### Splitting a change

- The refiner may propose a split when an item holds several independently verifiable outcomes, and keeps tightly coupled changes together. A person can also ask for a split proposal. A proposal has two to five children. Each child has a title, a description, at least one criterion, and blockers among earlier siblings only.
- Every original criterion maps to at least one child or is listed as dropped with a reason. The service refuses a proposal with an unmapped criterion, and the decision shows dropped criteria plainly. An agent can never drop a criterion silently. SWE-AF's advisor can, and that is what this rule excludes.
- Only a person accepts a split, whatever the ask policy says. The person may edit children before accepting. Acceptance does three things in one retained publication plan, which restart reconciles: it creates the children as queued work items that link the original as context, re-points items that depended on the original to all of its children, and archives the original with a comment naming the children and the criterion mapping.
- A go-ahead given with the split carries to every child. It stops at anything new for a child: a question, a flagged criterion, or a test author's concern.
- A split is refused while the original has an active or uncertain refinement, test-writing run, job or verification. Rejecting a split leaves the item unchanged.
- CLI, HTTP and the board expose proposing, showing, editing, accepting and rejecting a split, with request IDs and expected revisions.

## What stays the same

- One active implementation claim per work item ([ADR 0005](../../docs/adr/0005-factory-controller-owns-operational-state.md), [vision](../../docs/vision.md#work-claims)).
- Verification in clean worktrees, protected paths, holdout tests, the judge, bounded repair, and acceptance bound to the exact candidate.
- The orchestrate skill remains the explicit, hand-driven checkpoint workflow. A builder may use its harness's own subagents inside its one session; the factory sees one attempt.

## Not now

Each item has the observation that would reopen it.

| Not built | Reopen when |
| --- | --- |
| Checkpoints or several implementers inside one job | A trial item the person refuses to split fails because one builder session cannot finish it |
| Predicted write scopes | The trial shows conflicts rebuilding a meaningful share of parallel builds |
| Authorable pipelines and conditional QA | [Decision 01](../records/factory-parallel-runs/decisions/01-pipeline-shape.md) is resolved by a project need |
| Stacked builds on unmerged prerequisites | Waiting for merged prerequisites dominates lead time in the trial |
| More than one refinement or test-writing run per project | Parallel builds wait on them in the trial |

Isolation beyond a Git worktree, harness adapters, agent review and claims across machines belong to the sibling plans for sandboxes, harnesses, agent review and team scale.

## Verification

Each ticket ships with deterministic service, worker and, where the board changes, browser checks using fixture workers and real Git repositories. The checks cover two builds touching the same file, two touching different files, a restart with a queued merge, a conflict before and after approval, a split with dependents and a dropped criterion, and an interrupted split publication. Live providers are verified separately in the [trial](../records/factory-parallel-runs/issues/06-parallel-trial.md).

## Tickets

1. [Merge one change per branch at a time](../records/factory-parallel-runs/issues/01-merge-queue.md)
2. [Build several ready work items at once](../records/factory-parallel-runs/issues/02-parallel-builds.md), after 1
3. [Rebuild a change that no longer merges](../records/factory-parallel-runs/issues/03-rebuild-conflicting-change.md), after 1
4. [Propose and accept a split of a work item](../records/factory-parallel-runs/issues/04-split-work-item.md)
5. [Split a work item from the board](../records/factory-parallel-runs/issues/05-split-on-board.md), after 4
6. [Trial parallel builds and splits on a real backlog](../records/factory-parallel-runs/issues/06-parallel-trial.md), after 2, 3 and 5
