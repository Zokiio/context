# Opsbase manual trial

Status: local trial complete. Opsbase PR 248 has passing hosted CI and awaits human review. No init implementation has started.

The user selected [Opsbase](https://github.com/Zokiio/opsbase) and [issue 165](https://github.com/Zokiio/opsbase/issues/165), deterministic onboarding browser coverage. The trial uses Codex on macOS and the project's existing `.agents/skills` directory.

## Checkout and authority

The original checkout is on another feature branch and has a modified README. Trial work uses `/Users/zoki/code/opsbase-ctx-trial`, branch `feat/ctx-trial-165`, starting from origin/main at `74992af`. The original checkout remains untouched.

GitHub Issues remains authoritative. Local reader inputs live under ignored `.context/trial/`, including the original GitHub retrieval and a labeled Markdown snapshot. The snapshot has a source URL, source update time, retrieval time, and stable local UUID. Its execution field describes observed local work, not a remotely synchronized status.

## Friction observed during bootstrap

| Observation | Manual action | Implication to evaluate |
| --- | --- | --- |
| No ctx binary was on PATH | Built ctx in its source checkout and supplied the binary under `.context/trial/bin/ctx` | Target-project skills must locate a supplied binary instead of building ctx from the target repository |
| GitHub Issues is authoritative, while ctx reads local Markdown | Retained a raw retrieval and made a labeled local input for issue 165 | Tracker preservation requires a source/freshness boundary before an installer can claim usable task context |
| The GitHub issue says `None.` under Blocked by | First orientation reported an invalid relationship section; normalized only the local input to `None` | A semantically clear issue can require format adaptation even when context assembly succeeds |
| The GitHub issue has no ctx execution field | First orientation reported an invalid profile; recorded observed local work as in-progress | An open issue or triage label cannot supply execution state automatically |
| Opsbase uses a context map and multiple domain glossaries | Linked the root map, relevant glossary, and onboarding/auth ADRs explicitly | Copying ctx's single-context assumptions would omit project structure |
| Existing AGENTS.md already routes tracker, triage, and domain guidance | Added three scoped trial pointers and preserved existing instructions | Existing guidance needed a small addition, not replacement |
| Copied skills assume `.scratch/records` and a Go build in the current repository | Replaced those instructions with the supplied binary, selected local records, and snapshot freshness checks | These edits are concrete portability requirements |
| Recovery instructions link a profile and assume locally authored Decision records | Copied PROFILE.md and routed authoritative questions back to GitHub conventions | Installing only SKILL.md leaves required references or wrong ownership assumptions |
| Acceptance instructions contain ctx's historical bootstrap-ticket exceptions and fingerprint reference | Removed obsolete exceptions and scoped local acceptance to the retained snapshot | Tool-development history does not belong in another project's completion procedure |
| Supplied binary, snapshots, evidence, and recovery cache are local artifacts | Added ignore rules for `.context/trial`, `.context-cache`, and configuration locks | Bootstrap must distinguish durable guidance from disposable local inputs |

## Checks so far

After the two snapshot adaptations, `ctx context`, `ctx orient`, and `ctx resume` each exited 0 and reported complete results without diagnostics. The agent read all six selected context sources. Resume reported recovery absent, as expected before a checkpoint.

These checks establish a working local read path for the captured issue. They do not establish live GitHub integration or acceptance of issue 165. Full reports remain in the ignored trial directory.

## Task use and verification

The agent used the assembled context to implement issue 165 in the trial worktree. The change adds a serve-only WorkOS identity adapter, three Playwright scenarios, and a focused frontend CI step. Opsbase's session probe, capability provider, router, and wizard remain on their production code path.

- All three focused browser scenarios passed, also with `CI=1`, without WorkOS credentials.
- The completion scenario reaches the company application through the product's normal completion navigation and creates an invoice draft with the new 45-day payment default. It allows no additional test-authored refresh.
- The resume scenario closes the page and clears browser storage before reading the persisted-session API from a fresh page.
- Replacing the full-bootstrap action with a history-only navigation made the completion test fail. The original production file was restored and the tests passed again.
- Admin-web typecheck, a separate typecheck of the new fixtures, scoped ESLint, and whitespace checks passed. A production build excluded the test identity adapter.
- The full browser suite had 25 passes and four failures. All four reproduced with the original Playwright configuration. They concern existing customer headings, settings readiness text, and a companion photo-empty-state expectation.
- A recovery checkpoint was published from the exact context used. `ctx resume` returned it as available in the same session, with a complete report and no diagnostics. A fresh-session continuation has not yet been measured.
- The adapted acceptance procedure retained local evidence and source identity. No Acceptance record was created, and the GitHub issue was not closed or updated. Hosted CI and human code review remain unobserved.

Full reports and source digests remain in ignored `.context/trial/`; recovery observations remain in ignored `.context-cache/`. The original checkout and its README edit remain unchanged.

## Comparison with the debate

The trial needed one agent, ordinary file copies into an existing skills directory, explicit document links, a supplied binary, and a binding. It did not need hooks, multiple harnesses, shared-directory links, copy synchronization, or a new human questionnaire.

The largest gap was preserving GitHub authority while supplying local reader inputs. The binary and record-path assumptions in the copied skills were immediate problems. Required supporting references and historical acceptance instructions also needed attention. These are observed requirements; the broader installation machinery remains untested.

The [init draft](../records/ctx-init/issues/01-agent-guided-project-initialization.md) has been narrowed accordingly. Its scope decision remains open for review of these findings. The local trial is complete without resolving unrelated cases or claiming hosted CI passed.

## Review follow-up on 2026-09-20

The trial records are in [ctx PR 12](https://github.com/Zokiio/context/pull/12), stacked on PR 11. The application change is in [Opsbase PR 248](https://github.com/Zokiio/opsbase/pull/248). Trial bootstrap guidance and disposable inputs remain local to its worktree.

Automated standards review found contradictory browser-test prerequisites. Automated spec review found that the company-name assertion only inspected its own mocked response. Both were corrected. The completion test now opens Settings through its sidebar, checks the actual company-name field, returns through client-side navigation, and creates an invoice with the new defaults. It still permits only the initial document load and the product's normal completion navigation. All three focused scenarios passed with `CI=1` after these corrections, with fixture typecheck and scoped ESLint also passing. Human review remains outstanding.

### Adaptation size and elapsed time

Compared with ctx source revision `826c03f`, the copied documents required these line edits. Counts are removed plus added lines, including blank lines, rather than a percentage of unique lines rewritten.

| Document | Original lines | Removed | Added | Total edits |
| --- | ---: | ---: | ---: | ---: |
| task-context SKILL.md | 28 | 6 | 5 | 11 |
| recovery-notes SKILL.md | 87 | 4 | 4 | 8 |
| Acceptance procedure | 66 | 27 | 3 | 30 |

No phase timer ran during the original trial. Filesystem timestamps place worktree creation at 12:59:57 UTC and the completed binding at 13:02:05, approximately two minutes for bootstrap including adaptation. The copied guidance was written within that interval, so its effort cannot be separated reliably. The final checkpoint is timestamped 13:19:28, approximately seventeen further minutes for task work, verification, and reporting. These are reconstructed wall-clock intervals, not measured human effort, and exclude earlier research and the later review. Future trials should record separate phase start and end times.

### Fresh-session recovery measurement

A new CLI session started in the existing trial worktree with only the absolute ticket path as its prompt. It received no conversation history or continuation hints. The original checkpoint and snapshot were preserved. The checkout did include the intervening review commits, so the session had to distinguish retained observations from current code.

The first attempt with installed CLI 0.154.0 failed after 4.87 seconds because the configured model required a newer client, before the ticket was read. A temporary CLI 0.155.1 installation supplied the retry without changing the global installation. Timings below start at that second launch and exclude the failed attempt and installation.

- 31 seconds: read the selected context and checked the live GitHub issue against the snapshot.
- 62 seconds: captured exact context and ran `ctx resume`.
- 102 seconds: explicitly read the selected checkpoint body and retained evidence.
- 109.49 seconds: correctly stated that the implementation was already committed, recognized the later company-name assertion, noted that the checkpoint predated the commits, and chose current-code review and focused checks as its next action.

This demonstrates recovery to a correct continuation plan in one fresh session. It does not measure time saved against a session without the checkpoint. The full event stream and monotonic timestamps are retained locally under `.context/trial/resume-experiment-current-cli/`; the failed attempt remains under `.context/trial/resume-experiment/`.

Hosted CI for Opsbase PR 248 subsequently passed frontend, backend, and smoke-test jobs on revision `812c919`. Those automated results do not replace human review.
