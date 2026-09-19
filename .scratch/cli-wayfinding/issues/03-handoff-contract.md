# What does a handoff retain, and how does a session select it?

Type: grilling
Labels: wayfinder:grilling
Status: resolved
Blocked by: 01
Parent: [Direct and resume agent work with the local CLI](../map.md)

## Question

What minimum skill-authored handoff can the CLI consume to support the agreed resumption scenario? Decide identity, location and selection, observed requirement sources, source hashes or retained text, code checkout/revision, completed and unfinished work, evidence, and useful failed attempts.

Work within the [agreed finish line](01-usable-local-loop.md#answer). It establishes disposable `.context-cache/` working notes and continuation from a ticket reference. This ticket decides the detailed representation and lookup without reopening that boundary.

Account for multiple handoffs, moved records, and uncommitted source observations. Choose what must be retained to explain changes, since a digest alone cannot recover old content. Keep mutable continuation material separate from authoritative requirements and immutable accepted evidence. Decide whether context-delivery reporting means assembled output or needs any host-reported loading evidence; never imply model comprehension.

Inputs: [research findings](../research.md), [handoff proposal](../supporting/workflow-improvements.md#make-interrupted-work-cheap-to-resume), [glossary](../../../CONTEXT.md).

## Comments

2026-09-17: The user confirmed one current recovery note per task, maintained by skills in `.context-cache/` and found using the ticket reference. It records the current approach and relevant requirement references, completed and remaining work, checks and their results and tested revision, open questions, useful failed approaches, and the next intended step. The agent checks current requirements and files before proceeding.

2026-09-17: The user confirmed that a missing note triggers reconstruction from project records and code, with uncertainty stated. The agent proceeds where the next action is clear and asks only when missing information prevents a sound decision. If two sessions leave conflicting notes, preserve both and report the conflict rather than treating the newest note as authoritative. Detailed conflict representation and checkout scoping remain to be specified.

## Answer

Resolved with the user through the live discussion on 2026-09-17.

### Recovery note and selection

Skills maintain one current recovery note per task within the current checkout's `.context-cache/`. A ticket reference is sufficient to locate the relevant note. Separate checkouts keep separate notes, even for the same task; do not automatically mix another checkout's observations into continuation.

The note retains:

- Current approach and relevant requirement references.
- Completed and unfinished work.
- Checks run, their results, and the revision checked.
- Open questions and useful failed approaches.
- The next intended step.

Use the existing stable project and work-item identities when specifying lookup. Record paths and checkout information locate observations; they must not silently substitute for identity. Exact filenames and serialization are implementation-specification details, subject to these selection rules.

### Previous requirement text

Retain copies of the requirement documents the agent used alongside the note, with their source references and observed content identities. This supports explaining previous versus current wording, including uncommitted requirements. Do not copy the entire repository.

The retained material records an earlier observation, not current authority. Refresh requirements and inspect current files before relying on a continuation step. Deleting the cache may cause repeated investigation but must not delete authoritative requirements, decisions, blockers, or accepted evidence, as established by the [milestone finish line](01-usable-local-loop.md#answer).

### Missing or conflicting notes

If there is no note, reconstruct what is available from project records and code, state uncertainty, and proceed where the next action is clear. Ask the user only when missing information prevents a sound decision.

If sessions leave conflicting notes for the same task in one checkout, preserve both and report the conflict. Do not infer that the latest timestamp identifies the correct account. One current note is the normal case, not permission to overwrite competing observations. The resumption prototype must show the ambiguity and how it affects continuation.

### Responsibility and limits

Skills author and maintain notes and retained requirement observations. CLI lookup and reporting assist continuation; they do not launch agents or establish exclusive ownership of work. Treat assembled or returned context as evidence of what the tool provided, not proof that a model understood it.

This decision establishes the handoff behavior. The resumption prototype and implementation specification must make the storage and reporting contracts concrete, including bounded snapshot reads and detection of incomplete or conflicting observations. No managed write operation, retention daemon, cross-checkout search, or automatic conflict merge is introduced by this decision.
