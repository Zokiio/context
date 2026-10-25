---
type: WorkItem
id: b144d30f-79a1-4c03-a4bc-ab5cce87f091
title: Tell a repair why the previous attempt failed
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#repair-brief)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- A repair attempt's assignment carries a brief about the implementation attempt it repairs, built only from retained records. The brief lists each failed or unknown required check on the previous candidate, with its key, command, exit observation and a bounded tail of its `command-N.log` artifact. It also lists the protected paths the previous candidate changed, the previous builder's final message from `agent-result.json`, and the person's summary when they requested changes.
- The brief has a fixed size limit per log tail and in total, and says when it truncated. It is labelled as observations, not requirements. The admitted requirements, context and protected paths are unchanged and supplied in full.
- `AgentPrompt` renders the brief in its own section after the rules. A first attempt has no such section.
- The brief is retained with the repair attempt, and the attempt's detail in History shows what the repair was told.
- Deterministic service and worker tests cover three cases. A fixture builder whose check fails gets that check's key and log tail on repair. A protected-path rejection names the path. A changes-requested review passes its summary through.
- `docs/factory-platform.md` describes what a repair receives.

## Comments

- 2026-10-25: Joakim chose a failure brief before native session resume. The agent-review plan adds its findings as another section of this brief.
