# Controlled fixture review for ticket 07

I reviewed the ticket, `workflow_probe.py`, `checkpoint_probe.py`, `check_resume_contract.py`, and `07-controlled-results.json`. This is controlled fixture evidence only. It does not establish the separate fresh-agent documentation trial.

The probe does make useful assertions. It creates a note whose local test statement lacks retained evidence, then checks that the task has no acceptance and that the operator report says completion is not established. It records a separate Acceptance for the prerequisite, with retained evidence and an older tested revision, then checks that the acceptance remains valid after the application advances. It also checks separate record and working directories, an independent non-Git project, distinct namespaces for the same task ID in two projects, cache deletion, and byte-for-byte plus Git-status preservation around the read commands.

## Findings

1. The controlled result has no identity for the CLI under test. `workflow_probe.py` writes only the fixture application's historical revision at line 233. The result commands identify the executable as `/tmp/context-wayfinding-ctx`, and the JSON records hashes of stdout, but neither the binary hash nor the repository revision and working-tree state that produced that binary. The fixture result therefore cannot show that it tested the final source identity required by this ticket.

   Record the CLI binary SHA-256 and the source revision plus dirty-state or diff hash when building and running the probe. Put those values in `07-controlled-results.json`, alongside the exact probe command.

2. The limit cases mostly assert only an exit status. `current-source-limit` at line 200 only requires exit 1. `cache-entry-limit` at lines 201 through 202 adds the weak extra check that there are no candidates. Neither verifies the corresponding incomplete fields, omission details, or a limit-specific diagnostic. `explicit-limit-overrides` at line 203 accepts exit 0 without asserting that the recovered report is complete. An unrelated error that returns the expected exit can satisfy these cases.

   Assert the exact incomplete state and limit diagnostic for each failure. For the source limit, check `context.complete` and the reported omitted source. For cache inventory, check `inventoryComplete`, `graphStatus`, and the cache-limit diagnostic. After explicit overrides, assert `report.complete` is true. Preserve those selected fields in the result artifact if it is meant to be reviewable without rerunning the temporary fixture.

3. The combined workflow probe never drives the cache byte limit into failure. It uses `--max-cache-files 1` but does not run `--max-cache-bytes 1`; `--max-cache-bytes 16777216` appears only in the successful override. `checkpoint_probe.py` has a byte-limit unit fixture, but `workflow_probe.py` neither runs it nor includes its result in `07-controlled-results.json`.

   Add a `workflow_probe.py` case with a deliberately too-small `--max-cache-bytes`, assert the bounded/incomplete cache inventory and its diagnostic, then retain that case in the controlled result. Alternatively, include the separately executed checkpoint-probe result and source identity in this ticket's evidence.

4. The open blocking decision is set up correctly, and the probe checks that `checkpoint-task` is blocked. It does not assert that `fixture-runtime-decision` is the reported reason. The nearby assertion for the device/runtime gap matches literal prose injected into `task.md`, so it does not establish that the authoritative decision link, rather than another condition, caused the blocked result.

   Assert the decision's stable ID and source path in the blocker or diagnostic returned by `resume` or `orient`. Then resolve or remove that decision while retaining the accepted prerequisite and assert that the task is no longer blocked. That isolates the decision's authority from the note's advisory content.
