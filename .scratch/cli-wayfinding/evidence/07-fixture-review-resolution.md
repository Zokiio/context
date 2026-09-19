# Controlled verification review resolution

Actor: Codex /root. Date: 2026-09-19. Source: `bba08b9b3bebdfad55c1ac794fd7fd186ea97a28`.

The independent [fixture review](07-fixture-review.md) found four evidence gaps in the original controlled run. The original drivers/results remain preserved. A separate [version-2 driver](07-controlled-v2/workflow_probe.py) addresses those findings; the three imported helpers are byte-identical copies of the originals.

The root inspected the new assertions and ran the driver with `--build-from /Users/zoki/code/context`. All 22 cases passed. The [run manifest](07-controlled-v2/manifest.json) records the exact invocation and all retained file digests. The [results](07-controlled-v2/results.json) record the built binary hash, build command, source revision, clean product-source status, before/after build source manifests and selected report fields.

1. The driver builds its own tested binary, records its SHA-256, and requires unchanged source identity before and after the build. The final run used the same committed product source as the final Go checks.
2. Limit cases assert incomplete current context or recovery inventory, the expected graph status, and diagnostic codes at the affected source or cache path. The override asserts complete evaluation. The result retains these fields rather than relying only on an exit code.
3. A separate cache-byte-limit case returns exit 1 with incomplete inventory and `recovery_limit_exceeded` at a fixed note path. Its selected diagnostics are retained alongside the cache-entry and current-source limits.
4. The decision case asserts its stable ID, path, open state and attributed `open_decision` reason. Resolving that Decision makes the task ready; restoring it makes the task blocked again while the accepted prerequisite stays unchanged.

These corrections strengthen controlled verification; no product code changed. The separate real task trial retains the fresh agent's actual behavior. The historical fixture result is an executed local arithmetic check, not a product backend or device test. The broader fixture outcome is an explicit dependent WorkItem and remains unstarted/blocked while backend acceptance is absent. No new parent relationship or runtime verification is inferred.

The controlled quarantine example exercises reader behavior after a sequential fixture writer stops. Full procedural writer attribution, collision refusal, unknown/live-writer refusal and archive/reset metadata are verified by the accepted [ticket 06 filesystem trial](06-recovery-instructions.md).
