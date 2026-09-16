# Project discovery implementation

The approved specification and six WorkItems own scope and acceptance criteria. This log records coordination and links to retained verification.

## Starting point

- Planning revision: `fc1741962ff95fc830e6fe836c6c79f002865c5a`.
- PR #3 merged as `1eb54f87242502020181b46504d9556e5bea6a6a` before implementation began. The implementation branch starts from that merge and will use a follow-up PR.
- Local branch: `feat/project-discovery-implementation`.
- Baseline `go test ./...` passed.
- Baseline `ctx context` returned exit 0, complete and traversalComplete for all six tickets.
- Baseline orientation returned a complete report without diagnostics. Ticket 01 was ready. Tickets 02 through 06 were blocked by their declared unfinished prerequisites.

## Coordination

The coordinator owns integration, commits, ticket execution, evidence, and Acceptance records. Implementers receive the full assembled task context before starting. Agents share this worktree and receive disjoint file ownership. Independent reviewers do not approve their own implementation.

Ticket 01 is assigned to `/root/configuration`. The agent owns configuration parsing, saved path values, and focused tests under `internal/discovery`. A separate read-only integration analysis checks existing reader seams while that work proceeds.

Ticket 02 starts after ticket 01 has valid, fresh acceptance. CLI integration and setup can proceed independently after ticket 02 is accepted. Workspace navigation follows CLI acceptance. Documentation and the independent-project/workspace trial follow tickets 03, 04, and 05.

Acceptance decisions record criterion-specific results and tested source identity. Execution labels alone do not unblock dependencies. Full tests, race checks, vet, build, the trial, and independent standards/specification reviews remain required before the implementation is ready for review.

## Integration constraints

The read-only integration analysis confirmed that discovery belongs before the existing explicit-scope reader calls. Direct bundle selection bypasses configuration before any registry reads. Configuration and resolver code share path semantics. Setup prepares a proposed document before printing its summary and applying an atomic update. Workspace navigation receives an explicit declaration and does not call project orientation.

Legacy CLI fixtures that intentionally use manifestless or malformed bundles will use `--bundle`. Discovery receives separate filesystem tests. Tests isolate the home directory. The source authorization implementation and reader requests remain unchanged.

The starting repository orientation inspects 97 files. Acceptance checks use explicit limits of 500 files and 8 MiB as retained evidence grows; reader defaults remain unchanged.

## Ticket 01 accepted

The configuration slice adds strict shared/personal parsing and saved path identity helpers under `internal/discovery`. The final focused race suite passed 81 cases. Independent review found alternate-case filesystem identity and non-directory parent traversal issues; both were corrected and the independent probes passed. [Criterion evidence](evidence/01-configuration-after-review.json) records the tested source manifest. [Acceptance observation](evidence/01-read-configuration-implementation-observation.json) reports valid, fresh acceptance and passing prerequisite checks. Ticket 02 is now ready.

## Ticket 02 in progress

`/root/configuration` owns resolver implementation and temporary-filesystem tests. The agent received refreshed complete task context from a reader built at `389cc88`. The resolver returns scope and provenance without output or writes. Selected discovered record stores require the Project marker; direct bundle mode remains a CLI bypass. An internal replacement-config seam will let setup check proposed declarations through the same selection rules.

`/root/setup` is preparing the persistence design without code changes. Its implementation waits for ticket 02 acceptance.

Under ticket 02's implementation claim, `/root/integration_design` owns `resolve_conformance_test.go`. That independent test slice covers workspace declaration comparisons, physical ancestry, reserved personal configuration, and boundary/stopping behavior. `/root/configuration` owns production resolver code and the remaining resolver tests.

## Ticket 02 accepted

The resolver slice passed 147 discovery test cases under the race detector, with no skipped cases. The independently authored conformance slice passed. Review corrected a prefix heuristic that could discard an unreadable registration hiding a symlink into the start directory. The setup proposal seam validates future configuration without writing or cleaning away broken target components. [Criterion evidence](evidence/02-resolver-verification.json) retains the actual source identity. [Acceptance observation](evidence/02-resolve-scope-implementation-observation.json) reports valid, fresh acceptance and passing checks. Tickets 03 and 05 are now ready for parallel implementation.

## Tickets 03 and 05 in progress

`/root/integration_design` owns CLI discovery integration and migration of existing reader-focused fixtures to direct bundle mode. `/root/setup` owns setup preparation, atomic persistence, setup-specific CLI code, and tests. Both received refreshed complete task context from a reader built at `9d04352`. CLI environment callbacks keep cwd and home lookup lazy and support injected input and terminal detection. Setup registration remains separate until its implementation is ready.

Ticket 03 covers workspace selection/provenance and protection against invoking a project reader for workspace scope. Ticket 04 supplies workspace navigation rendering. `/root/configuration` is preparing that rendering design without implementation while ticket 04 remains blocked.

## Ticket 03 accepted

The reader CLI slice passed 120 focused race test cases, including setup factory tests compiled in the shared worktree. Root reviewed scope selection, output isolation, source authorization, and unchanged filesystem snapshots. Original direct path traversal now validates before canonicalization so missing path components cannot silently select another directory. [Criterion evidence](evidence/03-cli-verification.json) retains the complete tested source identity. [Acceptance observation](evidence/03-integrate-cli-implementation-observation.json) reports valid acceptance and passing checks. Ticket 04 is ready for implementation.

## Ticket 04 in progress

`/root/configuration` received refreshed complete task context from the verified ticket03 working-tree reader. The agent owns workspace membership inspection and the separate CLI renderer. Root owns registration in the orient command. Membership availability is shallow directory access and never evaluates work readiness.

## Ticket 05 accepted

Setup passed 59 focused race test cases after root registered the command and verified setup followed by a selector-free reader invocation. The plan preserves unrelated content and file permissions, reports replacement removals, checks other declarations, and applies an atomic update with concurrent-change detection. [Criterion evidence](evidence/05-setup-verification.json) retains the tested source identity. [Acceptance observation](evidence/05-connect-records-implementation-observation.json) reports valid acceptance and passing checks. The persistent lock sidecar coordinates writers; dry-run and identical setup create no lock.

A [test-name erratum](evidence/03-cli-evidence-erratum.json) corrects criterion03 evidence from TestScripts to the actual TestCLI driver. The original executed stdout is retained unchanged. Final reassessment will include the erratum.

## Directory-input regression

Workspace review identified that opening a FIFO as a directory can block on this macOS runtime, including through os.OpenRoot. Root added an original-path Stat/IsDir guard before opening directory handles, with timed CLI regressions for bundle, selector, source-root, and setup inputs. Focused resolver/setup/direct-path race checks pass. Final source verification and acceptance reassessment will cover this shared change.

## Ticket 04 accepted

Workspace navigation passed the combined discovery/workspace/CLI race suite: 342 cases, zero skips. Output preserves membership order and separate same-ID record stores, reports shallow availability, and emits safe POSIX commands and structured argument arrays with member-specific roots. [Criterion evidence](evidence/04-workspace-verification.json) and [acceptance observation](evidence/04-workspace-navigation-implementation-observation.json) retain the tested source identity and valid acceptance with passing checks. Ticket 06 is ready.

## Ticket 06 verification

Root read all 11 sources in the refreshed task context. `/root/setup` owns the documentation slice under root's claim; root owns the trial and final acceptance. Full `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, and build checks passed against one unchanged program-source manifest. [The trial](evidence/06-trial.json) passed all 37 scenarios across this repository, a separate non-Git project, a moved shared checkout, personal aliases, and a workspace. Reader and dry-run snapshots are unchanged. Setup changed only the intended registration, persistent sidecar, and immediate parent metadata. The caller's personal registry was not used or changed.

The generated repository `.context/config.md` is retained. Its lock sidecar is ignored. The trial verifies both complete orientation with explicit larger budgets and the expected partial report at preserved default budgets. Documentation and independent review remain outstanding before final acceptance.
