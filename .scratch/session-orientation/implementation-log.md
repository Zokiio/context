# Session orientation implementation

This is a working coordination log. Immutable slice evidence and Acceptance records retain verification history; this changing log is not an acceptance snapshot.

## Plan

1. Establish project orientation, shared reading, CLI output, and authoring conventions.
2. Implement standalone shortlisting and requirement fingerprints independently.
3. Evaluate blocking decisions after shortlisting.
4. Validate direct prerequisite acceptance after decisions and fingerprints.
5. Evaluate recursive prerequisites and cycles.
6. Migrate records, retain the reader build, and finish preflight checks.
7. Use a fresh session for the unstarted real-ticket pickup trial.
8. Review against the PR1 baseline, fix findings, retain final evidence, and create the PR.

## Current work

- Branch: feat/session-orientation.
- Review baseline: ecb8f294144dab8bb36f2e74e6ad174bcce17ef1, the merged PR1 revision.
- Planning records committed as 5742063.
- Baseline `go test ./...` passed before implementation.
- Ticket 01 implementation is verified and execution is completed. Agent orientation_core owns the application operation and shared reading; orientation_cli owns CLI output and its tests; the coordinating agent owns authoring conventions, ticket states, integration, and evidence.
- The existing reader supplied ticket 01's full authored context. Both completeness fields were true.
- Ticket 01 was committed as 9666b19 after integration verification. Tickets 02 and 04 are in progress in parallel. Each agent received complete task-context JSON from that retained reader build.
- orientation_core owns eligibility and readiness for ticket 02. orientation_cli owns fingerprints for ticket 04. The coordinating agent added a CLI readiness fixture and is preparing immutable evidence and acceptance authoring. A separate reviewer is checking shared-reader compatibility against the first-reader contract.

## Review during ticket 01

- A separate workflow review found one P2 issue: the staged acceptance procedure would require recursive prerequisite checks before ticket 05 could close, although recursion belongs to ticket 06. The guide now keeps unsupported checks unknown through the full bootstrap, separates acceptance validity from prerequisite satisfaction, and retains the approved blocker order. Missing evidence and invalid records are not exempt.
- CLI agent reports its suite passing, including existing context fixtures and new orientation text, JSON, scope, partial-result, and injected-failure cases. Core source and limit cases now pass, including the repeated-goal and invalid-UTF8 budget regressions.

## Completion

Ticket 01 has [immutable verification evidence](evidence/01-overview.md). Structured acceptance is intentionally pending fingerprint authoring and validation in tickets 04 and 05. No unsupported check is reported as passed.
