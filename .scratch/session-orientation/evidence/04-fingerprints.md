# Ticket 04 verification

Workflow actor: Codex coordinating agent. Verification finished at 2026-09-16T12:07:09.348176+00:00. No human approval is asserted.

The [retained verification record](02-04-verification.json) identifies the tested working tree based on `9666b19340834db9360c43c12dc2f40920e16c21`, with a SHA-256 manifest of source and fixture files, exact commands and results, toolchain, and retained reader digest. This source identity describes the actual verification run; a later commit or merge does not replace it.

The original [ticket](../../records/session-orientation/issues/04-expose-requirement-fingerprints.md) owns the criteria.

## Criteria observations

| Criterion | Evidence |
| --- | --- |
| 1 | Fingerprint values are exposed through Orient and literal CLI text/JSON assertions, distinct from whole-file SHA-256 and derived from captured body bytes. |
| 2 | Whole-body mutation tests cover introductory requirements, Scope, criteria, and dependency links; every exact H2 Comments and Acceptance section is excluded. |
| 3 | Conformance vectors cover parsed fences, nested/repeated/Setext sections, raw CRLF, whitespace, Unicode, and document order. |
| 4 | Full, shortcut, collapsed, and image references append only effective external raw definitions in first-use order, including multiline titles and excluded-section definitions. |
| 5 | Criteria vectors cover full ordered section spans and external reference definitions, ending at the next H1/H2 boundary. |
| 6 | Six literal vectors use independently authored retained byte strings, Python struct/hashlib framing, and an OpenSSL digest cross-check. No expected digest calls the production helper. |
| 7 | Eleven mutation cases establish requirement-sensitive hashes and stability for comments, current Acceptance links, and frontmatter execution/title edits. |
| 8 | Coverage includes fenced/nested/repeated sections, CRLF definitions, empty destinations, multibyte text, retained and excluded definitions, and changed whole-source hashes after harmless comment edits. |
| 9 | Missing criteria keep criteriaSHA256 null; malformed inputs retain identifying partial-result diagnostics. Fingerprints do not grant readiness or accepted completion. |
| 10 | At 2026-09-16T12:03:03.927237+00:00, authored ticket01 Acceptance with nine required snapshots and three retained evidence sources. Both subject hashes stayed unchanged after the current link was added. Validation is explicitly deferred to05. |
| 11 | Fingerprint authoring was exercised before ticket02 closed and without waiting for03. Ticket01 received the first record;02 and04 receive records during this closeout after evidence. Unfinished03 has no claimed acceptance. |

## Verification outcome

All retained commands passed. Real-project orientation returned 1 in both formats, complete=false and inventoryComplete=true. Reader commands left source files and Git state unchanged. Ticket01 acceptance preserves its original working-tree source identity. No acceptance validator is claimed to have run; current records await ticket05.
