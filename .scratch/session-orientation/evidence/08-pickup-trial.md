# Fresh-session pickup observation

On 2026-09-16, workflow actor `Codex agent /root/orientation_pickup_trial` selected [Verify fresh-session pickup on the real project](../../records/session-orientation/issues/08-verify-fresh-session-pickup.md) through orientation. The ticket was unstarted and explicitly committed when selected. Its ID is `9881a140-ff67-47f8-8b61-a7cc173743b2`.

The verification criteria have successful observations. Acceptance authoring remains with the coordinator. This actor changed only the ticket's execution to `in-progress` and added the three evidence files named here. The specification and acceptance checklist remain unchanged by this actor.

The [observation manifest](08-pickup-trial.json) retains the exact command arrays, exits, source paths, digests, inclusion reasons, selection facts, code identity, read-only comparisons and per-criterion results. Repeated source paths use a lossless path table described by `source_path_encoding`. Command arrays remain literal. The [original selected ticket](08-pickup-original-ticket.md) preserves all 4,034 pickup bytes, including the unchecked criteria and `execution: unstarted`. Its whole-file SHA-256 is `9ace0290eb26a33072eb92cd58cf736607a1c68bdc087a3e93eed21b5fa894dc`.

## Reader and pickup order

The fresh session received the workflow objective, explicit scope, exact retained reader and [preflight handoff](../preflight-handoff.md). Its launch supplied no ticket to choose and no earlier planning conversation. Its first reader project query was `ctx orient`, followed by `ctx orient --json`, both from the external capture directory. No limit override was supplied.

The session verified the retained reader's SHA-256 before its first query: `70d37e8989db00f305e05d0ce0f498b7fa00dfca45ab1a6ee74d1f93ff789a21`. The [successful predecessor preflight](07-preflight-verification.json) retains the original test commands and source manifest. The trial confirmed that all 64 reader source files match that manifest, commit `4ccb1f7171082fa97af7455b0d86f56c5228b71d`, and the observed working tree. The binary's build metadata identifies the earlier dirty base `0808609e8de4ad1d80c52cadd8fd0b6fca31ac0d`. Pickup Git HEAD was `9485397082d1551524acba9616ebcbb354f96398`. These identities have separate meanings and remain recorded separately.

Both orientation formats exited 0, with complete evaluation, complete inventory and no diagnostics. The report contained 13 work items and 82 sources totaling 764,843 source bytes. The sole shortlist item was this ticket. The actor chose it after observing unstarted execution, current commitment, ready-for-agent triage, and passing criteria, context, dependency and blocking-decision checks. The predecessor was completed with valid acceptance at pickup.

After selection at `2026-09-16T13:25:21.606391+00:00`, the session requested the ticket's context with the same explicit project and allowed-source roots. Context exited 0 with `complete: true`, `traversalComplete: true`, 26 sources totaling 243,809 bytes, and no diagnostics. The actor consumed every full source text with its path and reasons before authoring. Large displays were split, and portions hidden by tool-output truncation were displayed again. All delivered text bytes and digests match the observed source files.

## Verification and authoring boundary

The text and JSON comparison passed for project identity and goals, commitment order, work groups, all work-item states and checks, dependency edges, acceptance details, and all orientation source paths, digests and reasons. All five completed reader tickets remain in the inventory with valid acceptance explanations and stay off the shortlist. No reader or comparison failure occurred.

The read-only boundary ran from `2026-09-16T13:24:28.848819+00:00` through `2026-09-16T13:32:09.036578+00:00`. All 239 project-file entries and 709 `.git`-file entries compare equal. Git HEAD, branch, status, index, working diff, staged diff and refs also compare equal. The snapshots cover tracked, untracked and ignored files. File access times and directory modification times are excluded. The comparison establishes the two observed states and does not assert atomic filesystem isolation.

Raw captures stayed outside the repository at `/var/folders/hd/nxsy37l9371g67_scvcnpxb00000gn/T/ctx-pickup-trial-1vl7i7uq`. The manifest records their hashes and the capture method. The coordinator reported no repository writes during this boundary.

At `2026-09-16T13:33:39.289089+00:00`, explicit authoring began. The actor replaced only `execution: unstarted` with `execution: in-progress`. A later orientation exited 0 with both completeness values true and no diagnostics. The ticket appeared in the in-progress list, remained ready, and was absent from the empty shortlist. Its requirement fingerprints were unchanged. Coordinator provenance corrections occurred after the read-only boundary. The later report is retained separately and does not replace the pickup snapshot.

## Separate guidance and closeout

Repository guidance, domain guidance, the retained-reader handoff, successful preflight, and the external technical-writing and unslop skills informed the trial separately. Their paths, digests and reasons appear under `unlinked_guidance` in the manifest. Issue-tracker guidance and the task-context skill were also delivered through authored Context links. The absence of other guidance from task context follows authored selection and is not a reader defect.

Criteria 1 through 9 have successful observations in `criteria_results`. Criterion 10 remains pending. The coordinator owns checklist completion, current Acceptance authoring, completed execution and final orientation validation. Historical acceptance establishes the reader's recorded comparisons. This trial does not infer automatic relevance discovery, evidence grading, authenticated human approval, or current-checkout certification from that history.
