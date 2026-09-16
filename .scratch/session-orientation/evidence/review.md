# Session orientation independent review

Two independent workflow agents reviewed the application implementation at `0808609e8de4ad1d80c52cadd8fd0b6fca31ac0d` against merged PR1, `ecb8f294144dab8bb36f2e74e6ad174bcce17ef1`. The fixed point resolved and the three-dot diff was nonempty before review. The captured command was `git diff ecb8f294144dab8bb36f2e74e6ad174bcce17ef1...HEAD`. The frozen range contains planning commit `5742063` and implementation commits `9666b19`, `c69d818`, `b266094`, `b6befb1`, and `0808609`.

The review used AGENTS.md, the product vision, domain glossary and ADRs, tracker and acceptance conventions, the session-orientation specification, and its eight tickets. Standards also applied the code-review skill's twelve Fowler smell heuristics as judgment calls. Neither reviewer edited repository files. Migration and the actual fresh-session trial were explicitly pending at this review boundary.

## Standards

Reviewer: workflow agent `/root/orientation_standards_review`.

- P2, documented-standard violation. A completed prerequisite with an explicitly empty Acceptance criteria section could retain fresh acceptance and release its dependent. Tracker conventions require at least one nonempty criterion. The isolated probe produced a blocked prerequisite with valid acceptance, a passing dependency edge, and an eligible dependent. The fix applies the criteria check when evaluating prerequisite satisfaction. Acceptance provenance and matching snapshot facts remain separately visible.
- P3, judgment call, possible Duplicated Code. Decision and prerequisite aggregation separately combined statuses and removed repeated findings. No incorrect result was demonstrated from the duplication. The shared `mergeChecks` helper in `checks.go` now owns that behavior for both callers.

The reviewer ran the full Go suite in an archived checkout of the frozen commit. After the fixes, it independently reran the original empty-criteria probe, all three new criteria regressions, and focused decision, accepted-chain, and cycle tests. The dependent is now blocked with an empty shortlist while the unchanged acceptance remains visible as valid. Both findings are resolved. Standards has zero remaining findings.

## Spec

Reviewer: workflow agent `/root/orientation_spec_review`.

- P2. Repeated project flags silently selected the last bundle, contrary to the specification's requirement to reject invalid or ambiguous flags. An isolated CLI probe with two valid bundles returned exit 0 and the second project. Orientation now rejects repeated project, file-limit, byte-limit, and format flags before invoking the operation. Allowed-source roots remain repeatable, and task-context flag behavior is unchanged.
- P2. Fresh acceptance released work behind an empty criteria section, contrary to the required nonempty criterion and known-failure behavior. This reviewer independently reproduced the same criteria defect found by Standards. Prerequisite evaluation now retains that failure through direct and transitive edges.

The reviewer inspected schemas, profiles, fingerprints, scope, budgets, decisions, prerequisite graphs, CLI behavior, and shared-reader changes. Its follow-up ran twelve independent CLI probes against a rebuilt reader: eight repeated/conflicting flags, repeated allowed roots, unchanged context flag behavior, the original accepted-empty case, and a valid-criteria counterpart. All passed, and both findings are resolved. Spec has zero remaining findings.

## Fix verification and acceptance reassessment

[Core regression observations](review-core-regressions.json) retain the original failing tests and their successful rerun. The [CLI failing output](review-cli-flags/red.txt), [focused successful output](review-cli-flags/green.txt), and [full CLI output](review-cli-flags/cli-suite.txt) preserve that agent's reported red/green sequence. [Independent Spec verification](review-spec-fix-verification.json) retains exact invocations, stdout/stderr, the reader digest, and source hashes for all twelve probes.

The [program preflight](07-preflight-verification.json) identifies the actual tested working tree, all source and fixture digests, exact commands, Go versions, and the retained reader. Full normal and race tests, vet, build, and Go 1.25.0 tests, vet, and build passed. The independent review build has the same SHA-256 as this retained build. These observations concern the recorded sources; they do not substitute a later commit for the tested revision or assert human approval.

The earlier acceptance judgments for orientation tickets 01, 05, and 06 need reassessment because the added regressions exposed missing cases in their implementation. The coordinating agent will author new decisions that retain the earlier records and evidence and identify this corrected implementation's actual tested sources. Record migration and the real pickup trial have their own evidence and remain separate from this code-review result.

Standards: two findings resolved, zero remaining. Spec: two findings resolved, zero remaining. The empty-criteria defect appears in both independent reports.
