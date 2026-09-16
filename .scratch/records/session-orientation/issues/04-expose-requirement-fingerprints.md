---
type: WorkItem
id: 8888abfc-d72c-42c0-8a28-08c7235d68c6
title: "Expose reproducible requirement fingerprints"
triage: ready-for-agent
---

# 04: Expose reproducible requirement fingerprints

## What to build

An acceptance author can obtain the current ticket and criteria fingerprints from orientation, even before any acceptance record exists. The fingerprints detect changed requirements while allowing routine comments and current-acceptance bookkeeping.

## Scope

Implement the confirmed whole-body coverage and exact version-1 encoding in the specification. This slice exposes values and supports explicit acceptance-record authoring. Semantic acceptance validation belongs to ticket 05. Tickets 02 and 03 are not prerequisites.

## Acceptance criteria

- [ ] Return fingerprintVersion, ticketSHA256, and criteriaSHA256 for each work item when computable, even without recorded acceptance. Present the values in text and JSON, clearly separate from whole-file source digests. Derive every value from the already captured source bytes.
- [ ] The ticket fingerprint covers the Markdown body after frontmatter removal, excluding every exact level-two Comments and Acceptance section with its nested content. Retain introductory requirements, Scope, criteria, and dependency relationships.
- [ ] Use parsed Markdown structure for exclusions and section spans. Fenced heading-like text must not act as a section boundary. Preserve repeated sections, document order, whitespace, Unicode, and original line endings.
- [ ] Include effective raw reference definitions used by retained content when their bytes are outside retained spans. Append each additional definition once, in first-use order, including its line ending. Changes to a retained reference's meaning must affect the fingerprint even when its definition appears in Comments.
- [ ] Compute the criteria fingerprint from complete raw Acceptance criteria section spans in document order, plus required external reference definitions. Follow the specification's exact section-boundary rules.
- [ ] Use the specified domain tags, zero separator, unsigned 64-bit big-endian counts and lengths, and lowercase SHA-256 digests. Independently checked conformance fixtures cover both encodings rather than deriving expected values with the production fingerprint function.
- [ ] Public-operation tests show that introductory requirements, Scope, criteria, dependency links, and used reference definitions change the relevant fingerprint. Ordinary comments, current Acceptance links, and frontmatter-only execution or title edits do not change accepted requirement bytes.
- [ ] Tests cover fenced headings, nested and repeated sections, CRLF, multibyte text, reference definitions already retained, and definitions in excluded sections. A comment edit changes the whole-source digest while leaving the requirement fingerprint unchanged when no retained meaning changed.
- [ ] Missing or malformed inputs retain identifying diagnostics and explicit unavailable values where computation is impossible. Fingerprint availability does not imply readiness or accepted completion, and must not remove unrelated partial-result diagnostics.
- [ ] Exercise the documented authoring procedure on orientation slices already verified at this point, using retained evidence and actual decision times. Record current acceptance links and snapshots after final requirement edits; preserve any earlier decisions and original tested revisions. Do not claim that the new acceptance validator has run.
- [ ] Do not wait for unfinished tickets 02 or 03 to complete this slice. Record which completed slices received acceptance records and which still lack evidence or await validation. Retain this slice's own verification evidence for its closeout.

## Blocked by

- [01: Show project orientation and work inventory](01-show-project-orientation.md)

## Blocked by decisions

None

## Spec

- [Session orientation specification](../../../session-orientation/spec.md)

## Context

- [Domain glossary](../../../../CONTEXT.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
