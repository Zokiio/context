# Independent implementation review

Reviewed revision: `223dc1a21ffea9b3bfdd341dd6f8bbfa6df356fe`.
Fixed point: `fc1741962ff95fc830e6fe836c6c79f002865c5a`.
Diff: `git diff fc1741962ff95fc830e6fe836c6c79f002865c5a...223dc1a`.
Recorded on 2026-09-16 UTC by Codex coordinating agent `/root`.

The two reviewers did not implement the changes. Their reports remain separate below. Ticket 06 acceptance and final reassessment were pending this review.

## Standards

Reviewer: Codex subagent `/root/standards_review`.

No hard documented-standard violations found.

One judgment call: possible **Duplicated Code** in `internal/cli/scope.go:107`, `internal/workspace/navigation.go:121`, and `internal/discovery/resolve.go:437`. Each repeats the sequence `os.Stat(path)`, `!info.IsDir()`, `os.OpenRoot(path)`, and `root.Open(".")`. The first two also repeat `ReadDir(1)` and EOF handling. Consider sharing the directory-access check while retaining caller-specific diagnostics and canonicalization. This would keep future special-file and permission fixes together. This is a smell heuristic, not a required change.

The acceptance snapshot digests match their retained sources. All 70 program-source hashes in the final test, race, vet, build, and trial evidence match the reviewed revision.

## Spec

Reviewer: Codex subagent `/root/spec_review`.

Two findings:

- **P2: Reserved special files can hang discovery and setup.** `internal/discovery/config.go:95`, `internal/discovery/setup_write.go:28`, and `internal/discovery/resolve.go:464` open inputs before rejecting nonregular files. FIFOs at local/personal `config.md` and a discovered `project.md` made five subprocess probes time out without output, including setup dry-runs. This leaves the specified failure handling incomplete: "A malformed or unreadable registry fails discovery" and "2 for invocation, discovery, or operation failure", from specification lines 55 and 68. Reject special files before opening them.
- **P2: An unrelated invalid binding blocks valid local discovery.** `internal/discovery/resolve.go:311` rejects every canonicalization error. A personal registration whose directory is `old-checkout/subdir`, where `old-checkout` is now a regular file, prevents orientation of an unrelated valid project. `ENOTDIR` establishes that this binding cannot contain cwd. This exceeds the required diagnostic condition for scopes that "might contain the start directory" and contradicts "Unrelated missing target locations do not block selection", from specification line 60.

No scope creep or further conformance defects found in the reviewed parser, resolution, CLI, navigation, setup, and documentation. All six task-context reads were complete; orientation confirmed valid acceptance for slices 01 through 05. The reviewer did not repeat the full test/race suite. Both fixes require rechecking at their resulting revision.

Standards: zero hard violations and one advisory judgment. Spec: two P2 findings, both awaiting fixes at this observation.

## Retained reproduction observations

The specification reviewer ran five disposable subprocess probes with a two-second timeout. Local config orientation, personal config orientation, local setup dry-run, personal setup dry-run, and Project marker orientation all timed out with no stdout or stderr. Probe output was retained locally at `/tmp/context-spec-review.VhVlrB/fifo-d9yn68fz/results.json`; this document retains the observation independently of that disposable path.

The unrelated-binding probe used a regular file as an ancestor of an unrelated personal checkout directory. A valid local project's orientation returned status 2 with an ENOTDIR diagnostic. The fixture was `/tmp/context-spec-review.VhVlrB/bindings-jk46gb59`.

Root added regressions before the corresponding fixes. `go test ./internal/cli -run '^TestDiscoveryFilesRejectFIFOsWithoutWaitingForAWriter/shared_config_orient$' -count=1 -timeout=15s` failed after three seconds with `discovery waited for a FIFO writer`. `go test ./internal/discovery -run '^TestResolveUnrelatedNonDirectoryBindingsDoNotBlockSelection$' -count=1` failed for implicit, project, and workspace selection with the unexpected ENOTDIR diagnostic. The tests exercise the production CLI and resolver using temporary filesystems.
