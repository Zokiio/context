# Run the factory for a small team

The vision names a possible pilot: a small team working in one monorepo inside an enterprise. This spec says what the factory service needs before three to six people can share it, and which extra interfaces come first. It was planned on 2026-10-25 against `dev` at `afe9989`. Joakim settled the open decisions recorded below in the same session. Nothing here is a current commitment until he selects the tickets.

## The pilot this plan serves

- Three to six people work in one monorepo, possibly across several registered projects.
- One factory service runs on an always-on host the team can reach over TLS. Workers run on that host.
- Teammates use the browser and the `ctx factory` CLI from their own machines.
- A person agrees what done means for their own change, and a different teammate approves it. The two decisions per change stay as they are.

Remote workers, parallel builds and pushing to the team's remote belong to sibling plans. See [Dependencies](#dependencies-on-sibling-plans).

## What breaks for a team today

These were checked in the code. None of them shows up in solo use.

1. "Needs you" means "needs a person". `deriveCard` works out each card per project, not per viewer (`internal/factory/board.go`). Bob sees his own Approve card under Needs you, but the reviewer rule refuses his approval (`independentReviewer`, `internal/factory/accept.go:359`). Nobody can tell whose turn it is.
2. A link opened in a new tab asks for the token again. The browser keeps the bearer token in per-tab `sessionStorage` (`internal/factoryhttp/assets/app.js:7`). Every link from a chat message would open a sign-in screen.
3. Nothing tells a person that work is waiting on them. Events exist only per job, and nothing leaves the service.
4. One person's ask policy approves everyone's changes. The autopilot acts as `Ask.SetBy` for every job in the project (`internal/factory/pilot.go`). If that person saved "ask only when untested", the autopilot approves a teammate's change in their name, and nobody read the diff.
5. The service's record writes never reach the team. Task edits, accepted refinements and acceptance publication land as uncommitted files in the host's checkout, and nothing in `internal/factory` or `internal/factoryrecords` commits, fetches or pushes them. The plan for pushes, PRs and CI owns the fix.

The store is not on this list. One SQLite connection behind one process (`SetMaxOpenConns(1)`, `controller.lock`) is plenty for a few people and a handful of workers. The real risk is losing the data directory, and a backup covers that.

## Decisions

### What comes first

In this order:

1. Sign in once per browser ([01](../records/factory-team-scale/issues/01-browser-session.md)). Links from a notification need it.
2. Each card names who its decision waits on, and "Needs you" means the viewer ([02](../records/factory-team-scale/issues/02-decision-owners.md)).
3. A project webhook announces work that waits on a person ([03](../records/factory-team-scale/issues/03-notification-webhook.md)), formatted for the team's chat once that is known ([04](../records/factory-team-scale/issues/04-chat-formats.md)).
4. A guide for running the factory on a shared host, with backup and restore, rehearsed end to end ([06](../records/factory-team-scale/issues/06-team-host-guide.md)).

The read-only MCP server ([05](../records/factory-team-scale/issues/05-read-only-mcp.md)) is independent and can land at any point. It isn't a pilot entry condition.

### Keep one service and SQLite

ADR 0005 stands, so this plan needs no ADR. One controller with a private SQLite store already serves several users and clients. It is also the shared claim authority the vision asks for: teammates on other machines submit jobs to the same controller, and the unique active-claim index (`one_active_implementation_claim`) does the rest. The separate claim service proposed under Team mode in `docs/software-factory-plan.md` isn't needed.

The pilot needs the data directory to survive a disk loss. An online SQLite backup plus a copy of the immutable evidence directory covers that with no new command. Restoring that backup behaves like a crash: on start, `recoverInterruptedAttempts` marks attempts that were running as unknown, and they need reconciliation. Ticket 06 documents and rehearses this.

### Sign in with a token, then a session

Joakim accepted admin-issued tokens for the pilot organisation; enterprise SSO isn't required. A person pastes their token once per browser. The service exchanges it for a server-side session with a fixed seven-day lifetime, held in an HttpOnly, SameSite=Strict cookie (Secure on TLS). Revoking the token, disabling the user or signing out ends the session. CLI, worker and typed-client authentication stay bearer-only.

### Who a decision waits on

Each waiting card names who can make its decision. Everything is derived from stored records and grants, with no new state:

- Plan and fix decisions wait on the work item's owner. The owner is the current job's coordinator. Without a job, it is the person who requested the current refinement or locked tests, or who gave the go-ahead. Before any of those, the card waits on anyone who can coordinate.
- Approval waits on reviewers. A reviewer is an enabled human with review and accept on the project who passes `independentReviewer` for that job.
- Merge decisions wait on the approver if they hold merge, otherwise on anyone who holds merge.

Approvals go to the pool of reviewers, not to a named teammate. Named review requests wait until approvals sit unclaimed because everyone assumes someone else will take them.

### Notify on work that waits on a person

The team's chat isn't chosen yet; Slack and Discord are both possible. The service therefore sends a versioned JSON payload to one HTTPS webhook per project (ticket 03). Formatting for the chosen chat is a small follow-up (ticket 04, `needs-info`). Email, direct messages and per-person preferences are out of scope.

One message goes out when a card starts waiting on a person:

- a plan decision for the owner;
- an approval for the reviewers;
- a merge that is ready or failed;
- a fix decision: checks failed after the last allowed fix, a failed build, changed requirements, an unknown outcome after a lost lease or restart, an unknown stop, a dispute that needs a person, or a step the autopilot could not take.

A stalled run means a decision a person has to make, and every case above already produces one, except one. A build that no builder has started 15 minutes after admission also sends a message, because a dead worker process on a shared host otherwise sits silently at "Waiting for a builder".

Failed holdouts are not a separate trigger. While the factory is fixing them, nobody needs to act. Once the last allowed fix fails, the "checks failed" message names the failing checks.

Messages name the project, work item title, decision, the people it waits on and a link to the task page. They carry no requirement text, diffs, worker output or check output, so a chat channel never becomes a copy of the record or a route for agent-authored text.

### A teammate approves every change in the pilot

Joakim chose this. The pilot project's ask policy stays at "always" before and after the build. The autopilot stays on, so go-aheads start builds and bounded repairs run. It never approves on anyone's behalf. Per-person delegation is planned only if the pilot shows that approval is the bottleneck.

A known wart remains, recorded here rather than fixed. Repairs and merge retries that the autopilot runs under the project policy are attributed to the person who saved that policy, even on a teammate's job. Fix it only if the pilot's history or metrics show the confusion.

### A read-only MCP server

Joakim added this over the recommendation to defer it. `ctx factory mcp` runs an MCP server over stdio and calls the service through `factoryclient` with the user's own token. It holds no database, no data directory and no workspace. Its tools read projects, orientation, the board, a work item's run and activity, and a job's attempts, change and events.

It has no claim, report, queue, plan, approve, merge or administration tool. Claims belong to jobs, reports belong to workers, and decisions belong to people. A tool that lets an implementing agent report its own result would put the judged in charge of the judge. Task context for an unadmitted ticket stays with the local `ctx context` reader.

Stdio over the existing client keeps the HTTP security model unchanged: no new listener, no CORS, no second credential path.

### Not in this plan

Each item below waits for a need the pilot shows:

- Postgres, several active controllers, high availability, a message broker or a workflow engine such as Temporal. Revisit if service restarts or write latency block the team, or if one host can't hold the installation.
- Custom roles and groups (SuperPlane keeps these in its separately licensed `ee/` directory). Per-project capability grants cover three to six people. Revisit if a wrong grant through the raw capability form causes a problem in the pilot.
- A TUI. CodeAF has one, but the web board and CLI cover every pilot use.
- Enterprise SSO, email, direct messages and per-person notification preferences.
- MCP write tools, as above.
- Workers on other hosts, sandboxes and spending caps. The sandboxes plan owns these.

## Dependencies on sibling plans

- **Pushes, PRs and CI.** The pilot can't start until merges and the service's own record writes reach the team's remote, and the service fetches the base it builds from (finding 5). On protected-main repositories this interacts with the PR flow, so that plan owns it.
- **Parallel runs.** The autopilot builds one change per project at a time (`pilotBusy`). For three to six people that limits throughput but doesn't block the pilot.
- **Sandboxes.** Until isolation lands, candidate code runs as the worker's OS user. Ticket 06's guide runs workers as a different OS user from the service and keeps no human token files on the host. This follows the sandboxes plan's finding that candidate code can read a same-user token file.

## Pilot entry conditions

- Tickets 01, 02, 03 and 06 are accepted, and 04 is accepted for the team's chat.
- The PR plan's push, fetch and record-commit work is accepted.
- The pilot project uses the ask policy above, and every teammate holds read, coordinate, review, accept and merge.

## Verification approach

Each ticket carries its own service, HTTP and browser checks with fixture workers. Ticket 06's rehearsal joins them with three users against a TLS service. Live-provider runs and a real chat channel are reported separately from fixture checks. Generated trial data and acceptance evidence stay out of Git.

## Sources

- [Product vision](../../docs/vision.md): the team pilot and cross-machine claims.
- [Controller decision](../../docs/adr/0005-factory-controller-owns-operational-state.md)
- [Factory architecture](../../docs/factory-platform-design.md) and [HTTP access](../../docs/factory-http.md)
- [Software factory research](../../docs/research/software-factory-2026-10-22.md) with its pinned sources. The research fixes SuperPlane at `1be6b59d` (Postgres, RabbitMQ, roles and groups under `ee/`), CodeAF at `c740a541` (TUI and headless) and Lightspeed at `86f6adf3` (Temporal).
- [Software factory proposal](../../docs/software-factory-plan.md), not adopted: Team mode, interfaces and Phase 7.
