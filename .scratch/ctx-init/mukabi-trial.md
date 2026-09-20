# Mukabi manual trial

Status: Claude skill discovery and the paired recovery trial are complete. The dashboard skip-navigation change has passing local behavior tests; its Vercel preview failed and human review is outstanding. `ctx init` remains draft and unstarted.

## Project and task

The trial used Mukabi with Claude Code 2.1.236 on macOS. Its existing local roadmap and phase backlog remained authoritative. The scoped WorkItem refined only the skip-navigation part of P9-08; it did not replace the roadmap or complete the broader accessibility task.

The isolated checkout is `/Users/zoki/code/mukabi-ctx-trial`, branch `feat/ctx-trial-skip-navigation`, based on Mukabi `e2b2e59`. Bootstrap commit `84251d0` was the frozen start for both measured sessions. The supplied ctx binary reports revision `48f2203dd6fc201e6a0a3d7eaf2df787b8bb879f`, modified false.

## Friction observed during bootstrap

| Observation | Manual action | Implication to evaluate |
| --- | --- | --- |
| Mukabi already has a shared skill linked from Claude and other separately copied skills | Preserved those entries and added individual symlinks from `.claude/skills` to `.agents/skills` | Claude discovered and loaded both task-context and recovery-notes through the Skill tool. No copy fallback was needed on this machine |
| The local roadmap already tracks work despite having no open GitHub issues | Added a scoped native WorkItem selecting part of P9-08 | Tracker discovery must include existing local plans; absence of remote issues is not absence of a tracker |
| Source skills assume they can build ctx in the target checkout | Used a supplied binary and retained `ctx version` output | This adaptation repeats from Opsbase |
| Recovery links a separate profile and tracker/acceptance guidance | Copied PROFILE.md and adapted the linked guidance | Supporting files and authority assumptions repeat from Opsbase |
| The source task-context skill still asks for an uncommitted-reader digest manifest | Replaced it locally with version output and visible uncertainty for modified binaries | Simplified provenance has not yet propagated to all source guidance |
| CLAUDE.md has older npm examples while AGENTS.md requires Vite+ | Added a scoped pointer to the newer `vp` workflow | Existing instruction conflicts must be resolved explicitly during adaptation |
| Installed Claude was signed out | Paused harness work until the operator authenticated | Installation alone does not establish a usable agent tool |
| A relative binary invocation failed after changing directories | Claude reran it with an absolute path | Supplied-binary guidance must account for command working directories |
| The retained test plan proposed click-only tests for a keyboard requirement | Added user-event coverage for Tab, Enter, main focus, and continued tabbing after the measurement | Recovery notes retain proposals, not proof that the proposed verification is sufficient |

No hooks were added. Existing user-configured Claude startup hooks remained active. Binary files, raw reports, screenshots, and recovery observations remain ignored; project guidance and the scoped records are committed locally.

## Paired recovery measurement

Run order was no checkpoint first, then checkpoint. Both sessions received only `.scratch/records/ctx-trial/issues/01-skip-navigation.md` as their user prompt. A common runner instruction asked each to reconstruct state, state a continuation plan, perform one implementation/test step, publish a checkpoint, and stop. Both resolved to `claude-fable-5`, with identical local tool permissions, no session persistence, automatic memory disabled, and an empty MCP configuration.

Before the second run, the first process stopped, its output was archived outside the checkout, and the frozen files and Vite caches were restored at the same absolute path. File hashes and symlink targets matched the frozen manifest. Reports containing the original checkpoint were removed from both input sets. The second run received only the original recovery cache, not the first run's replacement note. A temporary first-run resume report was moved outside the session inputs shortly after the second process launched; the transcript confirms it was never read.

Correctness criteria were recorded before either run. Context collection and exploratory reads did not count as task action. The first action below is the successful tool completion for a relevant source edit, assessed against the task; it is not completed feature verification.

| Observation | No checkpoint | Checkpoint |
| --- | ---: | ---: |
| Harness initialization | 0.68 s | 0.71 s |
| First relevant source edit | 91.32 s | 121.71 s |
| Explicit qualifying plan before editing | Not observed | Not observed |
| Entire bounded session, including checks and checkpoint publication | 175.16 s | 272.49 s |

The observed action-time difference, no checkpoint minus checkpoint, was **-30.39 seconds**. The checkpoint condition was slower in this pair. Both sessions edited before stating the requested continuation plan, so the intended plan-time comparison is unavailable. Their final summaries cannot be counted retroactively as plans emitted before action.

The checkpoint session read the original note, checked all seven source comparisons, inspected current code, and continued correctly. That establishes continuity, not a speed benefit. One ordered pair cannot isolate provider latency, prompt caching, or model variation. There is no basis here for a general claim that recovery saves time.

The preparatory Claude investigation and initial checkpoint took 220.58 seconds. Earlier manual bootstrap/adaptation effort was not separately timed. Do not combine that figure with the preparation time or present it as end-to-end bootstrap cost.

## Real task output

The retained checkpoint condition supplied the production change. After timing ended, Codex added keyboard regression tests and reviewed the browser behavior. The dashboard shell now offers a focus-only skip link before navigation; Enter focuses the main region and the next Tab reaches content.

- Both new keyboard cases fail against the original shell and pass with the change, for selected-guild and guild-less states.
- All 69 dashboard tests pass. Formatting/lint pass with eight existing warnings.
- Explicit dashboard TypeScript checking has two existing `vite.shared.ts` errors, reproduced on the clean baseline. No new type errors were observed.
- A Chromium fixture rendered the production shell, CSS, theme state, and TanStack router, with authentication responses stubbed. Tab, Enter, continued tabbing, focus visibility, and viewport containment passed at 1280 and 390 pixels in both themes. This is a component audit, not an authenticated application test. Other engines and screen-reader output remain untested.

The bootstrap and measurement files remain on the local trial branch. The [product PR](https://github.com/Zokiio/Mukabi/pull/4) contains only the feature, its tests/dependency, and a brief audit report. Its Vercel preview failed. The CLI could not retrieve build logs because its token is invalid, so the cause is unknown; do not attribute it to the existing local type errors. Human review remains outstanding.

Raw timestamped transcripts, frozen/restored manifests, original and result checkpoints, and scoring criteria are retained under the trial checkout's ignored `.context/trial/measurement/` directory. The report distinguishes measured observations from later verification.

## What repeats

Both projects required a supplied binary, adapted skills with supporting references, a binding, explicit source selection, preservation of existing tracker authority, and separation of disposable evidence from durable guidance. Mukabi adds observed Claude symlink discovery on macOS. It does not establish behavior for other platforms, multiple active tools sharing a setup, link failures, or installer reruns.

Keep the checklist and the init draft. The repeated adaptation work identifies possible future command content, but this trial does not justify the original multi-tool installer or a recovery speed claim. Tighten the source skills' portable binary guidance and measure another bounded continuation only when it answers a concrete remaining question.
