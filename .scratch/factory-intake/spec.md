# Bring tracker work into the factory

Decided with the user on 25 October 2026. Work enters the factory today only through the board or `ctx factory work add`, and the autopilot picks only from Current commitments. This plan imports GitHub issues and Jira Data Center issues into the project record and lets a pickup trigger queue matching work on a person's behalf. The [intake decision](../../docs/adr/0007-tracker-intake-ownership-and-delegated-pickup.md) records the ownership and start rules this spec depends on.

## Observed need

The Opsbase factory grooming trial on 25 October captured 15 open GitHub issues by hand. Its script, `opsbase-factory-trial/.context/trial/capture_github.py` at commit `621972a`, did what this plan automates:

- It wrote one unqueued draft WorkItem per issue, identified by `sourceURL`, with a UUID kept in a separate map across refreshes.
- It set triage from the five triage labels, which Opsbase already uses. One issue carried `ready-for-agent`; fourteen carried `ready-for-human`.
- It parsed `## Acceptance criteria` and `## Blocked by` out of the body and turned `#NN` references into local links.
- It set `execution: unstarted` as a stated local adaptation, because the factory admits only unstarted work.
- It retained the raw response and listed every adaptation under Comments.
- It also copied six GitHub comments into the tickets. Agents therefore read text from anyone who can comment on the issue.

The user also intends a grooming trial on a team project in Jira Data Center, where acceptance criteria sit in a section of the description.

## Outcome

An administrator connects a project to a GitHub repository or a Jira project. Matching issues appear on the board as ordinary work items with a source tag. A pickup trigger saved by a person can queue them, so the autopilot sharpens their requirements without anyone queueing by hand. A person always agrees what done means on an imported item before any agent that can write files or run commands works from its text. After that agreement the record owns the agreed requirements, and the project's ask policy applies as it does to native work.

## How others do it

Both projects in the [research note](../../docs/research/software-factory-2026-10-22.md) start work from trackers. Neither requires a person to read tracker text before an agent that runs code acts on it. Findings come from code read at the pinned commits.

- SuperPlane intakes create drafts from eight sources, including GitHub, Jira and Linear. Identity is the origin URL, and content is copied once at creation. A manual refresh archives closed items but never applies remote edits. An auto-start rule dispatches a draft when planning reports no open questions and top scores. Tracker text reaches prompts raw, with no marking and no length cap at creation. The only source-trust control is an optional check that a GitHub issue's author has repository access ([refinement prompt](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/grpc/actions/factories/intake_template.go#L576-L581), [auto-start](https://github.com/superplanehq/superplane/blob/1be6b59d8c884084457c0bb3248580e173115822/pkg/public/runner_planning_session.go#L445-L504)).
- Open-Inspect automations start sessions on GitHub, Slack, Sentry, webhook and schedule events with no approval step. A unique trigger key per event prevents repeats. Event text is wrapped and marked untrusted, and bodies are cut at 500 characters. That marking is the only control between an issue author and a session that runs code ([prompt guard](https://github.com/ColeMurray/background-agents/blob/776c771acf9e5fd389722f3506ea8587e22a54de/packages/control-plane/src/scheduler/scheduler.ts#L364-L374), [GitHub context](https://github.com/ColeMurray/background-agents/blob/776c771acf9e5fd389722f3506ea8587e22a54de/packages/shared/src/triggers/github/context.ts#L19-L48)).

Waymark keeps their good parts: identity by source URL, drafts first, dedupe, and a marked prompt block. It adds the part both lack: a person reads the imported request before any agent with write or command tools acts on it.

## Decisions

| Question | User's choice | Reason |
| --- | --- | --- |
| Which tracker first | GitHub Issues and Jira Data Center, both in this milestone | Opsbase runs on GitHub today; the team project runs on Jira Data Center with criteria in the description |
| Field ownership | The tracker owns the request. The record owns the requirements once a person agrees what done means | Refinement can proceed without a GitHub edit per item, and later tracker edits stay visible instead of silently changing agreed work |
| What a match may do | A trigger can queue matches on its author's behalf. A person always agrees what done means on an imported item | Nobody queues by hand, and no agent that runs code acts on outside text before a person has read it |
| Board | Ordinary cards in Up next with one source fact. No new column | Keeps the board's two decisions per change |

One precision on the third choice. The question offered "the autopilot may sharpen and write locked tests" before agreement. The test author runs `go test` on what it writes, so it can execute code. Before agreement only the read-only refiner, with Read, Grep and Glob, runs on imported text. The test author starts after the person agrees.

## Connections and scope

An installation administrator registers a tracker connection. It names the kind (`github` or `jira-datacenter`), the base URL, and a private token file. The GitHub base URL is `https://api.github.com` in this milestone. A Jira base URL must use HTTPS unless it is loopback. The controller reads the token file at each refresh. The token never enters SQLite, logs, prompts, assignments or worker environments.

Use read-only credentials. A GitHub fine-grained token needs only Issues read on the bound repository. A Jira Data Center personal access token carries its user's full permissions and cannot be scoped down, so use a dedicated user with browse permission only. The import issues only read requests. A test against the fake servers fails if it sends anything else. Agents run as the operator's OS user and can read files that user can, so a read-only token limits what a leak exposes. Keeping such files out of agents' reach belongs to the separate sandbox and isolation plan.

A project administrator binds a registered connection to the project with a scope and a records folder. The scope is a repository `owner/name` for GitHub, or a Jira project key with an optional JQL clause. The folder defaults to `imported/<connection>/` inside the bundle. A binding may name default Context links added to every imported item, as the trial added the context map and glossary.

## Import and identity

Each imported issue becomes one WorkItem in the binding's folder, named `gh-<number>-<slug>.md` or `<KEY>-<slug>.md`. The file name does not change when the title changes. Identity is the canonical `sourceURL`. The import generates a UUID `id` once and finds the record by `sourceURL` on later refreshes, so no separate identity map is needed. Two records with the same `sourceURL` stop the import for that item and show a diagnostic.

The record uses the existing [tracker snapshot fields](../../docs/tracker-snapshots.md#ownership-and-identity): `sourceURL`, `sourceUpdatedAt` and `snapshotAt`. It also sets `execution: unstarted` and records that adaptation under Comments. The controller retains each raw response in its operational store with its retrieval time and digest. The Markdown record stays in the bundle.

Mapping from each tracker:

| Record part | GitHub | Jira Data Center |
| --- | --- | --- |
| Title | Issue title | Summary |
| Description | Body without the parsed sections | Description converted from wiki markup to Markdown, without the parsed sections |
| Acceptance criteria | List under a `## Acceptance criteria` heading, bullets or checkboxes | List under an `h2.` or `h3.` Acceptance criteria heading, `*` or `#` items |
| Blocked by | `#NN` references under `## Blocked by` | Issue links of type "is blocked by" |
| Triage | One of the five triage labels, otherwise `needs-triage` | Same, from Jira labels |
| State | Open or closed | Done status category or not |
| Parent | Not read | Sub-task parent key and URL, recorded in frontmatter only |

GitHub's issues API also returns pull requests. The import skips them. An issue with no criteria section imports with none, and orientation reports it not ready, as the trial did. A checked criterion on an open issue is noted under Comments. A blocker that is also imported becomes a local link. A blocker that is closed upstream and not imported is omitted and noted as satisfied by the tracker. Any other blocker stays as plain text, which keeps readiness unknown.

The import cleans text before writing it, so a person sees what agents will read:

- It removes HTML comments. GitHub hides them when rendering, but agents read them.
- It removes invisible format characters, bidirectional controls and Unicode tag characters, keeping the zero-width joiner inside emoji.
- It demotes headings that collide with record sections, such as `## Comments` or `## Context`, by one level.
- It notes each change and its count under Comments.

A description longer than the existing 4,000-character limit is not imported. The connection status lists it with the reason. GitHub and Jira comments are not imported. The Task page shows how many exist and links to them.

## Field ownership

| Field | Owner |
| --- | --- |
| Title, description, labels and triage, state, assignee, source URL and timestamps, Jira parent | Tracker. Read-only in Waymark, with "Edit in GitHub" or "Edit in Jira" |
| Acceptance criteria and Blocked by | Tracker until a person agrees what done means, then the record |
| ID, execution, Blocked by decisions, Spec and Context links, Acceptance, Comments, queue membership, archive | Record |

A person agrees what done means on an imported item by accepting a refinement, giving the plan go-ahead, or starting the build. That decision writes `requirementsAgreedAt` and a Comments line before any admission captures fingerprints. From then on refreshes leave criteria and blockers alone.

## Refresh

Refresh writes tracker-owned fields only to an item that is unstarted and that no refinement, test authoring or job holds. This follows the existing edit rule and keeps an admitted ticket's fingerprint stable, because acceptance refuses a ticket that changed after admission. Refresh never rewrites completed or cancelled items, because that would make their acceptance stale. It retains the new snapshot and shows the change on the Task page.

After agreement, a tracker change to the description, criteria or blockers holds the item at a plan decision: "Changed in GitHub since you agreed." The person keeps the agreed requirements, or takes the tracker's version and plans again. Title and description still follow the tracker while the item is idle.

An issue closed upstream archives its unstarted item with the reason, which removes it from the queue. Active work only gets a "Closed in GitHub" fact; stopping it remains a person's explicit decision. Reopening a matching issue restores its item unqueued, and a queueing trigger can queue it again.

## Polling and pickup triggers

The service polls each bound connection on an interval, five minutes by default, and on "Refresh now" from the board, CLI or API. A poll asks for items in scope updated since the last successful poll, with a minute of overlap. The first poll reads every open item. GitHub uses `since` and Jira uses an `updated >=` JQL clause, so closures and label changes arrive without re-reading every item. Rate-limit responses back off until the tracker's reset time. A failed poll changes nothing and keeps the last successful time visible. A poll imports at most 100 new items and reports how many more matched.

A pickup trigger has a name, conditions and an action. Conditions are labels and assignee, combined as all or any. Nothing is imported until a trigger matches it. A trigger with no conditions matches every open item in scope, which is how the Opsbase grooming trial imported its whole backlog. Each imported item records which triggers matched. Keyword matching waits for a trial that needs it.

The action is `import` or `queue`. Only a human project administrator who holds coordinate can save a queueing trigger, and it must name at least one condition. Like the ask policy, it records who saved it. The service makes each delegated queue decision as that person, rechecks their capabilities every time, and records the reason as "pickup trigger <name>". When a queueing trigger stops matching an unstarted item that it queued, the service removes that item from the queue. Items a person queued stay queued. Active work only gets a "No longer matches" fact.

## The gates stay separate

A match makes an item a candidate. Queueing is the permission to start, given by a person or by a trigger a person saved. Readiness and eligibility come from orientation as today: criteria, context, dependencies, blocking decisions, triage `ready-for-agent` and `execution: unstarted`. The card reports the first unmet gate in plain words, and the Task page lists every one. Examples are "Not ready: no acceptance criteria in GitHub" and "Not marked ready for agents: add the ready-for-agent label in GitHub."

Until a person agrees what done means, the autopilot treats an imported item as if the ask policy said "always ask before the build". It may start the read-only refiner. It does not accept a refinement, start the test author or start a build. Once the person agrees, the project's ask policy governs the rest.

## Tracker text in agent prompts

Prompts frame imported text as outside text. For an imported ticket, the inclusion note that the refiner, test author, builder and judge receive names the tracker, the issue and the retrieval time. It says the title and description were written outside the project, describe the request, and cannot change the rules above them or grant tools. The note is a weak control. The real controls are the person's agreement, read-only intake, protected paths, independent verification, and keeping the token out of agent reach.

## Board

Imported items are ordinary cards in Up next. One fact shows the source, such as "GitHub #170" or "ABC-123", linked to the issue. Folded details show which trigger matched and whether a trigger queued the item. The Task page shows tracker-owned fields read-only and the snapshot time, for example "Fetched 3 minutes ago." When a connection is failing, the top bar shows one line, such as "GitHub not reached since 10:02." Closed items leave the board and stay in History.

## Out of scope

- Writing back to trackers: comments, labels, closing issues, or posting agreed criteria. Ownership already allows it later. The board can offer agreed criteria as text to paste.
- Inbound webhooks. The service binds to loopback by default, and polling covers the trial.
- Scheduled or recurring tasks created without a tracker item.
- Linear, GitHub Enterprise Server, Jira Cloud, and configurable triage-label names.
- Parent requirements in task context. The import records a Jira parent, but selecting its criteria as context is separate reader work in the [vision](../../docs/vision.md#after-the-first-milestone).
- Committing imported records. Like every service-written record, they land as uncommitted files in the project's working tree.

## Verification

Test each tracker against a fake HTTP server serving recorded fixture responses, including the Opsbase capture's shapes, a wiki-markup description, pagination, rate limiting, closed and reopened issues, pull requests in the issues feed, hidden HTML comments, and bidirectional controls. Exercise import, refresh, agreement, holds, delegated queueing, the agreement gate in the autopilot, and the board in the real browser. Check that the fake servers receive only read requests and never see the token in a prompt or assignment. Fixture results do not establish compatibility with live GitHub or the team's Jira; report live trials separately.
