# Real-task acceptance trial

## Outcome

On 2026-09-16, agent `/root/milestone_trial` used the [task-context skill](../../.agents/skills/task-context/SKILL.md) for acceptance verification of [ticket 05](../records/context-reader/issues/05-bound-context-collection.md). The skill delivered the real ticket, its specification, all four recursive blockers, and five shared context documents. The agent consumed every delivered source's full `text`, `path`, and `reasons` from the JSON before inspecting the implementation. Large output was read in smaller JSON-derived sections to avoid tool-output truncation. The user did not gather those sources manually.

The default result contained 11 whole sources totaling 83,702 bytes, with exit status 0, `complete: true`, `traversalComplete: true`, and no diagnostics. All recorded boundary checks passed. Verification found no reader defect, so it added no application code or regression test.

## Reader revision and prerequisites

The trial used the uncommitted reader based on `0c74f92506a67242247a4e76a40785561a561bb3`, with Go 1.27.1 on darwin/arm64. At trial time, `go.mod`, `go.sum`, `cmd/`, `internal/`, and the task-context skill were untracked. The base commit alone does not identify the tested implementation.

The [trial manifest](acceptance-trial-manifest.json) records SHA-256 digests for every file under `cmd/` and `internal/`, both module files, and the skill. It also records the binary digest, each delivered source's path, byte count, digest, and reasons, and every invocation's observed result and diagnostics.

The coordinating agent reported successful full application and CLI checks before delegating this trial. Ticket 05's delivered source independently records `go test ./...`, `go vet ./...`, `go test -race ./...`, and `go build -o /tmp/context-ticket05 ./cmd/ctx` passing against this working tree. Those are prior verification results, not commands rerun by this trial agent.

This agent built the trial binary successfully and reran the focused behavioral checks successfully:

```sh
go build -o /tmp/context-reader-acceptance-ctx ./cmd/ctx
go test ./internal/taskcontext -run 'Test(Limits|FirstBreach|LimitInside|Oversized|DefaultLimits|NegativeApplication)' -count=1
```

The focused test command returned status 0 and `ok github.com/Zokiio/context/internal/taskcontext`.

## Invocation and observed limits

The working directory was `/Users/zoki/code/context`. Every trial invocation used this explicit scope:

```sh
/tmp/context-reader-acceptance-ctx context \
	--project /Users/zoki/code/context/.scratch/records \
	--ticket context-reader/issues/05-bound-context-collection.md \
	--allow-source /Users/zoki/code/context
```

The following rows show flags appended to that command. Defaults mean 100 files and 1,048,576 source bytes. The manifest preserves the full argument arrays. Successful and incomplete JSON invocations emitted no stderr. Invalid invocations emitted no stdout.

| Additional flags | Exit | Complete | Traversal complete | Sources | Source bytes | Diagnostics |
| --- | --- | --- | --- | --- | --- | --- |
| `defaults` | 0 | true | true | 11 | 83,702 | none |
| `--max-files 11` | 0 | true | true | 11 | 83,702 | none |
| `--max-bytes 83702` | 0 | true | true | 11 | 83,702 | none |
| `--max-files 11 --max-bytes 83702` | 0 | true | true | 11 | 83,702 | none |
| `--max-files 10` | 1 | false | false | 10 | 77,416 | 1 |
| `--max-bytes 83701` | 1 | false | false | 10 | 77,416 | 1 |
| `--max-files 1` | 1 | false | false | 1 | 5,649 | 8 |
| `--max-bytes 5649` | 1 | false | false | 1 | 5,649 | 8 |
| `--max-bytes 6206` | 1 | false | false | 1 | 5,649 | 8 |
| `--max-bytes 5648` | 1 | false | false | 0 | 0 | 1 |
| `--max-files 7` | 1 | false | false | 7 | 60,732 | 2 |
| `--max-files=0` | 2 | no JSON | no JSON | n/a | n/a | stderr error |
| `--max-files=-1` | 2 | no JSON | no JSON | n/a | n/a | stderr error |
| `--max-files=not-an-integer` | 2 | no JSON | no JSON | n/a | n/a | stderr error |
| `--max-bytes=0` | 2 | no JSON | no JSON | n/a | n/a | stderr error |
| `--max-bytes=-1` | 2 | no JSON | no JSON | n/a | n/a | stderr error |
| `--max-bytes=not-an-integer` | 2 | no JSON | no JSON | n/a | n/a | stderr error |

A Python assertion script executed these invocations with `subprocess.run`, parsed each result, and checked its exit status, stderr, completeness, source order, whole file bytes, and SHA-256 digests. It compared every returned source against its current file bytes and confirmed that all selected files still matched the initially delivered bytes after the runs. The script returned status 0. Its disposable copy and JSON captures were retained under `/tmp/context-reader-acceptance-*`; this document and the manifest are the durable evidence.

## Delivered source order

The default result selected these repository-relative paths in order. The manifest retains their absolute paths and full digests.

1. `.scratch/records/context-reader/issues/05-bound-context-collection.md`: 5,649 bytes.
2. `.scratch/context-reader/spec.md`: 27,737 bytes.
3. `docs/vision.md`: 18,553 bytes.
4. `CONTEXT.md`: 3,319 bytes.
5. `docs/adr/0001-one-okf-bundle-per-project.md`: 557 bytes.
6. `docs/adr/0002-work-relationships-in-markdown-sections.md`: 337 bytes.
7. `docs/agents/issue-tracker.md`: 4,580 bytes.
8. `.scratch/records/context-reader/issues/03-follow-blocker-tickets.md`: 5,084 bytes.
9. `.scratch/records/context-reader/issues/04-allow-external-context.md`: 5,348 bytes.
10. `.scratch/records/context-reader/issues/02-include-project-documents.md`: 6,252 bytes.
11. `.scratch/records/context-reader/issues/01-read-okf-ticket.md`: 6,286 bytes.

The shared specification and each shared Context document have five inclusion reasons, one per ticket. Ticket 02 has two blocker reasons, from tickets 03 and 04. Each source appears once. Exact limits returned the same JSON data as the default result, including all reasons.

## Acceptance findings

The agent inspected `internal/taskcontext/context.go`, `options.go`, `source.go`, `limits_test.go`, `internal/cli/cli.go`, and the CLI limits testscript after receiving the task context. These checks confirmed the following behavior on the real task:

- File limit 11 and byte limit 83,702 each fit the entire selection. Using both limits also succeeds.
- File limit 10 and byte limit 83,701 preserve the first ten whole sources, totaling 77,416 bytes. Ticket 01 is the identified first breach.
- File limit 1 and byte limit 5,649 preserve only ticket 05. The specification breaches first, followed by seven `source_omitted` diagnostics for the five Context documents and direct blockers 03 and 04.
- Byte limit 6,206 leaves 557 bytes after ticket 05, enough for ADR 0001. The reader still stops at the larger specification and does not skip ahead to fit that ADR.
- Byte limit 5,648 excludes the starting ticket entirely. The result has no sources and one identifying limit diagnostic.
- File limit 7 stops at blocker 03 and reports blocker 04 as known pending. Tickets 02 and 01 are absent from the diagnostics because their relationships have not been discovered.
- Every limit failure sets both completeness fields to false and returns exit status 1. Known pending diagnostics retain the referring path and authored link. No run claims an exhaustive count of undiscovered omissions.
- Zero, negative, and malformed values for both flags return status 2 with an error on stderr and no JSON.

Existing behavioral tests cover multibyte and JSON-escaped source text, cycles, repeated sources, default boundaries, retained earlier errors, and undiscovered descendants. The focused rerun passed. The real-task checks add evidence of the actual authored dependency graph and source bytes.

## Relevant documents without authored selection links

`AGENTS.md`, `docs/agents/domain.md`, and `.agents/skills/task-context/SKILL.md` informed this verification through repository guidance and the assigned workflow. They have no Spec or Context selection links from the selected tickets. The agent read them separately as instructions. `docs/agents/issue-tracker.md` was initially read as required repository guidance and was also delivered through authored Context links.

The external `technical-writing` and `unslop` skills guided this evidence document. They are workflow instructions, not ticket-selected sources. Code and tests were inspected separately to verify implementation. None of these separate reads is a reader defect or evidence of automatic relevance discovery.

The trial establishes delivery of the authored task context and the observed acceptance behavior. It does not establish automatic discovery of every relevant document, work readiness, or full OKF validity. The source manifest records the ticket before the coordinating agent adds the evidence link and completes its remaining checklist entries. Later ticket edits change its digest and exact byte boundaries.


## Independent review follow-up, 2026-09-16

An independent standards review found no actionable issues. An independent specification review found one P2 defect: an undefined Blocked by reference left `traversalComplete` true although its blocker could not be discovered. The reader now retains the relationship kind during reference diagnosis and marks traversal incomplete for that case. Undefined Spec and Context references retain their existing behavior. Application and CLI regressions cover both full and collapsed references and continued discovery of available branches.

The specification reviewer independently verified the fix with six CLI probes and the application and CLI suites, with no remaining actionable findings. Full race tests, vet, and build passed on Go 1.27.1; the full test suite also passed on the declared Go 1.25.0 minimum.

The coordinator reran all 17 real-task checks against the updated reader and current ticket snapshot. All passed with 11 selected sources totaling 84,813 bytes. The [review verification manifest](review-verification-manifest.json) records the exact invocations, outcomes, delivered source digests, and source-file digests for this revised implementation. The original trial and manifest remain historical evidence. Subsequent evidence notes change ticket bytes, so the recorded exact-fit limits apply to the captured snapshot.
