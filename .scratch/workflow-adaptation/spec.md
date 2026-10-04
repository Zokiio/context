# Adapt agent workflow skills to Waymark

Status: proposed on 2026-10-04. The user asked to adapt four public skills from [mattpocock/skills](https://github.com/mattpocock/skills/tree/main/skills/engineering), `tdd`, `implement-spec`, `retro`, and `code-review`, to Waymark and the projects it prepares, rather than copy them. The [workflow design](../workflow-improvements/orchestration-design.md) remains the source of confirmed orchestration policy. This specification proposes changes to it. The linked decisions hold the choices that need the user.

## Problem

On 2026-10-04, two Axpilot sessions ran the installed orchestrate skill in parallel:

| Measure | [Axpilot #254](https://github.com/Zokiio/Axpilot/issues/254) | [Axpilot #264](https://github.com/Zokiio/Axpilot/issues/264) |
| --- | --- | --- |
| Time to a verified result | 38 minutes | About 4 hours, then 2 more for a design change |
| Subagents | 18 | 52 |
| Orchestrator input tokens, mostly cache reads | 103M | 161M |
| Production lines | +62 −27 | +1,201 −59 |
| Test lines | +551 −63 | +5,311 |
| Outcome | Merged as PRs #270 and #271 | 19 local commits awaiting the user |

- **Test volume.** Both orchestrators told their expectation authors to write new test files and leave existing ones untouched. Each new file built its own harness. In #264, `session_attention_test.go` has 679 lines for three tests, and two packages each implement the same permission-admission fake. Standards reviews passed this code.
- **Orchestrator context.** #264 wrote 554 KB of dispatch packets for 20 KB of checkpoint files. Its checkpoint 4 implementer packet was five times the size of its checkpoint file. Runner and reviewer reports returned to the orchestrator in full.
- **Useful gates.** Reviews found two real focus defects in #264, and the combined-result verifier found a stale design document in #254. The user's first look at #264, a demo after 15 commits, removed the per-row action buttons that every reviewer had verified against the specification.
- **Unmeasured cost.** Nothing in the workflow reported these numbers. They came from a manual inspection of the transcripts, which also found a #254 doc repair that was never re-reviewed and records split across three locations. The [trial conclusions](../workflow-improvements/orchestration-conclusions.md) already list review effort as unmeasured.

## What Waymark supplies

The upstream skills infer information that Waymark records:

- `ctx context` returns a work item's task context with source identities. Upstream `code-review` searches commit messages for an issue reference instead.
- `ctx orient` reports readiness, blockers, and blocking decisions. Upstream `implement-spec` derives its own frontier from a ticket graph.
- Checkpoint files hold scoped requirements and gates. They can travel by path and digest where upstream passes pointers to tickets.
- Completion evidence and acceptance decisions retain their author and the revision they concern.

Projects receive Waymark guidance only through `ctx init`. They do not have the upstream skills. Orchestrate's `VERIFICATION.md` already carries condensed testing and review procedures, so prepared projects need neither.

## Adaptation rules

- Use ctx readers and records wherever an upstream skill searches, guesses, or asks.
- Ship guidance through the `ctx init` templates, with explicit invocation, generated development copies, and drift tests.
- Proposed project guidance becomes mandatory only after the user approves it, following orchestration design choices 5 and 18.
- Keep Waymark's evidence model. Every result retains its observer and evaluated revision.
- Judge each change by the next real run rather than by further design.

## Proposals

### Dispatch checkpoint work by pointer

From `implement-spec`: communicate through pointers rather than duplicated text. See [ticket 01](../records/workflow-adaptation/issues/01-dispatch-by-pointer.md).

The checkpoint file becomes the implementing agent's packet. It holds the outcome, requirements, approved seams, gates, permitted paths, and dependency contract, as design choice 3 already requires. A dispatch then names the role, the checkpoint path and digest, the starting or evaluated revision, shared project guidance by path and digest, and the findings file for a repair. The exclusion of the whole plan, sibling checkpoints, and other workers' conversations stays. The orchestrator still retains each dispatch.

This replaces the skill's rule that "a pointer list alone is insufficient". That rule protected scoped delivery. Separate checkpoint files already provide it, and a supplied packet never established filesystem isolation.

### Return short summaries

From `code-review`: keep each report short. See [ticket 02](../records/workflow-adaptation/issues/02-short-summaries.md).

Runners, reviewers, and the combined-result verifier write their full report to a local evidence file. They return a summary of about 400 words or fewer, stating the verdict and evaluated revision, citing criterion identifiers for each finding, and naming the evidence file. Full reports remain available for repair dispatches and acceptance without entering the orchestrator's context.

### Review sessions through a retrospective

From `retro`: improve the environment for future runs. See [ticket 03](../records/workflow-adaptation/issues/03-retrospective.md), which follows [decision 02](../records/workflow-adaptation/decisions/02-retrospective-distribution.md).

An explicitly invoked `retro` skill reviews named sessions of one work item. It reads the agent transcripts, from Claude Code or Codex, and asks for their location when it cannot identify them. It reads the work item's records through the installed task-context procedure, plus any orchestration plan, checkpoint files, progress, decisions, and retained reports.

It reports measures with their sources: wall time, subagents, token use, production and test line changes, review findings that led to repairs, and points where the user intervened. A measure it cannot obtain appears as a gap.

Each proposal names where it belongs:

- Project guidance, in the project's own instruction or standards files.
- An automated check in the project. A mechanical mistake gets a check rather than a written rule.
- A follow-up in the project's tracker.
- Waymark feedback, for gaps in Waymark itself, such as records split across locations. This keeps the user's project separate from the tool.

The retrospective changes nothing until the user approves a proposal. Approved guidance records its source and scope. The report is local evidence and stays out of Git.

### Write checkpoint tests in the implementer's loop

From `tdd`: red before green, one vertical slice at a time, at agreed seams. See [ticket 04](../records/workflow-adaptation/issues/04-tests-in-implementer-loop.md), which follows [decision 01](../records/workflow-adaptation/decisions/01-checkpoint-test-authorship.md).

Upstream `tdd` names horizontal slicing, writing all tests first and then all implementation, as an anti-pattern. Orchestrate does this by design: a separate expectation author writes a checkpoint's tests before a different agent implements it, under design choice 16. This proposal moves test writing into the implementing agent's loop, at the seams the checkpoint file names and the user reviews with the plan. Independence moves to the Specification review. It checks that every requirement has a test at an agreed seam and that expected values come from the specification or worked examples rather than restating the implementation.

## Not adopted

- Parallel implementers on a ticket frontier, from `implement-spec`. Coordinating several writers needs implementation claims, which the vision defers. The user's manual parallel run of #254 and #264 is recorded here as an observation.
- One review at the end in place of checkpoint gates, also from `implement-spec`. The checkpoint reviews in #264 found two real defects.
- Separate `tdd` and `code-review` skills in prepared projects. Orchestrate's verification procedures carry what those projects need.
- Fixed `CODING_STANDARDS.md` and `AGENTS.md` conventions, from `retro`. Proposals use each project's existing files.

## Order and measures

Tickets 01 and 02 need no decision and can proceed in either order. Ticket 03 follows decision 02, and ticket 04 follows decision 01. Each ticket is one PR.

The next orchestrated run judges these changes. Compare its test-to-production line ratio, orchestrator input tokens, and defects found per gate with the table above. Once ticket 03 lands, the retrospective produces these measures.
