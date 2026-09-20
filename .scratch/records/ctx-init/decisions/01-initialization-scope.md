---
type: Decision
id: ctx-init-initialization-scope
title: Define initial ctx init capabilities and supported targets
decisionState: open
---

# Define initial ctx init capabilities and supported targets

The earlier proposal supports human callers and existing agents, preserves the chosen tracker, and prepares files for multiple harnesses. That proposal is parked until a second-project trial establishes what initialization actually needs.

## How to resolve this decision

Bootstrap the second project manually and use ctx for one real unfinished task. Record the files copied, the lines adapted, existing-file merges, harnesses used, and any tracker-access limitations in a friction log. Use those observations to narrow the draft issue and answer the questions below. Do not resolve them through another design debate.

If the trial does not exercise a harness, hook, external tracker, or repeated setup, defer that case rather than treating it as answered. This decision remains open until the trial findings have been reviewed.

## Questions

- Which skills and workflow capabilities must initialization install or adapt? Identify the required instructions and prompts, and whether any specific hooks are necessary for that workflow.
- Which Copilot client and which platforms must be tested alongside Codex and Claude Code? Distinguish documented file locations from verified compatibility.
- What completion can setup report when an existing tracker remains authoritative but ctx has no connector for it? The proposed boundary is working agent guidance with an explicit ctx retrieval limitation, but that boundary has not been confirmed separately.

## Observations from the Opsbase trial

The [friction log](../../../ctx-init/opsbase-trial.md) now supplies concrete answers to review:

- The trial used task-context, recovery-notes, its PROFILE.md, an adapted acceptance procedure, and three pointers in existing AGENTS.md. It needed a supplied binary and target-specific paths. No hooks were required.
- Codex on macOS was exercised. Claude Code, Copilot, platform combinations, symlinks, and copy fallback were not. They are deferred from the rewritten draft.
- GitHub remained authoritative. A labeled local snapshot retained the original retrieval and required manual freshness checking and two visible format adaptations. ctx assembled that snapshot and local documents; it did not retrieve live GitHub work.
- Existing instructions and the multi-context layout were preserved. The trial used a small additive merge, not a general reconciliation mechanism.

Local setup, task work, and verification have been exercised. Review these findings before resolving the first-release scope. The issue remains a draft; hosted Opsbase CI has passed; human code review remains outstanding. This decision stays open, with evidence available rather than awaiting another debate.

## Current direction after review

Use the [manual checklist and tracker-snapshot profile](../../../../docs/tracker-snapshots.md) for the next project. Keep command implementation parked until a second external project shows which adaptations repeat. The Opsbase evidence supports reusable guidance, but does not establish an installer contract. This decision remains open for that evidence rather than another debate.

The next trial uses Mukabi with Claude Code to change the harness axis. Follow the [paired recovery measurement](../../../../docs/tracker-snapshots.md#next-trial-and-recovery-baseline) with and without the checkpoint. The snapshot profile now defines the proposed missing-execution discriminator as a WorkItem with a valid authoritative `sourceURL`; the leniency itself remains unimplemented.

## Parked design direction

- Preserve an existing tracker and offer local files when none exists.
- Support multiple selected harnesses, preserving suitable existing layouts.
- Prefer shared skill content and links, with copies when links are unsuitable. Preserve differing existing contents for reconciliation.
- Keep the initial human handoff short and free of project-file writes. The existing agent performs project discovery and preparation.

## Context

- [Design interview and debates](../../../ctx-init/discovery.md)
- [Project initialization issue](../issues/01-agent-guided-project-initialization.md)

## Mukabi observation

The [Mukabi trial](../../../ctx-init/mukabi-trial.md) confirmed Claude skill discovery through individual symlinks while preserving the local roadmap. Shared preparation steps repeat, but the checkpoint condition took 30.39 seconds longer to reach its first task edit in one pair. Neither session emitted the requested plan before editing. Keep this decision open and init draft; do not infer a speed benefit or multi-tool installer requirements from these results.
