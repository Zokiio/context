---
type: WorkItem
id: f49ee233-f7d0-4266-92d1-f354df72a604
title: "Include linked project documents"
triage: ready-for-agent
---

# 02: Include linked project documents

**What to build:** A caller receives a ticket together with the project documents explicitly linked from its Spec and Context sections. The result explains each inclusion and retains available context when a selected document is missing, unsupported, or outside the permitted scope.

## Scope

This slice supports documents inside the selected bundle. It establishes relationship extraction for the recursive-blocker slice, but does not yet traverse blocker tickets or authorize external directories. Do not expand selection through ordinary document prose.

## Acceptance criteria

- [ ] Use Goldmark v2 on the Markdown body after frontmatter separation. Recognize exactly level-two Spec, Blocked by, and Context headings. Sections end at the next level-one or level-two heading; nested subsections remain inside them and repeated sections combine in document order.
- [ ] Recognize inline and reference-style Markdown links in relationship sections. Ignore images, links inside code, fenced heading examples, and apparent headings or links inside YAML metadata. Unresolved reference-style links in recognized sections produce identifying diagnostics and incomplete context.
- [ ] Include the starting ticket, then its Spec sources, then its Context sources, preserving link order within each relationship kind. A ticket without a Spec section remains supported.
- [ ] Resolve relative document links from the referring document's directory and leading-slash links from the selected bundle root. Do not interpret a leading slash in a document link as the operating-system root.
- [ ] Select the whole target file for a fragment link, preserve the original link including its fragment in the inclusion reason, and do not check whether the heading exists.
- [ ] Use resolved absolute paths for source identity and containment, with os.Root enforcing permitted reads. Deduplicate repeated links and symlink aliases while retaining every distinct referring file and original link. Keep separate files with identical contents as separate sources.
- [ ] Each included document retains its full original text and matching SHA-256 digest. Full OKF validation is not required, and linked context documents without frontmatter remain consumable.
- [ ] Do not recursively follow links in included Spec or Context documents. A parent explicitly linked as Context is included; unlinked parents, siblings, and the project manifest are not automatically selected.
- [ ] A missing or unreadable Spec or Context document makes complete false and returns exit code 1, while traversalComplete remains true if the ticket's relationships were fully discovered. Return all other available sources.
- [ ] HTTP links in relationship sections and links outside the currently permitted bundle produce diagnostics and incomplete results. Do not fetch remote content. Web links in ordinary prose remain references.
- [ ] Behavioral tests cover section boundaries, both link forms, raw-source preservation, fragments, symlink aliases and escapes, stable source order, and partial results. Add only the CLI checks needed to demonstrate linked context and its exit behavior.

## Blocked by

- [01: Read an OKF ticket as JSON context](01-read-okf-ticket.md)

## Spec

- [Context reader specification](../../../context-reader/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Project bundle decision](../../../../docs/adr/0001-one-okf-bundle-per-project.md)
- [Markdown relationship decision](../../../../docs/adr/0002-work-relationships-in-markdown-sections.md)
- [Tracker authoring profile](../../../../docs/agents/issue-tracker.md)
