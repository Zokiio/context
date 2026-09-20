# Orient a session within one local project

Status: ready-for-agent

## Problem Statement

A fresh coding-agent session cannot tell what the project has committed to deliver, which work is available, or why other work cannot start. The user must reconstruct that view from tickets, specifications, decisions, and completion notes before asking the existing reader for detailed task context.

The first reader makes the gap visible in this repository. Its five implementation tickets have completed acceptance checklists and recorded evidence, yet their triage role remains ready-for-agent. Triage alone cannot distinguish unfinished work from accepted completion. A missing relationship section can also look like an empty dependency list unless the record format distinguishes those cases.

The user needs a compact, traceable project view that helps an agent choose suitable work. The view must distinguish recorded progress, current commitments, computed readiness, and acceptance evidence. Missing information must remain visible.

## Solution

Add the read-only command `ctx orient`, backed by one reusable Go application operation. The caller selects one local project bundle explicitly. The operation reads current records, discovers work items, resolves the project's authored goal and commitment links, and evaluates readiness using declared relationships and recorded acceptance.

Return a concise text report by default and a versioned JSON result with `--json`. Show project direction, current commitments, ready work, work in progress, other backlog work, open decisions, and gaps. Explain why each ticket is eligible or excluded. The human or agent chooses the next task and uses `ctx context` for its detailed requirements and documents.

The default shortlist contains only unstarted, committed work whose triage role is ready-for-agent and whose readiness checks pass. A completed prerequisite also needs a usable acceptance decision with current requirement and evidence fingerprints. Orientation relies on that authored decision; it does not rerun tests or independently judge whether evidence proves the requirements.

## User Stories

1. As a developer, I want to select one project explicitly, so that orientation cannot silently use a different project.
2. As a developer, I want a project bundle outside a Git root to work, so that record location remains independent of code location.
3. As a developer, I want invocation from another directory to work, so that my shell location does not determine project scope.
4. As a coding agent, I want the project's stable identity and title, so that I can identify the work I am inspecting.
5. As a coding agent, I want authored project goals and their source references, so that I can understand the intended direction.
6. As a developer, I want goal selection controlled by explicit links, so that orientation reflects the documents I designated.
7. As a developer, I want current commitments to be explicit, so that readiness alone does not expand the intended delivery scope.
8. As a developer, I want unstarted work to count as a commitment, so that planned delivery is visible before execution begins.
9. As a coding agent, I want commitment membership separate from execution state, so that I can distinguish intended work from active work.
10. As a developer, I want commitments selected per ticket, so that linking a specification does not silently commit all associated work.
11. As a coding agent, I want associated specifications shown for navigation, so that I can locate the larger requirements behind a ticket.
12. As a developer, I want all work-item records in the selected bundle discovered, so that I do not maintain a second backlog index.
13. As a developer, I want new and uncommitted records included, so that orientation reflects my current working files.
14. As a coding agent, I want stable work-item identities retained through moves, so that a filename change does not create a different work item.
15. As a developer, I want conflicting record identities reported, so that the reader does not silently choose one duplicate.
16. As a coding agent, I want unstarted, in-progress, completed, and cancelled execution states distinguished, so that finished or active work is not offered as a new task.
17. As a coding agent, I want missing execution state reported as unknown, so that an unfinished record template cannot appear unstarted by default.
18. As a developer, I want document lifecycle, triage, and execution state kept separate, so that each field retains one meaning.
19. As a coding agent, I want the default shortlist limited to current commitments, so that my choice stays within the selected delivery scope.
20. As a coding agent, I want the shortlist to require ready-for-agent triage, so that work still needing preparation is not offered for implementation.
21. As a coding agent, I want every shortlisted item to have passing readiness checks, so that a promising label does not hide unmet conditions.
22. As a developer, I want eligible work listed with reasons, so that I can choose without the command imposing a recommendation.
23. As a developer, I want ready work outside commitments shown as backlog, so that available future work remains visible.
24. As a coding agent, I want work in progress shown separately, so that I can see current activity without treating it as unclaimed work.
25. As a developer, I want completed and cancelled records retained in the inventory, so that progress remains explainable without putting those records on the shortlist.
26. As a coding agent, I want readiness reported as ready, blocked, or unknown, so that a known obstacle is distinct from missing information.
27. As a coding agent, I want every failed or unknown check explained, so that I can identify what needs to change before work can start.
28. As a developer, I want explicit acceptance criteria required for readiness, so that a ticket has a stated contribution to verify.
29. As a coding agent, I want required context availability checked, so that a broken specification or context link cannot be mistaken for complete preparation.
30. As a developer, I want an explicitly empty dependency section to mean no dependencies, so that self-contained work can be ready.
31. As a developer, I want a missing required dependency section to mean unknown, so that absence is not mistaken for a deliberate declaration.
32. As a coding agent, I want direct and indirect prerequisites evaluated, so that an unmet prerequisite cannot be hidden behind another ticket.
33. As a developer, I want cancelled prerequisites to remain unsatisfied, so that cancellation does not silently authorize dependent work.
34. As a coding agent, I want dependency cycles identified, so that evaluation terminates and explains why affected work cannot be shortlisted.
35. As a developer, I want unrelated work evaluated independently of a known cycle or blocker, so that one problem does not automatically block the project.
36. As a developer, I want open decisions listed from authoritative records, so that unresolved questions remain visible to a fresh session.
37. As a developer, I want a decision to block only explicitly linked tickets, so that unrelated commitments can continue.
38. As a coding agent, I want a resolved decision to include an authored answer, so that a state change alone cannot hide an unanswered question.
39. As a developer, I want completed prerequisites to cite an acceptance decision, so that an execution flag alone does not establish acceptance.
40. As a coding agent, I want acceptance to identify the criteria and tested revision, so that I can understand what was evaluated.
41. As a developer, I want acceptance linked to supporting evidence, so that its basis can be inspected.
42. As a developer, I want authorized workflow skills able to record acceptance, so that routine verified completion does not require a separate manual transcription step.
43. As a developer, I want authorship and human approval separately attributed, so that an agent's conclusion is not presented as a teammate's approval.
44. As a coding agent, I want missing or unreadable acceptance evidence reported, so that unavailable proof cannot silently satisfy a prerequisite.
45. As a developer, I want changed requirements to invalidate their previous acceptance, so that an old result cannot certify a new requirement.
46. As a developer, I want changes to supporting evidence detected, so that an acceptance decision remains tied to the evidence it used.
47. As a developer, I want ticket scope and introductory requirements covered by freshness checks, so that changes outside the acceptance checklist are not missed.
48. As a developer, I want routine ticket comments excluded from the requirement fingerprint, so that adding a completion note does not invalidate otherwise unchanged acceptance.
49. As a developer, I want previous acceptance decisions preserved, so that reassessment adds history instead of rewriting the original result.
50. As a developer, I want a tested revision preserved as historical provenance, so that unrelated progress on the main branch does not invalidate every completed prerequisite.
51. As a developer, I want linked external documents constrained to allowed source directories, so that record scope remains explicit.
52. As a developer, I want repeated source references deduplicated while retaining their reasons, so that shared context does not hide how it was selected.
53. As a coding agent, I want source paths and content digests, so that I can identify the exact records used for the report.
54. As a developer, I want a readable default report, so that I can inspect project state in a terminal.
55. As an automated caller, I want versioned JSON containing the same facts, so that tools do not parse terminal prose.
56. As a coding agent, I want orientation to remain compact, so that project selection does not require reading every task's full source text.
57. As a coding agent, I want a clear transition to task context for a chosen ticket, so that I can obtain detailed requirements after selecting work.
58. As a developer, I want partial reports to retain useful information, so that one unavailable source does not erase everything the reader established.
59. As a coding agent, I want incomplete discovery distinguished from known blocked work, so that I do not mistake a partial inventory for a complete backlog.
60. As a developer, I want configurable file and source-byte limits, so that large projects cannot force unbounded collection.
61. As a coding agent, I want omissions and affected evaluations identified, so that a limit cannot produce a misleading shortlist.
62. As an automated caller, I want completed evaluation to succeed even when no ticket is ready, so that normal project state is distinct from a failed read.
63. As an automated caller, I want separate partial-result and operation-failure exit statuses, so that I can respond appropriately.
64. As a developer, I want stable ordering, so that repeated runs over unchanged records are easy to compare.
65. As a developer, I want orientation to leave records and Git state unchanged, so that inspection does not mutate the project.
66. As a developer, I want the Go operation usable independently of the CLI, so that future interfaces share the same rules.
67. As a developer, I want existing task-context behavior preserved, so that adopting orientation does not break known-ticket workflows.
68. As a developer, I want a second project fixture, so that the implementation does not depend on this repository's layout or identities.
69. As a developer, I want a real ticket-pickup trial, so that acceptance demonstrates the intended workflow beyond fixtures.
70. As a developer, I want the five completed reader tickets excluded from the shortlist after migration, so that the feature fixes the concrete problem that motivated it.

## Implementation Decisions

### Application interface and module responsibilities

- Add one orientation application operation that accepts explicit project scope, allowed document roots, file-count and byte limits, and a cancellation context. It returns one structured result and a separate operation error. It does not print, exit, write records, or invoke a model.
- The orientation module owns discovery, record-profile checks, commitment membership, dependency and decision evaluation, acceptance freshness, ordering, limits, and diagnostics. The CLI module owns argument validation, rendering, and exit-status mapping.
- Reuse the existing application-operation testing pattern. The CLI receives the task-context and orientation operations as explicit dependencies. Keep command-framework types in the CLI module.
- Extract shared document parsing, relationship parsing, and authorized source reading where both operations need them. Keep their implementation behind the application interfaces. Preserve the task-context operation and its existing result contract.
- Continue using the installed Markdown, YAML, CLI, and Go filesystem libraries. A new persistence abstraction, public Go library contract, service process, and plugin interface are unnecessary for this milestone.
- Skills continue to author records. Orientation performs read-only checks of those records and cannot authenticate the person or workflow named in their metadata.

### Project scope, discovery, and source snapshots

- Require one explicit project bundle. Resolve its directory and allowed source directories as the current reader does. A Git repository and an installed copy of the product are not prerequisites.
- Read and validate the reserved project manifest first. Require the existing Project type, stable nonempty ID, and title. A valid directory with a missing or invalid manifest produces a partial result. An unusable project directory is an operation failure.
- Inventory Markdown files in the bundle in deterministic relative-path order. Include new and uncommitted files. Do not use Git tracking or ignore rules to decide whether an authored record exists.
- Identify WorkItem, Decision, and Acceptance records through parsed frontmatter. Preserve unknown metadata. Other document types and plain Markdown may serve as linked sources but do not become work items. Full OKF conformance validation remains outside this operation.
- An unreadable candidate or malformed frontmatter can hide records and makes inventory discovery incomplete. A recognized record with missing required profile fields remains visible with identifying diagnostics and unknown affected evaluations.
- Detect duplicate nonempty IDs across discovered typed records. Never choose an authoritative record by traversal order. Mark ambiguous records and their dependents unknown; unrelated unambiguous records remain inspectable.
- Do not descend through directory symlink aliases during inventory. Resolve file aliases and explicit links through the existing containment rules, deduplicate by resolved path, and reject record targets outside the bundle. Additional allowed roots authorize linked documents and evidence, not external work items or decision records.
- Read each resolved source once per operation and use those bytes for parsing and digests. Report the observed source revisions without claiming an atomic filesystem-wide snapshot.
- Preserve the existing explicit source rules: local file links only, whole files for fragment links, authored links retained in provenance, and no recursive expansion through ordinary document prose.

### Project manifest and work-item profile

- The manifest has required level-two Goals, Current commitments, and Open decisions sections. Recognize repeated sections in document order, with the existing section-boundary and Markdown-link rules.
- Goals contains authored prose and meaningful link labels for the designated goal documents. Return that authored material and source references. Do not generate summaries of linked documents.
- Current commitments links directly to WorkItem records. These links are the authoritative membership list. Specifications provide navigation and requirements; they do not implicitly confer commitment on associated tickets.
- Open decisions links to Decision records. Also discover decisions explicitly linked as blockers of work items. The decision record owns its state; an outdated index entry does not override a resolved answer.
- An explicitly empty required section declares none. A missing required section leaves the corresponding information unknown. In link-only sections, a nonempty prose-only declaration must be the literal `None` or `None.`; other prose without usable links is diagnosed rather than interpreted as an empty list. Goals may also contain authored prose.
- WorkItem records retain required type, ID, title, and triage metadata. Add the authored execution field with exactly unstarted, in-progress, completed, and cancelled values. Missing or invalid execution is reported as unknown rather than defaulted.
- Preserve the existing document-lifecycle status field and triage vocabulary. Neither substitutes for execution. This milestone adds no document-lifecycle condition to the four agreed shortlist conditions.
- Require an Acceptance criteria section containing at least one nonempty authored criterion. Check presence and structure, not the substantive quality of its prose. A missing section is unknown; an explicitly empty section is a known readiness failure. Triage and recorded acceptance remain responsible for judging adequacy.
- Require explicit Blocked by and Blocked by decisions sections. Continue to use Spec and Context for selected requirement and context documents. Spec remains optional for self-contained work.
- Use a level-two Acceptance section for the single current acceptance-record link. It is optional for unstarted and in-progress work. Completed work cannot satisfy a dependency without one usable current acceptance record. Historical acceptance links belong in comments or other non-current history.
- Preserve one authoritative home for each relationship. Linked documents retain their own requirements and decisions; reports contain references and attributed observations.

### Decision and acceptance records

- A Decision record has type Decision, stable ID, title, and decisionState set to open or resolved. A resolved decision requires a nonempty Resolution section. Missing state or a missing required answer makes its effect unknown.
- Only an explicit Blocked by decisions link makes a decision block a ticket. An open project-level decision does not automatically block every commitment. A resolved decision releases that particular condition, subject to the ticket's other checks.
- An Acceptance record has type Acceptance, stable ID, title, projectId, workItemId, actor, decidedAt, testedRevision, fingerprintVersion, ticketSHA256, and criteriaSHA256. The subject IDs must match the selected project and linked work item.
- Actor records a human or workflow kind and a nonempty identity. The decision timestamp uses an explicit UTC offset. TestedRevision contains nonempty origin and revision values, allowing a commit, artifact digest, or other recorded revision without requiring a Git checkout.
- Optional human approvals retain their own actor, timestamp, and revision. A workflow actor or an automated review must never be rendered as human approval. Authorization is the surrounding workflow's responsibility; metadata is an attributed assertion, not an authentication mechanism.
- An Acceptance record asserts coverage of the subject's entire Acceptance criteria section, identified by criteriaSHA256, together with the accepted ticket requirement fingerprint. Partial acceptance does not satisfy a prerequisite in this milestone.
- Required Requirements and Evidence sections hold source snapshots. Each entry is a Markdown list item containing one file link and one inline-code SHA-256 digest. Diagnose missing, malformed, conflicting, or ambiguous entries. Evidence must contain at least one source.
- The Requirements snapshots must cover the subject's directly selected Spec and Context documents and its linked decision records. Snapshot membership and digests must match the current selected sources. Additional requirement snapshots are explicit accepted requirements and must also remain available and unchanged.
- Evidence identifies readable local UTF-8 source files under the allowed roots. External observations may be recorded in those files with their original provenance. Orientation does not retrieve remote evidence or interpret binary attachments.
- Reject acceptance records that snapshot themselves or use themselves as evidence. A reference to the subject ticket uses the ticket's requirement fingerprint rather than a whole-file snapshot that would include its comments and current acceptance link.
- Keep previous acceptance records when reassessing. The ticket's single current Acceptance link chooses the applicable record. Never choose one implicitly by timestamp or filename.

### Acceptance fingerprints and freshness

- Use SHA-256 with fingerprintVersion 1 and lowercase hexadecimal digests. Whole linked requirement and evidence files use their exact source bytes. Do not replace their current digests with the revision of a Git checkout.
- Confirmed decision A1 fingerprints the whole Markdown body after frontmatter removal, excluding exact level-two Comments and Acceptance sections and their nested content. Retain all other body content, including introductory requirements, Scope, Acceptance criteria, and dependency relationships.
- Apply exclusions using parsed Markdown structure, not regular expressions that can mistake fenced text for headings. Repeated excluded sections are all excluded. Preserve the retained source bytes in document order, including their line endings.
- Reference definitions used by retained content remain fingerprint-relevant even when authored in an excluded section. Include the effective raw definitions used by the Markdown parser, including their line endings, once each in first-use order when they are not already retained. An unrelated comment edit must not change the fingerprint; a comment edit that changes a retained reference's meaning must change it.
- Form the ticket fingerprint's byte-string sequence from one string containing all retained body bytes, followed by the additional reference-definition strings. Form the criteria fingerprint's sequence from the complete raw source span of each Acceptance criteria section in document order, followed by its additional used reference definitions. A section span starts at its heading and ends immediately before the next level-one or level-two heading, or at the end of the body.
- Encode a fingerprint as its UTF-8 domain tag, one zero byte, the number of byte strings as an unsigned 64-bit big-endian integer, then each string's byte length in the same integer encoding followed by its bytes. Use domain tag ctx.ticket-body.v1 for ticketSHA256 and ctx.acceptance-criteria.v1 for criteriaSHA256. SHA-256 hashes that encoded byte stream. Do not normalize whitespace, line endings, or Unicode. Add independently checked conformance fixtures for both encodings.
- Changing ticket requirements, selected requirement-source membership, requirement bytes, or evidence bytes makes the current acceptance stale and requires a new decision. Unchanged data with an unavailable source cannot be judged fresh and remains unknown.
- Excluding frontmatter keeps execution updates outside the body fingerprint. Validate subject identity and the other consumed metadata separately. A title or unrelated comment edit does not by itself change accepted requirements.
- The original tested revision remains visible as historical provenance. It need not equal current Git HEAD. Freshness checks establish the relation between current authored requirements, evidence, and recorded acceptance; they do not certify the current application checkout or rerun verification.

### Readiness, prerequisite satisfaction, and shortlist membership

- Evaluate four readiness conditions for each work item: explicit acceptance criteria; available selected requirement and context sources; satisfied work dependencies; and resolved explicitly linked blocking decisions. Report each condition as pass, fail, or unknown with its reasons.
- A known failed condition makes aggregate readiness blocked. Otherwise, an unknown condition makes it unknown. Only all passing conditions produce ready. Preserve every unknown condition even when a known blocker determines the aggregate state.
- A prerequisite satisfies its edge only when it is completed, its current acceptance is valid and fresh, its explicitly linked blocking decisions are resolved, and its own prerequisites satisfy their edges. An unfinished or cancelled prerequisite is a known blocker. Missing records, ambiguous identity, or unavailable required acceptance information are unknown.
- A stale acceptance record is a known failure requiring reassessment. Missing evidence or malformed acceptance metadata is unknown. These states must not collapse into accepted completion merely because the execution field says completed.
- Detect dependency cycles, including self-dependencies, without unbounded recursion. A cycle makes its members and work depending on those cyclic edges blocked. Report the involved identities and edges. Continue evaluating unrelated work.
- A fully observed cycle is a known project condition and does not by itself make the operation incomplete. Preserve the existing task-context distinction: it may still return complete source context for the same cycle.
- A ticket is eligible for the default shortlist only when execution is unstarted, commitment membership is true, triage is ready-for-agent, and readiness is ready. Eligibility is an output fact, separate from readiness. Missing execution or commitment information can exclude a ticket even when its readiness checks otherwise pass.
- Show work in progress separately. Show uncommitted unfinished work as backlog, including its readiness. Completed and cancelled records remain in the inventory and progress information but never enter the shortlist.
- If the commitment list is incomplete, retain proven positive memberships and mark membership unknown where it cannot be established. If the project identity is invalid or inventory discovery is incomplete, suppress the default shortlist because project scope or undiscovered records may affect identity or graph interpretation. Retain observed records and individual check results, with an explicit explanation.
- Unavailable project information that does not affect a ticket's scope, identity, commitment, or checks does not automatically change that ticket's readiness. The overall result can be partial while some individually established facts remain usable.
- Preserve authored commitment order for the shortlist. Order remaining records by stable ID and resolved path. This is deterministic presentation order, not a priority score or recommendation.

### Output, limits, and failure behavior

- The command accepts required project scope, repeated allowed-source directories, optional positive max-files and max-bytes limits, and the JSON-format flag. It does not require a ticket. Reject unexpected positional arguments and invalid or ambiguous flags without prompting.
- Reuse initial defaults of 100 distinct source files and 1,048,576 source bytes. Count the manifest, inventory reads, and linked sources within one operation-wide budget, deduplicated by resolved path. Limits measure inspected source bytes rather than rendered output bytes.
- Stop admitting new sources at the first limit breach. Keep complete files already read, identify the first omitted source and known pending work, and mark discovery or evaluation incomplete as appropriate. Do not claim an exhaustive inventory or omission count after incomplete discovery.
- Read the manifest first, then inventory candidates in relative-path order, then remaining selected sources in deterministic record and authored-link order. Do not reread already captured files when evaluating another work item.
- JSON schema version 1 includes schemaVersion, complete, inventoryComplete, project, goals, currentCommitments, workItems, shortlist, inProgress, backlog, decisions, sources, and diagnostics. Lists are always arrays. Unavailable scalar facts and an unavailable project identity are represented explicitly rather than invented.
- Work-item entries include stable identity when known, title, source reference, triage, execution, commitment membership, specification references, readiness checks, aggregate readiness, eligibility and exclusion reasons, and applicable acceptance metadata. Use separate fields for those concepts. Also expose the current fingerprintVersion, ticketSHA256, and criteriaSHA256 when they can be computed, even when no acceptance exists, so authoring skills can obtain the values from this operation.
- Source entries contain resolved paths, whole-file SHA-256 digests, and distinct inclusion reasons with referring sources and authored links. Requirement fingerprints are separately named so they cannot be confused with the whole-file source digest. Do not include every full source body in orientation JSON.
- The text report uses the same result as JSON. Present project identity and authored goals, commitment progress, the shortlist, work in progress, other backlog work, open decisions, and gaps. Keep details attributable and concise; do not generate a narrative project summary.
- Include the selected ticket reference and information needed to request task context. The existing task-context selection rules remain intact. Decision answers that impose implementation requirements must also be authored as Spec or Context links so the detailed pickup workflow receives them.
- Complete means all information required by the requested orientation evaluation was available and evaluable within the limits. It does not mean all work is ready, every record is fully OKF-valid, or every relevant requirement was discovered.
- Exit 0 follows a complete evaluation, including an empty shortlist, known blockers, or known stale acceptance. Exit 1 follows a partial report caused by missing information, unreadable or malformed required records, ambiguity, or limits. Exit 2 follows invalid invocation, an unusable configured root, cancellation, an application operation failure, or output failure.
- Put structured data and ordinary text reports on stdout. Put invocation and operation errors on stderr. Report source and evaluation diagnostics inside the returned report. A failure while writing output may leave partial bytes on stdout and must return exit 2.
- Diagnostics have stable codes, severity, affected record or source, referring source and authored link where applicable, and a human-readable explanation. Distinguish missing source, outside-scope source, invalid profile, duplicate identity, missing required section, stale requirements, changed evidence, open decision, cancelled dependency, dependency cycle, and collection limit cases.

### Adoption and record migration

- Establish the authoring conventions and completion procedure in the first implementation slice, before recording acceptance against those conventions. Update relevant workflow skills to describe the new profiles and fingerprints. Their writes remain explicit file edits during this milestone. Document how to derive fingerprints reproducibly from the specified encoding.
- Retain actual criteria results, evidence, and tested revisions after every slice. Begin structured acceptance authoring when fingerprint output is available, then validate those records when acceptance checking is implemented. Preserve earlier decisions and reassess any requirements that subsequently change.
- Intermediate releases may expose the command while some checks remain unimplemented. Report each applicable unsupported check as unknown, mark the report partial, and exclude affected work from the shortlist. The completed milestone implements every check specified here.
- Migrate the five completed reader tickets using their existing evidence and the recorded merged revision. Preserve the original evidence's tested revisions and authorship. A backfilled acceptance decision records its actual author and decision time; it must not fabricate a historical human approval or imply new tests were run.
- Add explicit execution states and empty relationship declarations where the records support them. Populate project goals, commitments, and decision links from their authoritative sources. Leave uncertain facts unknown until evidence establishes them.
- The predecessor of the final trial ticket owns the final application checks, retained reader build, and project-record preparation. Its closeout must establish the prerequisites for pickup without performing the trial ticket's work.
- Exercise orientation before starting the real acceptance-trial ticket, while that ticket is still unstarted and committed. After selection, use the existing task-context workflow and record the observed result. This avoids creating a synthetic outstanding task solely to make the demonstration pass.

## Testing Decisions

- Put most behavioral tests at the orientation application's public Go interface, using real temporary project directories. This follows the existing task-context tests that construct files, call the application operation, and compare externally visible results. Avoid tests that depend on private parser functions, internal call order, or filesystem mocks.
- Keep a small CLI testscript suite for flags, defaults, text and JSON rendering, stdout and stderr, and exits 0, 1, and 2. Extend the existing injected-operation tests for application and writer failures. Prefer assertions on useful report facts over full prose snapshots.
- Test a complete project containing committed ready work, uncommitted ready work, active work, completed accepted work, and cancelled work. Assert all four shortlist conditions and separate execution, readiness, and membership values.
- Test explicit empty declarations, missing sections, empty acceptance criteria, invalid field values, malformed frontmatter, duplicate IDs, and wrong-type relationship targets. Test a known blocker together with unknown information and retain both explanations.
- Test direct and transitive dependencies, shared prerequisites, cancelled dependencies, self-cycles, longer cycles, and unrelated work beside a cycle. Assert deterministic termination and the distinction between known blocked work and incomplete evaluation.
- Test open and resolved decisions, a resolved state without an answer, stale open-decision index entries, and an open decision that has no blocking link to otherwise eligible work.
- Test valid acceptance, mismatched subjects, missing current acceptance, multiple current links, missing evidence, changed evidence, missing requirement snapshots, changed source membership, and workflow authorship with separately attributed human approval.
- Test the confirmed fingerprint coverage explicitly: changes to introductory requirements, Scope, criteria, and dependency links invalidate acceptance; routine Comments and Acceptance updates do not. Test fenced headings, repeated sections, CRLF, and reference definitions whose location is excluded but whose use is retained.
- Test that whole-source digests change after a comment edit while the requirement fingerprint and acceptance remain valid. Test that a changed tested checkout's HEAD alone does not replace or invalidate the recorded historical revision.
- Test external allowed roots, denied external records, missing sources, symlink aliases, escapes, unreadable files, invalid encoding, and exact source-byte accounting. Use the prior reader's real-filesystem tests as the pattern.
- Test default and explicit limits, exact-fit limits, first overflow, partial inventory, known pending omissions, source deduplication, and conservative shortlist suppression after incomplete discovery.
- Test a second project outside the application repository and invoke it from an unrelated directory. Verify that source files and Git state are unchanged by both rendering modes.
- Run the existing reader tests after shared-code extraction. Its source selection, result schema, cycle behavior, allowed roots, and exit statuses must remain compatible.
- Complete the normal Go tests, race tests, vet, CLI build, and the declared minimum-Go checks before the real workflow trial. Record the actual results and revisions rather than describing planned checks as passed.
- The real trial uses a fresh agent session to identify a real committed ticket, explain each eligibility condition, and obtain detailed context. Record the invocation, source digests, reader revision, selected ticket, observed reasons, and follow-up context result. Verify that all five completed reader tickets stay off the shortlist.

## Out of Scope

- Jira or other remote retrieval, synchronization, authentication, and provider drift checks.
- Workspace aggregation, automatic project or repository discovery, and cross-project dependency evaluation.
- A new epic hierarchy, automatic commitment inheritance, parent-chain requirement traversal, and implicit sibling selection.
- Agent launch, assignment, implementation claims, takeover, scheduling, and automatic task choice.
- CLI writes, automatic state transitions, acceptance authoring commands, record repair, and Git mutation.
- LLM-generated summaries, semantic grading of requirements or evidence, automatic test reruns during orientation, and certification of the current application checkout.
- A general policy language, configurable workflow engine, cryptographic approval verification, and mandatory human approval for every acceptance decision.
- Persistent caches, standards distribution, installed-skill or runtime health checks, a TUI, and a web interface.
- Full OKF conformance validation, remote or binary evidence ingestion, and filesystem-wide snapshot isolation.

## Further Notes

The [discovery record](discovery.md) preserves the interview and the user's image references. The [domain glossary](../../CONTEXT.md), [product vision](../../docs/vision.md#after-the-first-milestone), [bundle decision](../../docs/adr/0001-one-okf-bundle-per-project.md), [relationship decision](../../docs/adr/0002-work-relationships-in-markdown-sections.md), and [acceptance decision](../../docs/adr/0003-recorded-acceptance-for-readiness.md) provide the existing vocabulary and constraints. The [local tracker](../../docs/agents/issue-tracker.md) defines publication, and its [triage mapping](../../docs/agents/triage-labels.md) supplies the status above.

Decision A1 began as an explicit specification assumption because the user requested synthesis before separately answering the proposed expansion of fingerprint coverage. During the ticket-breakdown review, the user confirmed whole-body coverage with the designated exclusions. Scope, introductory requirements, criteria, and dependency links are covered; ordinary Comments and current Acceptance-link edits remain excluded.

The user also approved conservative intermediate releases, early authoring conventions and completion guidance, separate overview and shortlisting tickets, and separate acceptance-validation and recursive-dependency tickets. The [implementation handoff](discovery.md#implementation-handoff) links the eight approved tickets. This specification remains ready-for-agent; publication of tickets does not establish implementation or milestone acceptance.

Record field names, source-snapshot encoding, conservative behavior after incomplete discovery, and exact failure distinctions are technical synthesis choices that make the agreed behavior implementable. The user confirmed the testing seam during drafting: Go application-operation tests with real project files, plus a small CLI suite.

The source-reader milestone is merged at revision ecb8f294144dab8bb36f2e74e6ad174bcce17ef1. Its [recorded acceptance evidence](../context-reader/acceptance-trial.md) is the basis for the completed-ticket migration, not evidence that this orientation feature has already been implemented or tested.

## Tracker snapshot execution

A WorkItem with a string `sourceURL` that parses as an absolute HTTP or HTTPS URL with a nonempty hostname is an explicitly marked tracker snapshot. Only an absent execution key is tolerated without an invalid-profile diagnostic. Empty, null, or invalid values remain invalid. Native work items still require execution. Unknown execution stays ineligible and cannot satisfy a prerequisite. The URL is an authored authority marker; the reader does not fetch it or infer freshness, readiness, or remote completion.
