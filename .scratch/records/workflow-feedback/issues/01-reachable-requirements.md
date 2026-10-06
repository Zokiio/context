---
type: WorkItem
id: 6b4b8cfd-db3f-46d6-923e-115317a5af36
title: Check that a requirement is reachable before adopting it
triage: ready-for-agent
execution: completed
---

# Check that a requirement is reachable before adopting it

Recorded on 2026-10-06 from Axpilot retrospectives. Three consecutive runs proposed this rule independently: #272 proposal 5, #274 proposal 3, and #275 proposal 4. All three ran on Waymark 541e468, which Axpilot committed in its PR #276.

## Evidence

- **Axpilot #275.** The orchestrator wrote a speculative fallback into checkpoint 1's packet. Requirement R2 said: "if `turn/interrupt` fails, answer `{}` and warn". The provider source, readable at planning time, rejects an interrupt only when no turn is active. Repair 1 then adopted an "exactly once" race requirement from a Standards should-fix. The orchestrator did not first check the sessions service, which already ignores a terminal status for an inactive run. Checkpoint 1 took four review rounds, and the user dropped R2 at round 3. The two extra rounds cost about 41M subagent input tokens, and the repairs rewrote about 105 production and 133 test lines.
- **Axpilot #274.** The orchestrator found one `git status` call and proposed a Git flag to the user as the fix. The affected path runs `git diff`, so the change missed it and was reverted.
- **Axpilot #272.** A findings file told a repair to rename a describe block. The rename reached four tests outside the user's approved list, and the approval question that followed blocked the next repair for 4 h 54 m overnight. Another finding claimed a test spy would miscount, which it would not for that fixture.

The orchestrate skill says to investigate uncertain correctness concerns before treating them as findings. It sets no evidence bar for requirements the orchestrator writes into a packet or proposes to the user.

## Scope

Add the rule to orchestrate's "Check and correct an identified revision" section, as the retrospectives proposed. Before an error path, fallback, or reviewer's race finding becomes a packet requirement, a repair finding, or a proposal to the user, the orchestrator cites two things. The first is the dependency source that makes the case reachable. The second is the owning service behavior that makes it visible. Without both, the item stays an observation. Review axes, gates, and human review policy are unchanged.

## Acceptance criteria

- R1: Before an error path, fallback, or reviewer's race finding becomes a checkpoint requirement, a repair finding, or a proposal to the user, the orchestrate skill requires the orchestrator to cite the dependency source that makes it reachable and the owning service behavior that makes it visible.
- R2: An item that lacks either citation is recorded as an observation rather than a requirement, finding, or proposal.
- R3: The canonical template and generated development copy agree, and `go test ./internal/initialization ./internal/cli` passes.

## Blocked by

None

## Blocked by decisions

None

## Context

- [Domain glossary](../../../../CONTEXT.md)

## Comments

2026-10-06: Completed in [PR #37](https://github.com/Zokiio/context/pull/37) after independent Standards and Specification reviews and their renewals passed. Evidence and the current Acceptance record remain local under the tracker conventions; a fresh checkout reports acceptance unknown until they are restored together.

## Acceptance

[01-reachable-requirements-20261006.md](../acceptances/01-reachable-requirements-20261006.md)
