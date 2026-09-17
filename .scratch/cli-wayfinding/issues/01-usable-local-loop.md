# What proves the local directing-and-resuming loop is usable?

Type: grilling
Labels: wayfinder:grilling
Status: resolved
Blocked by: None
Parent: [Direct and resume agent work with the local CLI](../map.md)

## Question

What observable end-to-end scenario and success criteria finish the local milestone? Use the backend/device scenario in the research as a candidate, then agree what the person sees, what the fresh agent receives, and which missing information must stay visible after interruption.

Determine whether evidence presentation belongs in the first usable loop or remains a later refinement. Preserve the selected boundary: skills author records and handoffs, humans direct pickup, and the CLI reports facts and assists resumption. Decide how to evaluate usefulness and authoring effort without inventing quantitative targets.

Inputs: [research findings](../research.md), [existing proposed trial](../../workflow-improvements/discovery.md#a-small-sequence-to-evaluate).

## Comments

2026-09-17: During the live grilling, the user endorsed reducing repeated explanation, reconstruction of unfinished work, uncertainty about what can proceed, and uncertainty about completion evidence as desired outcomes. These outcomes still need a bounded acceptance scenario.

2026-09-17: The user proposed `.context-cache/` for temporary working notes so the same or a new agent can continue after compaction or interruption. The user confirmed that deleting it may lose continuation convenience and cause repeated investigation, but must not lose agreed requirements or accepted evidence. Notes should be updated after meaningful progress, failed approaches, and new blockers rather than only at session end. A fresh agent should find the relevant notes when asked to continue a task and check them against current files before relying on them.

2026-09-17: The user confirmed that undecided questions can remain in these notes until decided. Record settled decisions in authoritative project records and link from the notes. Still to clarify: whether a question that blocks work must be represented in the authoritative blocker relationships before it is decided. Exact cache layout and retrieval remain for the handoff decision; no cache implementation was created here.

2026-09-17: The user confirmed the instruction/tool boundary: skills guide writing and maintaining working notes; `.context-cache/` stores temporary notes in projects using the product; the CLI helps locate and read relevant notes alongside current context. Detailed CLI behavior remains to be designed.

2026-09-17: The user answered yes to all three proposed continuation defaults. A question that blocks implementation is recorded against the affected work item immediately; detailed investigation and nonblocking exploratory questions may stay in the cache. A claim that tests passed without saved results or tested revision is a lead, not completion evidence; rerun relevant checks before relying on it for completion. When the user asks to continue and notes agree with current requirements, the agent briefly states its next action and proceeds without another plan approval, asking only for scope or architecture decisions not settled by the existing agreement. These are skill-guided behaviors; they do not imply that the CLI runs agents or tests.

## Answer

Resolved with the user on 2026-09-17 after a live grilling. The user confirmed the following finish line for the milestone.

### Acceptance scenario

Interrupt an agent partway through a real task. Start a fresh session and ask it to continue, supplying the task's ticket reference. Using the CLI and workflow skills, the new agent must be able to:

1. Find the working notes and current requirements without the user reconstructing the previous session.
2. Explain what was done, what remains, and what changed since the notes were written.
3. Show blocking questions and missing verification, including unsupported claims that tests passed.
4. Briefly state its next action and continue within the agreed scope without another plan approval. Ask the user for scope or architecture decisions that the existing agreement does not settle.
5. Keep recovery notes updated after meaningful progress, failed approaches, and new blockers so continuation does not depend on a graceful session ending.

Use these observable outcomes to assess the first version. No time-saving target or particular backend/device fixture is required by this decision; the trial must use a real task and retain what was observed. This is an agreed future evaluation, not a claim that current code already passes it.

### Notes and authority

Skills author temporary working notes in `.context-cache/` in a project using the product. Those notes can contain progress, investigation, failed attempts, and undecided questions. The CLI helps locate and read relevant notes alongside current project context; exact storage and command contracts remain for the later decisions.

An unresolved question that blocks implementation is recorded against the affected work item immediately, even if detailed investigation stays in the cache. Settled decisions belong in authoritative project records, with links from the notes. A fresh agent checks notes against current files before relying on them.

Deleting the cache may cause repeated investigation but must not remove agreed requirements, recorded blockers, decisions, or accepted evidence. Treat a note claiming that tests passed without saved results or a tested revision as a lead; rerun relevant checks before relying on the claim for completion.

### Milestone boundary

Include a basic view of verification gaps in the continuation experience. Defer a richer criterion-by-criterion evidence view and new criterion identity conventions. Keep existing acceptance semantics.

The human directs pickup. Skills maintain records and guide agent behavior; the CLI assists with facts and context. This milestone does not add managed work-record writes, exclusive implementation claims, an agent runtime, or test execution by the CLI. Jira remains an early follow-up.
