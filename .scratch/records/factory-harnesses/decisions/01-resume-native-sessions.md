---
type: Decision
id: ae8b64ac-f2c7-4cc0-b7d6-d63d91eef8e4
title: Resume native sessions on repair only if briefed repairs keep failing
decisionState: open
---

# Resume native sessions on repair only if briefed repairs keep failing

Every repair starts a fresh native session. On 25 October 2026 Joakim chose to give repairs a failure brief first and to decide on resume from evidence.

## Question

Should a repair resume the previous attempt's native session, so the builder keeps what it learned?

## What the answer unblocks

[Resume the builder's session on repair](../issues/10-resume-repair-session.md).

## Evidence needed

- At least ten briefed repairs on real work, with the repair pass count from [ticket 02](../issues/02-repair-pass-count.md).
- For the briefed repairs that failed, a reading of their transcripts. It counts failures caused by lost context, such as redoing work, undoing earlier changes or repeating a mistake already shown in the brief, apart from failures with other causes.

Resume is worth building only if lost context explains a meaningful share of failed briefed repairs. Otherwise resolve this decision as "no resume" and mark the ticket `wontfix`.
