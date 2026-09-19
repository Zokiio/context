# Compact operator paths

Codex `/root` verified product and documentation revision `4d2b3822d311ecd99da4550b809fe158aca556c2` on 2026-09-19. The change implements the user's live-demo feedback about repeated absolute paths. The renderer is the only changed production Go file.

## Criterion coverage

| Ticket 08 criterion | Evidence and result |
| --- | --- |
| 1. Root once and authored goals | The compact header declares the record-store root once. Focused CLI tests preserve goal whitespace and paragraphs and reject the old source/reference dump. The live demo shows the same behavior. |
| 2. Actionable paths | Task rows retain relative selectors, including nested directories. Focused tests and independent real CLI probes cover same filenames, external sources, sibling-prefix boundaries, missing identity, ambiguous identity, and an unavailable project root. |
| 3. Useful provenance | Routine rows omit from/link details. Affected work retains identity, with a path when needed for disambiguation. Grouping/order tests remain passing, shared-source findings retain affected work, and distinct unresolved links retain referring paths and authored links. The cause grouping code is unchanged. |
| 4. Other output unchanged | Thirteen baseline/current CLI comparisons match stdout, stderr, and exit status byte for byte for detail, JSON, context, resume, and workspace reports. Tests also check that compact rendering does not mutate the result. |
| 5. Verification and delivery | Full test, race, vet, and build checks pass. The demo was rebuilt from the tested source and inspected through computer use. The shared spec, reader reference, and continuation guide describe the revised presentation. PR delivery and current acceptance observations are recorded separately. |

The [checks manifest](checks.json) identifies commands, environment, and source hashes. [Independent review](independent-review.md) records its probes and the corrected task-selector finding. [Parity results](parity.json) retain each invocation, exit, output size, and digest; [the driver](parity-driver.py) compares both binaries against the same files.

## Live demonstration

The local presentation at `http://127.0.0.1:56402/` ran the rebuilt CLI against the disposable `.cache/live-cli-demo/demo` project. Computer use selected Operator overview and observed `ctx orient`, exit 0, complete evaluation and inventory, one record-store root, relative task selectors, and the open decision's relative source path. A screenshot inspection confirmed the displayed result at the user's current browser width. The presentation and fixture are excluded from the product.

The same fixture produced [49 lines and 3,653 bytes before](demo-compact-before.txt) and [35 lines and 1,458 bytes after](demo-compact-after.txt). This is a measurement of that sample, not a general output-size guarantee. Both reports preserve the same authored goal, commitments, readiness, affected work, in-progress task, and shortlist.

## Reassessment basis

The shared specification and reader reference changed, so all seven earlier decisions require fresh requirement snapshots. Their historical tests and evidence retain their original tested revisions. The following assessment supplements that evidence; it does not relabel the earlier tests as new runs.

| Ticket | Effect of the revised requirements |
| --- | --- |
| 01. Shared capture | No capture, selection, authorization, budget, or schema changes. The new full suite and exact detail/JSON/context comparisons support the existing criteria. Default compact text intentionally follows revised ticket 02. |
| 02. Compact orientation | The declared record-store root and relative actionable paths replace the old repetitive source listings. The new tests, independent probes, and demo cover the revised first two criteria. Existing grouping, partial-state, argument, workspace, and read-only tests pass. |
| 03. No-note resume | Resume code and selection are unchanged. Exact no-note repo text/JSON comparisons and the full suite support the existing final behavior. The historical temporary nonempty-store restriction remains superseded by tickets 04 and 05. |
| 04. Retained checkpoint | Note validation and source comparison are unchanged. Exact retained-note demo resume comparisons and the full suite support the existing criteria. The historical single-root restriction remains superseded by ticket 05. |
| 05. Competing checkpoints | Graph, cache, comparison, and diagnostic code are unchanged. Existing conflict, interruption, limit, and concurrent-publication checks pass in the full suite. |
| 06. Recovery skills | Instructions and note profile are unchanged. Shorter operator paths remain usable ticket selectors. Historical instruction trials continue to support the criteria; no new claim of a fresh agent trial is made. |
| 07. Complete workflow | The guide now explains the compact path base. The original interrupted-task and 22-scenario evidence remains applicable to unchanged recovery behavior. The new source identity has passing full checks and exact report comparisons. Fresh prerequisite acceptance observations establish the reassessed dependency graph separately. |

The [independent reassessment review](reassessment-review.md) found no coverage gaps. [Delivery observation](delivery.json) records the updated draft PR description and rebuilt live demo. Acceptance remains an authored workflow judgment with no human approval asserted.
