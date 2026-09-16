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
