---
type: WorkItem
id: 275621c7-0090-4d6b-95ab-04b343e9141e
title: "Resume a task without recovery notes"
triage: ready-for-agent
execution: unstarted
---

# Resume a task without recovery notes

## What to build

Expose the read-only resume operation from command-line selection through refreshed task/project facts and readable or JSON output when the selected task has no recovery observations.

## Slice boundary

This first vertical resume slice intentionally handles no-note reconstruction only. It does not claim to satisfy the final retained-note or conflict cases.

## Acceptance criteria

- [ ] Resume enforces the specified ticket, selector, checkout, JSON, current-source-limit, and cache-limit flag contracts, including scalar repetition, positional arguments, workspace rejection, and no noninteractive prompts.
- [ ] Project and WorkItem identities are validated before cache lookup. Missing or ambiguous identity retains available current facts with unknown recovery selection and a partial report.
- [ ] Cache scope follows the selected binding or marker directory, with explicit checkout override; direct bundle access requires explicit checkout. Relative checkout paths use invocation cwd. Separate record stores, unavailable selected checkouts, aliases, non-Git directories, and multiple projects remain correctly scoped.
- [ ] The application composes current task context and orientation through the shared capture/budget from the prerequisite ticket. Text and the exact task-resumption JSON envelope expose current sources, blockers, reported acceptance, completeness, and attribution without a second readiness predicate.
- [ ] An absent or empty task observation store returns absent recovery, a valid empty graph, empty observation/candidate arrays, and an unavailable baseline without pretending that sources are unchanged. Complete current evaluation can return exit status 0; partial current facts return 1 and invocation/operation failures return 2.
- [ ] Nonempty observation stores are rejected with an explicit intermediate-operation failure until the next slice, never misreported as absent or complete. This temporary limitation is documented and tested, and is removed by the following recovery slices.
- [ ] CLI and filesystem fixtures cover text/JSON, direct/discovered scope, namespace separation, missing cache, unavailable identity, source-limit behavior, source containment, and no writes to files, cache, configuration, or Git.

## Blocked by

- [Share captured sources between readers](01-share-captured-sources.md)

## Blocked by decisions

None

## Spec

- [Local orientation and resumption](../../../cli-wayfinding/spec.md)

## Context

- [Product vision](../../../../docs/vision.md)
- [Domain glossary](../../../../CONTEXT.md)
- [Reader contracts](../../../../docs/readers.md)
- [Discovery contracts](../../../../docs/discovery.md)
- [Acceptance procedure](../../../../docs/agents/acceptance.md)
- [Recorded acceptance decision](../../../../docs/adr/0003-recorded-acceptance-for-readiness.md)
- [Directory selection decision](../../../../docs/adr/0004-directory-specific-project-selection.md)

## Comments

Created from the user-approved seven-ticket breakdown. Acceptance requires retained verification and an authored Acceptance record; unstarted execution and triage do not establish implementation eligibility.
