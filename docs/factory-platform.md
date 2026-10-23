# Run a local factory with offline workers

Use this guide to start the service, produce a real candidate commit, verify it with a separate worker, and record acceptance. The example uses a temporary Git repository and the bundled offline fixture. It exercises the platform without a model provider or a target-project pilot.

For a single person, [solo mode](#run-everything-for-one-person) replaces the user, token, and grant setup below with one command, and [real coding agents](#connect-claude-code-or-codex) replace the fixture.

Use Go 1.25 or later, Git, Python 3, and a Unix host for process workers. Keep the following commands in one shell. Reserve port `8741` for this service, or change `wm_factory_url` and `--listen` together.

## Build and bootstrap the service

Create private temporary storage and build the executable from the Waymark checkout:

```sh
umask 077
wm_factory_source=$(git rev-parse --show-toplevel)
wm_factory_dir=$(mktemp -d)
wm_ctx="$wm_factory_dir/ctx"
wm_factory_url=http://127.0.0.1:8741
go -C "$wm_factory_source" build -o "$wm_ctx" ./cmd/ctx
"$wm_ctx" factory --help
```

Bootstrap the first administrator before the service starts:

```sh
"$wm_ctx" factory init --data-dir "$wm_factory_dir/state" \
	--username operator --token-file "$wm_factory_dir/admin.token"
```

The JSON result identifies the administrator and token file. The command writes the token into a new file with mode `0600`. It refuses an existing output file and a second bootstrap.

Define two shell helpers. `wm_admin` sends authenticated requests. `wm_json` extracts a field from JSON output.

```sh
wm_admin() {
	"$wm_ctx" factory --url "$wm_factory_url" \
		--token-file "$wm_factory_dir/admin.token" "$@"
}
wm_json() {
	python3 -c 'import functools,json,sys; print(functools.reduce(lambda value,key: value[key], sys.argv[1].split("."), json.load(sys.stdin)))' "$1"
}
"$wm_ctx" factory serve --data-dir "$wm_factory_dir/state" \
	--listen 127.0.0.1:8741 >"$wm_factory_dir/service.log" 2>&1 &
wm_factory_pid=$!
for wm_try in 1 2 3 4 5; do
	if wm_admin me >"$wm_factory_dir/identity.json" 2>/dev/null; then
		break
	fi
	sleep 1
done
wm_admin me
```

The final command reports the authenticated administrator. If startup fails, inspect `$wm_factory_dir/service.log` before continuing.

Keep the service data directory and worker receipts across restarts. One service owns the installation's database and worktrees. A second service cannot open the same data directory.

## Register a disposable fixture project

Create a repository with an existing Project and a ready WorkItem. Generated evidence and Acceptance records stay outside Git.

```sh
wm_project_dir="$wm_factory_dir/project"
wm_project_id=$(python3 -c 'import uuid; print(uuid.uuid4())')
wm_work_id=$(python3 -c 'import uuid; print(uuid.uuid4())')
mkdir -p "$wm_project_dir/records"
git -C "$wm_project_dir" init
cat >"$wm_project_dir/.gitignore" <<'IGNORE'
.factory-evidence/
records/acceptances/
IGNORE
cat >"$wm_project_dir/records/project.md" <<PROJECT
---
type: Project
id: $wm_project_id
title: Offline factory example
---

## Goals
Exercise the local factory with an offline fixture.

## Current commitments
- [Produce a fixture candidate](ticket.md)

## Open decisions
None
PROJECT
cat >"$wm_project_dir/records/ticket.md" <<TICKET
---
type: WorkItem
id: $wm_work_id
title: Produce a fixture candidate
triage: ready-for-agent
execution: unstarted
---

## Acceptance criteria
- Produce waymark-fixture.txt in a real candidate commit.
- Verify that the candidate is clean and the fixture file contains data.

## Blocked by
None

## Blocked by decisions
None

## Acceptance

TICKET
git -C "$wm_project_dir" add -- .gitignore records
git -C "$wm_project_dir" -c user.name='Offline operator' \
	-c user.email=operator@example.invalid commit -m 'Add fixture requirements'
wm_base=$(git -C "$wm_project_dir" rev-parse HEAD)
```

Register the bundled fixture commands and an admin-owned project policy:

```sh
wm_admin executors register --key offline-agent --fixture
wm_admin checks register --key offline-check --fixture \
	--environment 'offline temporary repository'
wm_admin projects register --bundle "$wm_project_dir/records" \
	--repo "$wm_project_dir" --allow-source "$wm_project_dir" \
	--executor offline-agent --required-check offline-check \
	--max-repairs 1 --max-duration 1h --fixture
wm_admin projects inspect --project "$wm_project_id"
```

Inspect the returned work item for readiness `ready` and eligibility `true`. Registration retains the IDs and authored records. The fixture profiles use the absolute path of the current `ctx` executable.

For an existing project, supply its authoritative bundle, repository, and allowed source roots. Select trusted executor and required-check profiles, and omit `--fixture` for real project policy. An allowed source root outside the bundle is required for retained acceptance evidence.

## Create separate worker and reviewer identities

Create two worker accounts and a reviewer account:

```sh
wm_impl_user=$(wm_admin users create --username fixture-implementer \
	--kind worker | wm_json id)
wm_verify_user=$(wm_admin users create --username fixture-verifier \
	--kind worker | wm_json id)
wm_review_user=$(wm_admin users create --username fixture-reviewer \
	--kind human | wm_json id)
wm_admin users token --user "$wm_impl_user" \
	--out "$wm_factory_dir/implementer.token" --label offline-implementation
wm_admin users token --user "$wm_verify_user" \
	--out "$wm_factory_dir/verifier.token" --label offline-verification
wm_admin users token --user "$wm_review_user" \
	--out "$wm_factory_dir/reviewer.token" --label offline-review
wm_admin projects grant --project "$wm_project_id" --user "$wm_impl_user" \
	--capability read --capability worker_implement
wm_admin projects grant --project "$wm_project_id" --user "$wm_verify_user" \
	--capability read --capability worker_verify
wm_admin projects grant --project "$wm_project_id" --user "$wm_review_user" \
	--capability read --capability review
```

This example operates distinct accounts from one machine. For collaboration, give each person their own credential. The service requires a reviewer account distinct from the coordinator and implementation worker. Worker accounts cannot record review or acceptance.

The `human` account kind identifies an interactive user account, which an agent can also operate through the API. Accepted records attribute the workflow to the authenticated account. The platform does not infer personal human approval from an account category or API review, and it omits inferred human approvals from the Acceptance record.

Keep token files private with mode `0600` or `0400`. The CLI refuses public or symlinked token files. Save the nonsecret credential ID from token issuance if you need to revoke it later with `users revoke-token --token-id`.

## Submit and execute one bounded job

Submit the ready work item with a fixed request ID, base commit, repair limit, and deadline:

```sh
wm_deadline=$(python3 -c 'from datetime import datetime,timedelta,timezone; print((datetime.now(timezone.utc)+timedelta(minutes=30)).strftime("%Y-%m-%dT%H:%M:%SZ"))')
wm_job_id=$(wm_admin jobs submit --project "$wm_project_id" \
	--work-item "$wm_work_id" --base "$wm_base" --executor offline-agent \
	--repair-limit 1 --deadline "$wm_deadline" \
	--request-id offline-submit | wm_json id)
wm_admin jobs show --job "$wm_job_id"
"$wm_ctx" factory --url "$wm_factory_url" \
	--token-file "$wm_factory_dir/implementer.token" worker \
	--role implementation --project "$wm_project_id" \
	--receipts "$wm_factory_dir/implementation-receipts" --once
wm_admin jobs show --job "$wm_job_id" | wm_json status
```

The implementation receipt names a real commit, and the job reaches `verifying`. Inspect the attempt's command observations and retained artifacts. Process success establishes a candidate, while acceptance still requires independent evidence.

Execute the required check with the verifier's credential:

```sh
"$wm_ctx" factory --url "$wm_factory_url" \
	--token-file "$wm_factory_dir/verifier.token" worker \
	--role verification --project "$wm_project_id" \
	--receipts "$wm_factory_dir/verification-receipts" --once
wm_admin jobs show --job "$wm_job_id" >"$wm_factory_dir/job.json"
wm_json status <"$wm_factory_dir/job.json"
```

The job reaches `ready_for_review`. The service allocates a separate clean worktree for the exact candidate. Each verification assignment executes one required check. With several required checks, run the verifier again or omit `--once` to poll continuously.

## Review the candidate and record acceptance

Read the candidate change and the independent check results before approving:

```sh
wm_candidate=$(wm_json candidateRevision <"$wm_factory_dir/job.json")
wm_requirements=$(wm_json admission.ticketSHA256 <"$wm_factory_dir/job.json")
wm_version=$(wm_json version <"$wm_factory_dir/job.json")
git -C "$wm_project_dir" show "$wm_candidate" -- waymark-fixture.txt
wm_admin jobs show --job "$wm_job_id"
```

Use the reviewer account's token to approve the inspected candidate and requirement fingerprint:

```sh
"$wm_ctx" factory --url "$wm_factory_url" \
	--token-file "$wm_factory_dir/reviewer.token" jobs review \
	--job "$wm_job_id" --if-version "$wm_version" \
	--candidate "$wm_candidate" --requirements "$wm_requirements" \
	--decision approved --summary 'Reviewed the offline fixture change and independent check.' \
	--request-id offline-review
wm_admin jobs show --job "$wm_job_id" >"$wm_factory_dir/job.json"
wm_version=$(wm_json version <"$wm_factory_dir/job.json")
wm_admin jobs accept --job "$wm_job_id" --if-version "$wm_version" \
	--candidate "$wm_candidate" --requirements "$wm_requirements" \
	--summary 'Accepted the verified offline fixture outcome.' \
	--request-id offline-accept >"$wm_factory_dir/accepted.json"
wm_json status <"$wm_factory_dir/accepted.json"
wm_json acceptanceStatus <"$wm_factory_dir/accepted.json"
```

Expect `accepted` and `valid`. Acceptance rechecks current requirements, the exact candidate, every required check, and independent approval. Changed or unavailable evidence prevents acceptance.

Inspect the actual authored result through the existing reader:

```sh
"$wm_ctx" orient --bundle "$wm_project_dir/records" \
	--allow-source "$wm_project_dir" --json >"$wm_factory_dir/orientation.json"
python3 -c 'import json,sys; report=json.load(sys.stdin); assert report["complete"]; assert report["workItems"][0]["acceptance"]["status"]=="valid"; print("reader acceptance: valid")' <"$wm_factory_dir/orientation.json"
"$wm_ctx" context --bundle "$wm_project_dir/records" \
	--allow-source "$wm_project_dir" --ticket ticket.md
```

The original ticket now has `execution: completed` and one current Acceptance link. Retain the ignored decision and evidence files together. A fresh Git checkout reports unknown acceptance until those files are restored or the outcome is verified again.

## Merge accepted work

Acceptance does not change any branch: the candidate stays on its own commit. Merge it explicitly so that work depending on it builds on it:

```sh
wm_admin jobs merge --job "$wm_job_id" --if-version "$(wm_admin jobs show --job "$wm_job_id" | wm_json version)" \
	--candidate "$(wm_admin jobs show --job "$wm_job_id" | wm_json candidateRevision)"
```

Merging requires a human with the `merge` capability, and installation administrators have it. The service rechecks that the acceptance is still valid and that the version and candidate match. It then builds a `--no-ff` merge commit in a temporary worktree. The commit names the job, candidate, requirements, acceptance record, and approver, and it is pinned under `refs/waymark/merges/<job>`.

The merge is `verifying` until the project's required checks pass **on the merge commit**. Verification workers run them in clean worktrees, exactly as they verify candidates. These merge checks never change the job's own status, and they run even after the job's admission deadline. When every check passes, the branch moves:

- **checkout:** the target branch is checked out in the project repository, so it is fast-forwarded there from the tip the merge was built on.
- **ref:** no worktree has the target branch checked out, so the branch moves by compare-and-swap.
- **contained:** the branch already contains the candidate, for example after a merge by hand. The service records that tip at once, without checks.

Several things fail the merge and leave every branch as it was:
- a failed or unknown merge check;
- an expired lease, or a controller restart during a check;
- the branch moving during verification;
- local changes that block the fast-forward.

The failure reason is kept, unclaimed checks are cancelled, and the merge can be retried. A retry can rebuild an identical commit, so only the latest generation of checks decides it. A successful merge adds a `job_merged` event and a note under the ticket's Comments; Comments are outside the requirement fingerprints, so the acceptance stays valid.

Admission refuses work whose `Blocked by` prerequisite was accepted by a factory job that is not yet `merged`, or whose base revision does not contain that merge.

## Approve the change in one decision

`jobs approve` reviews, accepts, and merges a verified candidate as one decision:

```sh
wm_admin jobs approve --job <job> --if-version <version> \
	--candidate <commit> --requirements <fingerprint> --finding missing_test
```

The caller needs the review, accept, and merge capabilities. Unless the project allows coordinator review, the caller must not be the person who submitted the job. The service checks every capability before it records anything. It then records the review, the acceptance, and the merge in order. Each step keeps its own checks: acceptance rechecks the evidence, and the merge waits for the checks on the merge commit. Every record carries the same decision ID, so metrics count one decision. If a step fails, repeating the request with the same request ID finishes the remaining steps.

`--no-merge` accepts without merging. `--finding` records what the checks missed: `wrong_requirement`, `missing_test`, `bug`, `hard_to_read`, or `out_of_scope`. A review sent through the API can carry the same `findings`.

## Work from the board in the browser

Open the service URL and sign in with a personal token from a private token file. The browser and CLI share projects, jobs, evidence, permissions, and lifecycle operations. Keep tokens out of URLs.

The board places every open work item in one column: Up next, Plan, Build, Approve, Merge, or Done. A card either needs you, is running, or waits for something named on it. It shows that state, a few short facts such as "3 questions" or "Checks passed", and a bar for its six steps: plan, agree, build, checks, approve, and merge. The header counts what needs you, and its button opens the first one. Add work writes a new work item, as described in [Add and edit work items](#add-and-edit-work-items). The Graph view draws the same cards by their `Blocked by` links: each group of connected work is a row, left to right in dependency order, with dashed arrows for dependencies that are still open. `projects board` prints the same projection.

A change needs two decisions from a person, each made in a side panel:
- **Agree what done means**, before the build. Answer the refiner's questions, read or edit the sharpened requirements, or read the locked tests next to the requirement each one checks. The test author's concerns and any requirement without a test come first.
- **Approve the change**, after the checks pass. The panel shows the diff, the requirements no test covers, and a merge switch. Ask for a fix or reject from the same panel.

Agents write the tests, build, and repair the change; workers verify and merge it. None of those steps needs a decision. Insights shows the [measured numbers](#measure-the-factory), How much to ask you holds the [ask policy](#decide-how-often-the-factory-asks-you), and Agents holds [each agent's instructions](#tell-the-agents-more-about-the-project). The theme follows the system setting and can be switched to light or dark.

The browser retains the token in tab session storage. Sign out to remove it. Browser session restore can retain the credential.

## Recover an interrupted job

Restart `factory serve` with the same `--data-dir`. Preserve the worker's `--receipts` directory, the service worktrees, and retained artifacts. Inspect `jobs show` before deciding the next action.

A repeated mutation needs the same `--request-id` and unchanged payload. A new decision needs a new request ID and the current `--if-version`. The service rejects stale versions and conflicting reuse.

For a running job, use `jobs cancel --job <id> --if-version <version>`. A cancellation acknowledgement means that the wrapper received the request. Inspect the terminal outcome before treating the execution as stopped.

An unfinished receipt, lost contact, or an unsupported native reconciliation produces `unknown`. The generic command adapter keeps cancellation unknown when it cannot establish that external agent work stopped. The controlled offline fixture can confirm its own stopped execution. A missed heartbeat does not authorize another execution.

Use `jobs reconcile` to record a reconciliation request. Inspect the exact attempt, worker receipts, and the native tool's execution state. After obtaining evidence, a project administrator can use `jobs resolve` with `--attempt`, `--resolution`, and `--evidence`. Supported resolutions are `still_running`, `not_started`, `stopped_failed`, and `stopped_cancelled`. Record the observed facts rather than inferring a stop from elapsed time.

If a command finished and its terminal report could not be delivered, preserve the successful local receipt. The worker API can accept that original terminal observation from an uncertain attempt under the same worker identity, with the same attempt, dispatch, native run, candidate, and evidence. Replay it through `POST /v1/worker/reports`; do not launch another command or classify a known success as a failure. The built-in polling worker does not reacquire an uncertain attempt, and `jobs reconcile` records a request without replaying the receipt. This recovery currently requires an adapter or operator using the [worker API](factory-http.md).

For a definitively failed job, request `jobs repair` with the current version. The service enforces the admitted repair limit, deadline, and unchanged requirements. It preserves the failed attempt and requires checks and review for the new candidate.

For an interrupted acceptance publication, repeat the exact acceptance request. The retained publication plan reconciles existing bytes before completing the decision. Once publication begins, cancellation is refused because some acceptance files may already exist. Reconcile that publication before treating the job as settled.

## Connect Claude Code or Codex

Run workers on the service's Unix host, where the assigned workspace paths exist. `ctx factory agent` is an executor adapter for native coding agents. It reads the assignment on standard input and gives the agent the rules, the protected paths, and every admitted context source with its digest and inclusion reasons. It then commits what the agent left in the worktree as the candidate. The agent never commits, so the candidate is exactly the agent's change. A run that changes nothing or reports an error fails without committing.

Register an agent through a JSON argv file:

```sh
printf '["%s", "factory", "agent", "--harness", "claude", "--model", "sonnet", "--max-budget-usd", "1", "--allow-tool", "Bash(go test:*)"]\n' "$wm_ctx" >"$wm_factory_dir/claude.json"
wm_admin executors register --key claude-sonnet --name "Claude Code" \
	--command-file "$wm_factory_dir/claude.json"
```

Claude Code runs in print mode with `acceptEdits`, so it can edit files in the worktree but is denied any tool the profile does not allow. `--max-budget-usd` caps one attempt's spending. Codex (`--harness codex`) runs in its `workspace-write` sandbox. The attempt log records the agent's session ID, turns, cost, and final message, and the commit message names the job, attempt, harness, model, and session.

Run the worker with a normal login environment. A worker started from inside another agent session inherits that session's variables. Register required executable checks through `checks register`. Project policy selects permitted executors and mandatory checks.

Any other executable can still serve as an executor. Each command receives the assignment JSON on standard input, and absolute paths in its environment. `CTX_FACTORY_ASSIGNMENT` names its retained assignment file, `CTX_FACTORY_CONTEXT` names the admitted source snapshot, and `CTX_FACTORY_ATTEMPT` identifies the attempt. The command must leave a clean candidate commit in its assigned worktree. It may append one-line notes, such as a finished checkpoint, to the file named by `CTX_FACTORY_PROGRESS`. The worker sends the latest note with each heartbeat, and the job history records each change as a `worker_progress` event. Progress is an observation for operators, not evidence.

Waymark passes literal argv without a shell. It retains a supervised command identity, actual exit observations, bounded logs, and the candidate commit. The identity does not prove a provider account or native agent session. Executables retain the worker process's operating-system permissions, environment, filesystem access, and network access. Configure those permissions for the trusted tool you intend to run.

## Protect the tests that verification runs

Verification runs required checks in a clean worktree at the candidate, so a check such as `go test ./...` runs the candidate's own test files. Without protection, an implementer that weakens or deletes a test still passes. Name the tests and expectations the implementer may not change when registering the project:

```sh
wm_admin projects register ... --protect '**/*_test.go' --protect testdata/
```

Patterns are repository-relative. `**` matches any number of directories, and a trailing `/` protects a whole directory. Admission freezes the patterns. When an implementation candidate adds, edits, renames, or deletes a protected file, the job fails before verification is queued. The job history records a `protected_paths_changed` event with the files, the attempt keeps the rejected candidate for inspection, and a repair restarts from the admitted base. Protect check scripts and fixtures as well as tests: anything a check reads can change its result.

## Flag vague criteria before admission

A local decision model can flag acceptance criteria that may not name a checkable result. Start the service with one installed in Ollama:

```sh
ollama pull kev:0.8b-mxfp8   # kev:0.8b-q8_0 on NVIDIA
"$wm_ctx" factory serve --data-dir "$wm_factory_dir/state" \
	--decision-model kev:0.8b-mxfp8
```

Project orientation then includes `criterionSignals` for open work items: each criterion's probability of naming a checkable result and whether it falls below `--decision-threshold`. The default is 0.25. Rewrite flagged criteria before admission freezes them. Answers are cached by model, question version, and input digest, and admission copies the cached answers into the job without calling the model.

The signals are advisory. They never affect readiness, admission, or acceptance, and an unreachable model is reported as `unavailable`. The probabilities are uncalibrated. In trials, Kev 0.8B separated 37 real criteria from 6 vague controls well, at about 60 ms per criterion on an M4, but its absolute values were low. Compare criteria with each other rather than trusting a single number. In a later trial, vague criteria scored 0.27 to 0.30 and the trial ran with `--decision-threshold 0.32`.

## Refine criteria before admission

Admission freezes a work item's criteria, and protected tests then hold the implementation to them. A vague criterion is therefore locked in for the whole job. A refinement fixes it first:

```sh
wm_admin refine start --project "$wm_project_id" --work-item "$wm_work_id" --wait
wm_admin refine start --project "$wm_project_id" --work-item "$wm_work_id" \
	--previous ref_... --answer "Strict code points, no normalization." --wait
wm_admin refine accept --id ref_... --criterion "..." --criterion "..." --answer "..."
```

The refiner reads the ticket, its context, the decision model's flags, the protected paths, and any earlier proposal and answers. It returns rewritten criteria and at most three blocking questions. Each question says what its answer unblocks and offers two to four likely answers, best guess first. `ctx factory refiner` runs Claude Code with only the Read, Grep, and Glob tools and schema-checked JSON output, so it cannot edit anything. The decision model scores the current criteria before and the proposal after.

Answer the questions and refine again, reject the proposal, or edit and accept it. Acceptance rewrites the Acceptance criteria section and appends who accepted, which refiner proposed it, and the answers under Comments. The write is an expected-revision write: if the ticket changed after the refinement started, the refinement becomes `stale` and the ticket is left alone. Refinement is refused while an active job holds the work item, one refinement runs at a time per work item, and a run interrupted by shutdown is recorded as failed.

Configure a refiner with `serve --refiner-command-file` (a JSON argv that reads a `RefinerInput` on stdin and writes a `RefinerOutput` on stdout). Solo mode uses read-only Claude Code when it is installed: `--refiner-model` (default `sonnet`) and `--refiner-budget-usd` (default 0.50) tune it, and `--no-refiner` disables it. In the browser, open the item's card on the board.

In a trial on a ticket whose criteria read "Add a palindrome helper", "handle unicode properly", and "be fast", the first pass took 27 s and $0.47 and turned them into seven checkable criteria with two questions. After answers, a second pass took 36 s and $0.09. It also found that an earlier accepted candidate had never been merged, which its holdout test needed. Kev scored the original criteria 0.27 to 0.29, above the 0.25 default threshold, so it did not flag them. The proposed criteria scored 0.37 to 0.64. Use the scores to compare criteria rather than relying on the flag alone.

## Write holdout expectations

Holdout expectations are tests written from a work item's criteria, before implementation, by an author that never implements the work. They live under the project's protected paths, so the implementer cannot change them:

```sh
wm_admin expectations start --project "$wm_project_id" --work-item "$wm_work_id" --wait
wm_admin expectations show --id exp_...
wm_admin expectations accept --id exp_...
wm_admin jobs submit --project "$wm_project_id" --work-item "$wm_work_id" --expectations exp_... \
	--executor claude-sonnet --deadline "$wm_deadline"
```

The author runs in a temporary worktree at the base revision; nothing in your checkout changes. The service keeps its work only if it changed at least one file and every changed file matches a protected path. It then commits the files and pins them under `refs/waymark/expectations/<id>`. When the service refuses the work, the failed record still keeps the author's notes and cost, so the person can see why.

Read the patch, then accept it (a human decision) or reject it. Acceptance is refused if the criteria changed after the tests were written.

The author also reports which tests check each criterion, and any concerns about the criteria themselves. A concern might be a criterion that no test can check, or an example that contradicts its own rule. Criteria without a test are what the reviewer checks by hand after the build. The browser lists them, and the concerns, before the tests.

A job submitted with `--expectations` starts from that commit:
- the implementer is told the holdout tests exist and must pass;
- any candidate that changes them is rejected;
- verification runs them;
- they reach a branch only together with the implementation, through the verified merge, so no branch carries tests for work that does not exist yet.

Register a project with `--require-expectations` to admit jobs only this way. `ctx factory expectation-author` runs Claude Code with edit access, an optional allowlist of test commands, and a spending cap. Solo mode uses it when Claude Code is installed: `--expectation-model` (default `sonnet`) and `--expectation-budget-usd` (default 1) tune it, and `--no-expectations` disables it. Configure another author with `serve --expectation-command-file`.

## Agree what done means in one step

`POST /v1/projects/{id}/plan` records the decision before the build, which is what the browser's plan panel sends. Depending on what the item has, it does one of the following:
- sends answers back to the refiner;
- accepts a refinement, or criteria the person edited;
- accepts locked tests.

It then starts the next step. That step is locked tests when the project has a test author, and otherwise the build.

With `continue`, the person also lets the factory go on to the build without asking again. The [autopilot](#decide-how-often-the-factory-asks-you) carries the go-ahead out and acts as that person. The go-ahead stops at anything new: another question, a requirement the decision model still flags, or a concern from the test author. The board then says why it stopped. Rejecting the refinement or the tests withdraws the go-ahead.

## Decide how often the factory asks you

A project administrator chooses when the factory stops for a person:

```sh
wm_admin projects ask --project "$wm_project_id" \
	--before-build when_unclear --after-build when_untested --autopilot
```

| Setting | Values |
| --- | --- |
| `--autopilot` | Starts ready work and takes the steps that need no decision: sharpening, writing tests, building, repairing after a failed check, and retrying a merge that failed for a reason other than a check, at most twice. |
| `--before-build` | `always` asks to agree every plan. `when_unclear` asks only when a requirement looks vague or cannot be scored, the refiner has a question, or the test author raises a concern. `never` asks only for refiner questions, requirements that still look vague after sharpening, and test-author concerns. Both take effect through `--autopilot`. |
| `--after-build` | `always` asks to approve every change. `when_untested` approves and merges a change when every requirement has a locked test, and asks otherwise. `never` approves and merges every change that passes its checks. |

Every value except `always` delegates decisions, so the person saving the policy must hold the capabilities those decisions need. The autopilot acts as that person and marks each record `delegated` with its reason. `--after-build never` unlocks after 10 merges with no reported problem; the board shows the count.

The autopilot runs in `solo` by default (`--no-pilot` turns it off) and in `serve` with `--pilot`. It starts one build per project at a time. When a step fails, the card shows why, and the autopilot does not retry that step until the card changes.

Report merged work that turned out wrong:

```sh
wm_admin jobs report-problem --job "$wm_job_id" --summary 'Truncate splits flags.'
```

A report resets the trust count and lowers `--after-build` one level, from `never` to `when_untested` or from `when_untested` to `always`.

## Add and edit work items

Work items are Markdown records in the project's repository; the board reads them. Add one from the board's Add work button, or:

```sh
wm_admin work add --project "$wm_project_id" --title 'Sort words by length' \
	--description 'The preview lists headings with their shortest words first.' \
	--criterion 'SortByLength([]string{"ccc", "a"}) is []string{"a", "ccc"}.' \
	--blocked-by "$wm_work_id"
```

Waymark writes a new record next to the project's other work items. It has a new ID, `triage: ready-for-agent` and `execution: unstarted`, the description as prose before the first section, and the Acceptance criteria, Blocked by, Blocked by decisions and Acceptance sections. Waymark also lists the record under the project's Current commitments, so it becomes ready work once its dependencies are done. Dependencies must be work items in the same project. Waymark never commits; commit the new files with your other changes.

`work show` prints a work item's title, description, requirements and dependencies, and `work edit` rewrites them; a flag left out keeps its current value. An edit changes only those parts, keeps every other section, and adds a line under Comments saying who edited it. The write is an expected-revision write: an edit made from an older copy of the record is refused. While a build holds the work item, its requirements are frozen and edits are refused. On an item's page in the browser, Edit changes the same fields in place. As you type, the decision model marks each requirement as checkable or vague.

## Tell the agents more about the project

Each agent follows built-in rules that the service's guarantees rely on. A project administrator can add instructions for each agent, for example the project's conventions:

```sh
wm_admin projects instructions --project "$wm_project_id" \
	--test-author 'Use table-driven tests.' --builder 'Give every exported name a doc comment.'
```

`--refiner`, `--test-author` and `--builder` set the instructions for the agent that sharpens requirements, the one that writes locked tests, and the one that builds the change. A flag you leave out keeps its current text, and an empty one clears it. Each agent gets at most 4,000 characters. The browser's Agents page shows each agent's built-in rules next to the same fields.

The instructions follow the agent's built-in rules in its prompt, which tells it that the rules win where the two disagree. They cannot lift a guarantee: the service still rejects a candidate that touches protected paths, and a test author's change outside them. A run keeps the instructions in force when it started. A refinement and a test run record theirs, and a job freezes the builder's at admission, so changing them never alters running work.

## Measure the factory

```sh
wm_admin projects metrics --project "$wm_project_id"
```

Insights shows the same numbers, all measured from the installation's own records:
- jobs by outcome;
- first-pass acceptance, meaning accepted without repair;
- repairs and protected-path rejections;
- merge checks and merge failures;
- median time from admission to acceptance and from acceptance to merge;
- human decisions, and decisions per merged job, counting one decision per click however many records it writes;
- decisions the autopilot made for a person;
- refinement and expectation decisions;
- reviewer findings by kind, reported problems, and merges since the last problem;
- agent cost in total and per merged job.

Implementation cost comes from `agent-result.json`, which the agent adapter writes to `CTX_FACTORY_RESULT` and the worker attaches to the attempt. Refinement and expectation costs come from their own run records.

## Run everything for one person

`ctx factory solo` runs the service on loopback with an implementation worker and a verification worker in the same process:

```sh
ctx factory solo --data-dir ~/.waymark/factory --decision-model kev:0.8b-mxfp8
```

The first start creates an `operator` administrator and two worker identities, and keeps their tokens under `solo/` in the data directory. Workers are enrolled in every registered project automatically. When `claude` or `codex` is installed, solo mode registers `claude-agent` (capped at $2 an attempt) and `codex-agent` executor profiles. A project must still allow one before it runs. Register the project with `--solo-review` so the operator can review jobs they submitted. The autopilot runs unless `--no-pilot` is given, and it acts only on a go-ahead or a saved ask policy. The reviewer must still be human and never the implementer. Solo mode refuses non-loopback listeners. Use `serve` with TLS for shared access.

The bundled fixture proves only the offline outcomes it executes. The agent adapter was exercised once with Claude Code against a disposable Go repository. Codex was exercised only through argument construction.

## Stop the example service

Stop the process started in this shell and retain its temporary directory for inspection:

```sh
kill -TERM "$wm_factory_pid"
wait "$wm_factory_pid"
printf 'Retained factory files: %s\n' "$wm_factory_dir"
```
