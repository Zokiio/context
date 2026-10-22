# Run a local factory with offline workers

Use this guide to start the service, produce a real candidate commit, verify it with a separate worker, and record acceptance. The example uses a temporary Git repository and the bundled offline fixture. It exercises the platform without a model provider or a target-project pilot.

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

## Inspect the same installation in the browser

Open the service URL and sign in with a personal token from a private token file. The browser and CLI share projects, jobs, evidence, permissions, and lifecycle operations. Keep tokens out of URLs.

The browser retains the token in tab session storage. Sign out to remove it. Browser session restore can retain the credential.

## Recover an interrupted job

Restart `factory serve` with the same `--data-dir`. Preserve the worker's `--receipts` directory, the service worktrees, and retained artifacts. Inspect `jobs show` before deciding the next action.

A repeated mutation needs the same `--request-id` and unchanged payload. A new decision needs a new request ID and the current `--if-version`. The service rejects stale versions and conflicting reuse.

For a running job, use `jobs cancel --job <id> --if-version <version>`. A cancellation acknowledgement means that the wrapper received the request. Inspect the terminal outcome before treating the execution as stopped.

An unfinished receipt, lost contact, or an unsupported native reconciliation produces `unknown`. The generic command adapter keeps cancellation unknown when it cannot establish that external agent work stopped. The controlled offline fixture can confirm its own stopped execution. A missed heartbeat does not authorize another execution.

Use `jobs reconcile` to record a reconciliation request. Inspect the exact attempt, worker receipts, and the native tool's execution state. After obtaining evidence, a project administrator can use `jobs resolve` with `--attempt`, `--resolution`, and `--evidence`. Supported resolutions are `still_running`, `not_started`, `stopped_failed`, and `stopped_cancelled`. Record the observed facts rather than inferring a stop from elapsed time.

For a definitively failed job, request `jobs repair` with the current version. The service enforces the admitted repair limit, deadline, and unchanged requirements. It preserves the failed attempt and requires checks and review for the new candidate.

For an interrupted acceptance publication, repeat the exact acceptance request. The retained publication plan reconciles existing bytes before completing the decision. Once publication begins, cancellation is refused because some acceptance files may already exist. Reconcile that publication before treating the job as settled.

## Connect an existing coding-agent command

Run workers on the service's Unix host, where the assigned workspace paths exist. Register a trusted installed executable through `executors register --program`, repeated `--arg` flags, or a `--command-file` containing a JSON argv array. Register required executable checks through `checks register`. Project policy selects permitted executors and mandatory checks.

Each command receives the assignment JSON on standard input. `CTX_FACTORY_ASSIGNMENT` names its retained assignment file, `CTX_FACTORY_CONTEXT` names the admitted source snapshot, and `CTX_FACTORY_ATTEMPT` identifies the attempt. Use an adapter that understands this input or translates it into the existing agent's invocation. The adapter must leave a clean candidate commit in its assigned worktree.

Waymark passes literal argv without a shell. It retains a supervised command identity, actual exit observations, bounded logs, and the candidate commit. The identity does not prove a provider account or native agent session. Executables retain the worker process's operating-system permissions, environment, filesystem access, and network access. Configure those permissions for the trusted tool you intend to run.

The bundled fixture proves only the offline outcomes it executes. This guide requires no engine installation, paid evaluation, provider-backed model run, publication, or deployment.

## Stop the example service

Stop the process started in this shell and retain its temporary directory for inspection:

```sh
kill -TERM "$wm_factory_pid"
wait "$wm_factory_pid"
printf 'Retained factory files: %s\n' "$wm_factory_dir"
```
