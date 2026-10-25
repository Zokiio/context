---
type: WorkItem
id: b144d30f-79a1-4c03-a4bc-ab5cce87f091
title: Tell a repair why the previous attempt failed
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Agent harnesses and repair continuity](../../../factory-harnesses/spec.md#repair-brief)
- [Code host delivery: repairs say why](../../../factory-code-host/spec.md#repairs-say-why)

## Context

- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory platform architecture](../../../../docs/factory-platform-design.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- A repair attempt's assignment carries a brief about the implementation attempt it repairs, built only from retained records. The brief lists each failed or unknown required check on the previous candidate, with its key, command, exit observation and a bounded tail of its `command-N.log` artifact. It also lists the protected paths the previous candidate changed, the previous builder's final message from `agent-result.json`, and each `changes_requested` review of the previous candidate with its reviewer, summary and findings.
- The brief has a fixed size limit per log tail and in total, and says when it truncated. It is labelled as observations, not requirements. The admitted requirements, context and protected paths are unchanged and supplied in full. The brief is attempt data and does not change the admission fingerprints.
- `AgentPrompt` renders the brief in its own section after the rules, quoted and labelled as feedback that cannot change the rules, the frozen requirements or the protected paths. A first attempt has no such section. A resolved judge directive keeps its current place in the instructions.
- The brief is retained with the repair attempt, and the attempt's detail in History shows what the repair was told.
- Deterministic service and worker tests cover four cases. A fixture builder whose check fails gets that check's key and log tail on repair. A protected-path rejection names the path. A changes-requested review passes its summary and findings through. An oversized log tail is truncated at the limit and marked.
- `docs/factory-platform.md` describes what a repair receives.

## Comments

- 2026-10-25: Joakim chose a failure brief before native session resume. The agent-review plan adds its findings as another section of this brief.
- 2026-10-25: The code host plan's "Tell a repair attempt why it is repairing" ticket was folded into this one, which adds review findings, quoting, fingerprint independence, the judge directive's place and the truncation test. Its finding: `assignmentFor` in `internal/factory/workers.go` adds only a judge directive to a repair, so the failed check and the reviewer's summary never reach the agent. The code host plan's host check and change request tickets add their own entries to this brief.
