# Isolate factory execution and bound its spend

The factory runs coding agents and the code they write with the operator's own permissions. A stopped agent never counts as stopped, and nothing limits total spend. This spec settles how the factory confines execution, moves code into and out of that confinement, stops it with proof, runs on other hosts later, and caps spend. The user approved the four decisions below on 25 October 2026.

## What the code does today

Observed in `dev` at `afe9989`:

- Git worktrees are the only isolation. `factoryworker.Process` starts each command with the worker's `os.Environ()`, in a worktree that shares `.git` (hooks, refs, objects, config) with the project repository.
- Candidate code runs as the operator's OS user in three places. The builder runs `go test`, the expectation author runs its tests, and verification runs the required checks. In `solo` mode, that code can read `<data-dir>/solo/operator.token`. The file has mode 0600, but it belongs to the same user. The operator token can approve and merge the change that read it.
- The refiner, expectation author and judge run inside the service process through `exec.CommandContext`, not through workers. The refiner and judge get only `Read`, `Grep` and `Glob`. The expectation author can edit files and run tests.
- Every non-fixture cancellation ends `unknown`. The worker signals its own process group and then reports that it cannot establish whether native execution stopped. A descendant that calls `setsid` leaves the group unnoticed. A deadline takes the same path, so each one needs an administrator's `jobs resolve`.
- Workers must run on the service host, because assignments carry service-owned worktree paths.
- The only spending control is Claude Code's own `--max-budget-usd`, written into executor argv. Solo uses $2 per build attempt, $0.50 per refinement or judgment, and $1 per expectation suite. Spend becomes known only when a run ends, from the agent's self-reported `total_cost_usd`. Codex reports no cost. Metrics leave out judge runs.
- No control stops the whole installation. `--no-pilot` takes effect only at startup.

## Decisions

### Docker comes first

Sandboxed runs use OCI containers through the `docker` CLI. Any Docker-compatible runtime works: Docker Engine, Docker Desktop or colima. Colima runs Docker 28.4 on the author's machine.

Docker confirms stops on both macOS and Linux and also confines the network. When a container's first process exits, the kernel ends every process in its PID namespace, and the daemon reports the container's state. An OS sandbox wrapper (Seatbelt or bubblewrap) cannot confirm a stop on macOS. A separate OS user can confirm a stop, but gives no network control and needs privileged setup. Open-Inspect reaches the same conclusion on Linux by a harder route. It walks `/proc`, takes pidfds and uses a subreaper, and its own tests treat a killed supervisor as "not containment proof". Fabro's Docker sandbox stops containers best-effort and never re-checks the final state. This spec requires that re-check.

Running in Docker costs an image with Git, the toolchain and the agent CLI, and Claude must sign in with a named secret: a `claude setup-token` token or an API key. The macOS keychain login does not reach a Linux container.

### Deny egress by default, allow listed hosts through a proxy, and keep secrets on the worker host

A sandbox has no network unless its profile lists hosts. When it does, the only route out is a proxy that Waymark runs in a sidecar container on the attempt's internal Docker network. The proxy allows `CONNECT` to the listed hostnames only and logs every attempt. Fabro's Docker sandbox offers only all-or-nothing networking and sends allowlists to Daytona. A proxy gives the local backend the allowlist without a hosted provider.

Secret values stay in a private file on the worker host. A profile lists secret names. The worker sets those variables in that one container, and nowhere else: not in assignments, prompts, logs, artifacts or the service database. Waymark's own tokens never enter a sandbox. Credentials for pushing or opening pull requests belong to the controller, never to a build sandbox.

### Specify remote workers now and build them when a second host is needed

Sandboxed runs already need a workspace transport that does not share `.git`. This spec defines that transport so that remote workers reuse it unchanged. The remote claim path waits until a pilot needs a second machine; its tickets stay in `needs-triage` until then.

### Reserve each run's cap against job and project budgets, and add one installation halt

Before any agent run starts, the controller reserves the run's enforceable cap against the job's budget and the project's monthly budget. When a run would exceed either budget, it does not start. CodeAF charges each run when it finishes and lets in-flight work run on, so it can overshoot by every concurrent run. Reserving first makes each budget a ceiling. The only remaining overshoot is a harness exceeding its own cap, which the run records.

An installation-wide halt stops all new execution and stops every live run. It never pretends an unconfirmed stop happened. Such runs stay held and visible until reconciled.

## Where each process runs

Waymark's own processes stay on the host: the worker, the agent adapter, receipts, the egress proxy's control and the stop logic. Only the native agent, the commands it spawns, and check commands run inside the container. The worker's token, receipt directory and data directory are never mounted into a sandbox.

- **Required checks.** A check profile with a sandbox runs its argv inside the container.
- **Builds.** An executor profile may carry a sandbox only when its command is the built-in `ctx factory agent` adapter. The adapter runs on the host and starts the native agent inside the container through the runtime. Sandboxing other executor commands waits for a real need. A project that requires sandboxes refuses them.
- **Expectation author.** It uses the same launcher as builds.
- **Refiner and judge.** They stay unsandboxed in this plan. They have no shell, no edit tool and no network tool. Their exposure is that they can read host files into text that project readers see, and the judge's decisions route work automatically. The same launcher applies if that exposure becomes unacceptable.

## Sandbox profile

Executor and check profiles gain an optional `sandbox` object. Profiles are immutable, so changing a sandbox needs a new profile key, as changing a command does today. Admission freezes it with the rest of the job's policy.

- `backend`: `docker`.
- `image`: an image ID or registry digest. Registration refuses a mutable tag. The check result's `environment` records the exact image.
- `cpus`, `memoryMB`, `pids`: required limits.
- `egress`: hostnames the sandbox may reach on port 443. Defaults to none.
- `secrets`: secret names to set from the worker's secret file. Defaults to none.

Every sandbox runs as a non-root user, with no added capabilities, `no-new-privileges`, no host mounts except the read-only base bundle, and only the `CTX_FACTORY_*` environment variables and the named secrets. Labels name the installation, worker, attempt and dispatch. The container name derives from the dispatch ID. A repeated delivery therefore cannot start a second container on the same runtime.

Waymark ships a reference Dockerfile for Go projects: Git, the Go toolchain, Node and Claude Code. The operator builds it locally and registers its image ID. Waymark does not depend on a published Waymark image. The reference builder profile sets `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`, so its allowlist can stay at `api.anthropic.com`.

## Code moves in and out as Git bundles

A sandbox never mounts a service worktree or the project's `.git`. A worktree shares hooks, refs and config with the repository, so mounting one would let a confined agent plant a hook that runs the next time the operator commits. Running Git on the host inside a directory the sandbox wrote would also obey that directory's config. `core.fsmonitor` is one way that config can run a program. [ADR 0009](../../docs/adr/0009-sandboxed-runs-exchange-git-bundles.md) records this decision.

1. The service writes a bundle containing the attempt's input revision. For a build, that is the base or the previous candidate. For a check, it is the candidate.
2. The worker mounts that bundle read-only, clones it inside the container, and checks out the input revision.
3. After the adapter commits inside the container, the worker streams `git bundle create - <input>..HEAD` out of the container. It stops reading at a size limit and stores the bytes in the attempt's service-owned bundle directory.
4. The service verifies the bundle against the project repository and confirms that it contains exactly the reported candidate. It fetches the objects and materializes the attempt's service-owned worktree at the candidate. Candidate validation, protected paths, the change view, verification and merge then run unchanged on a worktree that only the service wrote.

The worker reads a dispute report out of the container the same way: as bounded bytes, never by running host tools inside the sandbox's files.

The first version sends full bundles. If transfer time on a large repository becomes a problem, incremental bundles against a revision the worker reports holding are the follow-up.

## Egress proxy

Each sandboxed attempt with a non-empty `egress` list gets a Docker network created with `--internal` and a proxy sidecar. The sandbox joins only the internal network, with `HTTPS_PROXY` and `HTTP_PROXY` pointing at the sidecar. The sidecar also joins an ordinary bridge. The proxy is Waymark code, built from this repository's Dockerfile.

- It allows `CONNECT` to a listed hostname and refuses everything else.
- It writes an `egress.log` artifact with each connection's time, host, port, decision and byte counts.
- When the attempt is cancelled or the installation halts, it refuses new connections and closes open tunnels. That stops provider spend even before the process stop is confirmed. The worker confirms the closure by reading the proxy's state. If the runtime is unreachable, neither the closure nor the stop is confirmed.

An attempt with an empty `egress` list runs with `--network none`. Direct connections from the sandbox fail in both cases. Verification records whether the runtime answers external DNS queries on internal networks, because a DNS answer would leave a residual channel.

The worker reads secrets from `--secrets-file`, a regular private file with the same checks as token files, holding a JSON object of names and values. It passes values to `docker run` through the CLI process environment, never through argv. A missing named secret ends the attempt before any container starts, with the check `unavailable` or the build failed. The service records the reason; it never treats the attempt as started.

## Stopping with proof

For a sandboxed run, the worker treats a cancellation, a deadline or a halt the same way:

1. It closes the run's egress.
2. It sends `SIGTERM` through the runtime, waits a grace period, then sends `SIGKILL`.
3. It waits until the runtime reports the container not running, and removes it.

When the runtime reports the container's processes ended, the worker reports `cancelled` with a `stop-proof.json` artifact. The proof holds the container ID, image, runtime-reported state and exit time, removal result and egress closure. If the runtime is unreachable, or still reports the container running after `SIGKILL`, the outcome stays `unknown` with what the worker tried. This is today's behavior.

The controller accepts `cancelled` for a non-fixture attempt only when the assignment's profile is sandboxed and the report carries a stop proof. Unsandboxed native runs keep reporting `unknown`. A proven stop releases the claim exactly as a resolved cancellation does today. The person's stop hold on the work item stays until they queue it again.

### Settle an interrupted run from its container

A worker that restarts finds an `unknown` receipt for one of its attempts. It looks up that attempt's container by label. If the container is absent or exited, or the worker can stop it with proof, the worker reports a proven stop as the same worker identity. The service then moves the attempt from `unknown` to `cancelled`, or to `failed` when no cancellation was requested. A stop without a delivered candidate is never success. The service never hands an `unknown` attempt to a different worker. An administrator's `jobs resolve` stays the path for everything a worker cannot prove.

## Budgets

Spend means the cost the agent reports for a run. With a subscription, that figure tracks usage rather than an invoice. Sandbox compute is not metered.

- **Run cap.** An executor profile may declare `runCapUSD`. The assignment carries it, and the agent adapter passes it to Claude Code's `--max-budget-usd`. The in-service refiner, expectation author and judge use their configured caps. A harness that can neither enforce a cap nor report cost cannot run in a project with a budget. That rules out Codex today. Admission and claim refuse it with that reason.
- **Job budget.** Project policy sets `jobBudgetUSD`, and admission freezes it. It covers every agent run the job causes: build attempts and repairs, judge runs, and test-author proposals made for its disputes.
- **Project budget.** Project policy sets `monthlyBudgetUSD` per calendar month in UTC. It covers every agent run in the project, including refinements and expectation suites written before any job exists.
- **Reservation.** In the same transaction that hands out an attempt or starts an in-service run, the controller reserves the run's cap against both budgets, and refuses when either has less left than the cap. When the run reports its cost, the reservation settles to that cost. A cost above the cap is recorded as observed, and it can leave a budget overdrawn, which blocks further runs. A run whose cost never arrives, such as an `unknown` attempt, keeps its full cap reserved until it is reconciled.
- **Exhaustion.** A job whose budget is used up waits with the reason "budget reached". The board shows it as a decision for a person: a project administrator raises that job's budget, an attributed and recorded action, or stops the job. A used-up project budget blocks new runs in that project until the month changes or an administrator raises it. Running work finishes inside the caps it already reserved. The autopilot never raises a budget, and an ask policy cannot delegate it.
- **Visibility.** Metrics and the board count judge runs alongside the existing costs. Each card's cost shows settled and reserved spend separately.

Zero means no limit, so existing projects behave as before until an administrator sets a budget. There is no installation-wide dollar budget. The halt covers the installation.

## Installation halt

A human installation administrator engages the halt with a reason through `ctx factory halt`, `PUT /v1/halt`, or a control in the browser. The service stores the halt durably, so a restart comes back halted.

- While halted, worker claims return `no_work` with the reason `halted`. The autopilot skips its passes. The service refuses to start refinements, expectation suites, judgments and merge checks. Record publication already in progress, such as an acceptance, may finish, because it starts no execution.
- Engaging the halt requests cancellation of every live attempt and in-service run. Sandboxed runs follow the proven-stop procedure. Unsandboxed runs keep today's signal-then-`unknown` behavior.
- The board shows a halt banner with the reason, the actor and the time. Each run whose stop is not confirmed shows "Stop not confirmed" with what the worker tried.
- Release is a separate explicit action by an installation administrator. It is allowed while some stops are still unconfirmed, so a lost worker cannot lock the installation. Release lists those runs. It never resumes or re-dispatches held work. Unconfirmed attempts keep their claims until a worker proof or an administrator resolution settles them, and stopped work items keep their holds until a person queues them.

## Remote workers (specified, not scheduled)

A remote worker runs on another host and claims the same assignments over the authenticated HTTPS worker API. The client already refuses cleartext outside loopback.

- **Ownership.** The service owns the project repository, every bundle it issues, the fetched candidates and every worktree used for review and merge. A worker owns only its disposable sandboxes and its receipt directory. Losing a worker host loses no authoritative state, only possibly an `unknown` attempt.
- **Assignment.** A remote assignment carries no host paths. It names the base bundle at `GET /v1/worker/attempts/{id}/bundle`, authorized for the assigned worker and dispatch. The worker uploads its candidate bundle to `PUT /v1/worker/attempts/{id}/candidate` with a size limit; the upload is idempotent by digest. The terminal report then names the candidate revision and the bundle's SHA-256. From there the service does what it does for a local bundle.
- **Receipts.** The current durable receipt contract is unchanged. The dispatch-derived container name adds a second guard against a repeated start.
- **Cancellation.** The heartbeat directive works as it does now, and stop proofs work as they do locally.
- **Reconciliation.** Only the owning worker identity can report a proven stop for its `unknown` attempt. The service never moves an attempt to another host. Lightspeed makes the same choice: a lost environment shows as unavailable and is never switched automatically.
- **Disconnection.** A remote worker that cannot heartbeat for longer than its configured disconnect limit stops its sandboxes with proof. It reports those stops when contact returns. The limit is a worker setting, separate from the 30-second service lease, so a short service restart does not kill builds. Until the worker reconnects, the halt cannot reach it, and its attempts stay `unknown` at lease expiry as they do today.
- **Secrets.** They stay on each worker host. Every remote host needs its own secret file.

## Out of scope

- Hosted sandbox providers (Daytona, E2B, Modal). The profile's `backend` field leaves room for them.
- Confining the Docker daemon itself. Anyone with access to the daemon can inspect containers, including the secret values passed to them. Waymark assumes the operator owns the daemon.
- Sandboxing the refiner and judge, and sandboxing arbitrary executor commands.
- Codex cost reporting. The harness planning work owns that. Until it lands, Codex cannot run in a budgeted project.
- Incremental bundles and image building beyond the reference Dockerfile.

## Interface

The board keeps two human decisions per change. Sandbox details (image, limits, egress log, stop proof) are attempt artifacts behind the Run view's technical detail. Three new states appear where a person must act or know: "budget reached" as a waiting decision on a card, "Stop not confirmed" on a run, and the halt banner.

## Order

1. [Halt all factory execution](../records/factory-isolation/issues/01-installation-halt.md), the smallest independent safety control.
2. [Run required checks in a Docker sandbox](../records/factory-isolation/issues/02-sandboxed-checks.md). This closes the verifier's path to host tokens.
3. [Limit sandbox egress to listed hosts and named secrets](../records/factory-isolation/issues/03-sandbox-egress-secrets.md).
4. [Build candidates in a Docker sandbox](../records/factory-isolation/issues/04-sandboxed-builds.md). This closes the builder's path.
5. [Confirm that a stopped sandboxed run ended](../records/factory-isolation/issues/05-proven-sandbox-stop.md).
6. [Settle an interrupted sandboxed run from its container](../records/factory-isolation/issues/06-settle-from-container.md).
7. [Write holdout expectations in a sandbox](../records/factory-isolation/issues/07-sandboxed-expectations.md).
8. [Require sandboxed execution in a project](../records/factory-isolation/issues/08-require-sandbox.md).
9. [Cap a job's agent spend](../records/factory-isolation/issues/09-job-budgets.md) and then [a project's monthly spend](../records/factory-isolation/issues/10-project-budgets.md). Budgets share no code with the sandbox chain, so they can go at any point.
10. When a second host is actually needed: [run sandboxed attempts on a remote worker](../records/factory-isolation/issues/11-remote-workers.md) and [stop a disconnected remote worker's sandboxes](../records/factory-isolation/issues/12-remote-disconnect-stop.md).

## Verification

Unit and service tests use a stub `docker` executable that records argv and environment and plays back runtime states: running, exited, unreachable, and still running after `SIGKILL`. This keeps the default `go test ./...` free of a daemon. Each sandbox ticket also adds an integration test against a real Docker-compatible runtime, run when `WAYMARK_DOCKER_TESTS=1` is set. It checks the confinement claims directly:

- no reachable host paths or tokens;
- no inherited environment;
- direct egress fails;
- only listed hosts pass the proxy;
- a container that has a detached descendant ends with proof.

Fixture agents stand in for model providers. Report any live-provider run separately from these checks. Keep generated evidence out of Git.
