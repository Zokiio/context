# Review each verified candidate with an independent agent

Approved plan, 2026-10-25. Not started.

## Why

Only a person reviews a change today. Two paths leave a gap:

- Under `--after-build when_untested` or `never`, the autopilot approves and merges a change once its checks pass, and nobody reads the diff (`approvesByPolicy` in `internal/factory/board.go`). Locked tests check the criteria. Nothing checks scope, readability or the project's standards.
- When a person approves, the approval view lists the criteria no locked test checks and leaves the rest of the diff to them.

The judge (`internal/factory/dispute_judge.go`) only settles disputes the builder raises about frozen tests or requirements. It never sees a candidate that passed.

No factory trial has shown yet that an agent reviewer catches what the checks miss. The Axpilot orchestrate runs on 2026-10-04 ran the two-axis review on every checkpoint, and it verified the spec. The person's first demo still changed the UI. So this plan measures what the reviewer finds and what the person still catches after it (ticket 05), and leaves a second harness out until those numbers exist.

## Decisions

The user decided three on 2026-10-25:

1. Claude Code runs the first reviewer, in a fresh read-only session that has none of the builder's context. A reviewer command file lets any harness plug in, and every review records its harness, model and session so metrics can compare harnesses later. Codex stays out for now. It has only had its arguments built, and it reports no cost.
2. Blocking findings start a repair without the autopilot, the way the judge routes an implementation defect. The controller acts as the job's coordinator, inside the repair limit and deadline fixed at admission.
3. Under `--after-build when_untested` or `never`, the autopilot approves only when the agent review of the exact candidate finished with no blocking findings. Advisory findings don't stop it. A running, failed or missing review asks the person.

The existing commitments settle the rest:

- The review runs after every required check passes and before the person's approval. Reviewing a candidate that fails its checks wastes the spend.
- A reviewer failure is never a pass. SWE-AF turns a reviewer exception into an approved, nonblocking review ([coding loop](https://github.com/Agent-Field/SWE-AF/blob/4d3dfe719e5ce9e2e8cd454722295bbe6512542f/swe_af/execution/coding_loop.py#L309)), and when its loop runs out with a nonblocking last review it reports `COMPLETED_WITH_DEBT`. Neither path exists here.
- Agent findings stay apart from a person's review. The vision says to keep automated review findings distinct from a teammate's approval.
- A person still makes two decisions per change. The agent review shows up inside the existing Approve decision.

No vision commitment changes, so there is no ADR. The ask policy only gets stricter, since delegated approval gains one condition and no path lets an agent approve. The judge already routes repairs without the autopilot.

## Behavior

### When it runs

Admission records whether the installation had a reviewer configured. The job keeps that flag for its lifetime, as it keeps its protected paths. Jobs admitted without it, including every job admitted before this change, behave as today.

When the last required check passes, the job becomes `ready_for_review` as now. In the same transaction the controller records one agent review for that candidate and requirement fingerprint, then runs the reviewer command once per axis, `standards` and `spec`, in parallel and in fresh processes. Each process gets 10 minutes and never runs past the job deadline.

The reviewer works in a clean worktree at the candidate. It may read files there and must leave them unchanged. The controller checks that the worktree is unchanged when the process exits.

### What the reviewer gets

Each axis receives:

- its axis name;
- the admitted sources with their digests, the base and candidate revisions, and the requirement fingerprint;
- the complete patch from base to candidate, read from Git objects. A patch over the judge's 2 MiB limit fails the review as too large to review;
- the required checks and their results on this candidate;
- for the spec axis, the locked tests' coverage and the criteria no locked test checks, when the job has locked tests;
- the project's reviewer instructions, frozen at admission like the builder's. They name the project's standards documents;
- for the standards axis, the smell baseline from the [code-review skill](../../.agents/skills/code-review/SKILL.md).

It never gets the builder's transcript, final message or progress notes, the other axis's result, or the review of an earlier candidate. Every candidate gets a full, fresh review, as the [orchestrate verification brief](../../.agents/skills/orchestrate/VERIFICATION.md) requires for checkpoint reviewers.

### What the reviewer returns

Each axis returns an explicit verdict, `pass` or `fail`, a summary, and its findings. A finding has a severity, a summary, a location (the changed path and line or hunk, when there is one) and a source. The source is an admitted criterion number on the spec axis. On the standards axis it is a standards document path at the candidate or the name of a baseline smell.

A finding is blocking when the reviewer shows a criterion is missing, partial or implemented wrong, demonstrates a defect, or shows a violation of a documented project standard. Everything else is advisory: baseline smells, unrequested behavior, uncertain correctness concerns, preferences, and any doubt about the requirements themselves. The reviewer cannot change intent, so a doubt about a requirement goes to the person and never to the builder.

The controller rejects output that breaks this contract, and the review fails, when:

- the axis does not match or the verdict is missing;
- the verdict is `fail` with no blocking finding, or `pass` with one;
- a blocking finding has no source, cites a criterion the admission doesn't have, cites the smell baseline, or cites a standards file absent at the candidate;
- it exceeds 30 findings per axis or 16,000 characters in any field;
- it has unknown fields or trailing data.

The review's outcome is `failed` if either axis failed, otherwise `blocking` if either axis has a blocking finding, otherwise `clear`.

### Failure is never a pass

Each of these records `failed`, with the reason and whatever log and cost the reviewer reported:

- the reviewer exits nonzero, crashes, times out or runs out of budget;
- its output breaks the contract;
- the patch is too large;
- the worktree changed;
- no reviewer command is configured for a job admitted with one;
- the service restarts during the review.

On restart the controller marks running reviews failed with "the service stopped before the agent review finished" and starts nothing. Nothing reruns a failed review automatically. A failed review starts no repair and never satisfies delegated approval.

Stopping the job cancels its reviewer processes. A result that arrives after the candidate changed, the job left `ready_for_review` or the requirements changed is kept and starts nothing.

### Where findings go

- `clear`: the job waits for the person's approval, or the autopilot approves under decision 3.
- `blocking`, when a repair is possible: in the transaction that records the result, the controller starts a repair as the job's coordinator. A repair is possible when repairs are left, the deadline hasn't passed, no execution is uncertain, the requirements are unchanged and the coordinator still has `coordinate`. The repair draws on the same admitted limit as failed checks, two by default (`startBuild` uses `min(2, maxRepairAttempts)`). Its repair brief carries the blocking findings in their own section, attributed to the agent reviewer, with the reviewed candidate. The new candidate goes through the required checks and a fresh agent review.
- `blocking`, when no repair is possible: the person gets the Approve decision with the blocking findings first. They can approve over them or reject the change. Nothing is accepted with debt.
- Advisory findings never start a repair. The person sees them with the change, and Run and History keep them.

The repair limit is the only loop bound. SuperPlane bounds its PR feedback the same way, with a per-PR attempt count (default 3) that resets after a passing run ([activity model](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/models/factory_pull_request_activity.go)). It never checks the PR head before pushing a fix. Here every review and repair binds to the exact candidate.

### The person's approval

- The service never refuses a person's approval because an agent review is running, failed or blocking. The agent proposes and the person decides.
- `reviews` stays human-only. The five `--finding` kinds keep their meaning: what the person caught. The approval view now asks what the checks and the agent reviewer missed.
- Every review event, whether a person's or delegated, names the agent review current at that moment, so the record shows what the approver had in front of them.
- Agent reviews are operational records in the controller store, like disputes. The Acceptance record does not change.

### Board and Run

The board gets no new column or decision:

- While the review runs, the card sits in Build, running, with "Reviewing the change" at step 3.
- A repair the review started shows as the existing "Fixing" card, with a fact giving the number of problems.
- A clear review leaves the existing Approve card, with the fact "Agent review: nothing blocking" and a count of notes when there are advisory findings.
- A blocking review with no repair left puts the card in Approve, needing the person, and its line gives the number of problems.
- A failed review puts the card in Approve, needing the person, with "The agent review failed. Read the change yourself."
- "Approving for you" appears only on a clear review.

Run adds an Agent review step between "Run the checks" and "Approve the change". It shows each axis's verdict and findings, and the model, turns, time and cost. History keeps every candidate's review. The card and job cost include the reviewer's reported cost.

The approval view shows the agent's findings in their own section, blocking first, each with its axis, location and source. "Check this yourself" stays separate. An agent reading an untested criterion does not take it off the person's list.

### Configuration

- `serve` and `solo` accept `--review-command-file`, a JSON argv that reads the reviewer input on stdin and writes the axis output on stdout. Without a reviewer command, jobs are admitted without agent review.
- `solo` configures `ctx factory reviewer` when `claude` is installed, with `--review-model` (default `sonnet`), `--review-budget-usd` per axis (default `1`) and `--no-review`.
- The adapter runs Claude Code in print mode with only Read, Grep and Glob, no session persistence, and a JSON schema for the output, as the judge adapter does.
- The Agents settings gain reviewer instructions next to the refiner, test author and builder.
- The agent harnesses plan turns the refiner, test author and judge into per-project role profiles. When those land, the reviewer registers as one more role, as the judge does. This plan doesn't wait for them.

### Metrics

Insights adds agent reviews by outcome, blocking findings by axis, repairs the reviewer started, and the reviewer's cost within agent cost. It also counts the person's findings on changes the agent review cleared. That last number says whether the reviewer is worth its cost.

## A repair doesn't know why it runs

When a person clicks "Send back for a fix", `Review` stores their note as a `changes_requested` review and the browser starts a repair. `assignmentFor` in `internal/factory/workers.go` adds only a resolved judge decision to the repair builder's instructions. The person's note never reaches the builder, and neither does a failed check's output.

The agent harnesses plan (`.scratch/factory-harnesses/spec.md`) owns the fix, a repair brief that tells the attempt about failed checks, protected-path rejections, the previous agent's final message and the person's note. Ticket 04 adds the agent reviewer's blocking findings as one section of that brief and no second channel. It stays `needs-triage` until the repair-brief ticket exists to link as its blocker.

## Not in this plan

- A second harness for the reviewer. Revisit once ticket 05 has numbers. Harness adapters, and Codex's missing dollar cap, belong to the agent harnesses plan.
- Rerunning a failed review on the same candidate. Add it if trials show failures are common. Until then the person approves by hand.
- The repair brief itself, which the agent harnesses plan owns.
- Agent review evidence in the Acceptance record. Add it only when an audit needs it.
- Review comments and CI results from a code host. Those belong to the plan-pr-ci work.

## Verification

Deterministic subprocess reviewers exercise a clear review, a blocking one, an advisory-only one, and every failure case: nonzero exit, timeout, invalid and inconsistent output, an oversized patch, a changed worktree, a missing command, a restart mid-review, a stop, and a late result after the candidate changed. Tests cover the delegated-approval gate for each outcome, the automatic repair and its limit, a person approving over blocking findings, and the board and Run states. A fake `claude` binary covers the adapter's arguments and parsing. Report one live Claude review separately from fixture results. Keep generated trial data and evidence out of Git.

## Tickets

1. [Review each verified candidate with an independent agent](../records/factory-agent-review/issues/01-agent-review-stage.md)
2. [Run the agent review with read-only Claude Code](../records/factory-agent-review/issues/02-claude-reviewer.md)
3. [Show agent findings where the person approves](../records/factory-agent-review/issues/03-approval-findings.md)
4. [Send blocking agent findings to a bounded repair](../records/factory-agent-review/issues/04-review-repair.md)
5. [Measure what the agent review catches](../records/factory-agent-review/issues/05-review-metrics.md)

01 is ready now. 02 and 03 follow 01. 04 needs 01 and the agent harnesses plan's repair brief, and 05 follows 04. Until 04 lands, a blocking review goes to the person. Tickets 01 and 02 together give a working reviewer that gates delegated approval, and 03 makes its findings readable where the person approves.
