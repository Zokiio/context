# Shared source capture verification

Actor: Codex /root, with implementation by /root/shared_capture using GPT-5.6 Sol at high reasoning. Observed on 2026-09-19 Europe/Stockholm.

Tested integrated revision: `7ab8d48`. The implementation agent tested identical product code at `4ce73bf` in its isolated worktree. This records local workflow verification, not human approval or device/runtime verification.

The implementation agent received every selected source from the task-context output for `cli-wayfinding/issues/01-share-captured-sources.md`. The reader was built from the initial implementation state `7b93da4`. Invocation used `context --bundle /Users/zoki/code/context/.scratch/records --ticket cli-wayfinding/issues/01-share-captured-sources.md --allow-source /Users/zoki/code/context`. Exit status was 0, with both completeness fields true and nine sources. Root reviewed the authored criteria and implementation before recording acceptance.

## Criterion results

1. Passed. `go test ./...` passes after integration. Existing standalone context, orientation, workspace, discovery, and setup fixtures retain their contracts. `TestSharedCaptureMatchesStandaloneReadersAndCancellationWritesNothing` also compares full standalone and composed reader values.
2. Passed. `TestSharedCapturePreservesBytesAcrossReaderPhases` captures the Project manifest, collects task context, then runs orientation using the same capture. The exported composition functions share scope and limits through `recordread.Capture`.
3. Passed. The same test changes both the ticket and requirement file between phases and verifies that identity and digests still describe the captured bytes. Usage equals four unique physical paths and their byte total. `TestCaptureReusesBytesAndRechecksRoleAuthorization` removes an already captured document, verifies retained bytes, and rejects its reuse as an out-of-bundle record.
4. Passed. `TestSharedLimitKeepsCompleteTaskContextAndPartialOrientation` preserves complete task context while marking orientation and inventory incomplete. `TestSharedLimitStopsNewSourcesAcrossReaderPhases` verifies that a byte-limit breach stops later smaller sources while keeping admitted facts available. Cancellation remains an operation error.
5. Passed. Temporary filesystem fixtures exercise mutation, removal, shared counting, limits, role authorization, cancellation, full result equality, and read-only behavior. No filesystem-wide atomic snapshot is claimed. The existing standalone regression suite passes.

## Commands and provenance

Root ran `go test ./...` and `go build -o /tmp/context-wayfinding-ctx ./cmd/ctx` at `7ab8d48`; both exited 0. The [retained test output](01-shared-capture-tests.txt) records the integrated suite.

The implementation agent also reported successful `go test ./...`, `go test -race ./...`, and `go vet ./...` at `4ce73bf`. Root's integrated test run is retained independently. Shared capture is a read-side refactor; it does not yet expose resumption or write checkpoints.
