# Deliver accepted work through the code host

A factory merge ends on a local branch. `jobs merge` and `jobs approve` build a `--no-ff` commit in a temporary worktree, pin it under `refs/waymark/merges/<job>`, run the required checks on it, then move a local ref (`internal/factory/merge.go`). Nothing pushes, opens a pull request, reads review comments, or takes in CI results from the host. A teammate cannot review the change where the team reviews code, and the project's own CI never counts.

This spec adds GitHub delivery to a project that asks for it. The local verification stays. The pull request becomes the place where host CI and teammates' reviews feed back before the person approves, and the person's approval merges it through GitHub. The [ADR draft](../../docs/adr/0006-deliver-accepted-work-through-the-code-host.md) records the trust decisions.

## Approved behavior

The user decided these on 25 October 2026:

- The pull request is an addition to the local verified merge, not a replacement. Waymark opens it before the person's approval, as soon as the candidate passes the local checks, so host CI and review arrive while the decision is still open.
- A "Request changes" review from a GitHub login mapped to a Waymark reviewer starts a repair within the admitted repair limit. It does so automatically under the autopilot, as a local change request already does.
- The service acts on GitHub through the operator's own `gh` login.
- GitHub is the first and only host for now. Waymark itself, Gups, and the next targets live there.

These follow from existing rules and were not separate questions:

- Only a host check that reports success on the exact head commit counts as passing. Skipped, neutral, cancelled, missing, and unreachable checks stay unverified.
- Host feedback can hold work back but never let it through. A GitHub approval never stands in for the Waymark review.
- A host check counts only when the files that define it are protected paths.

## Bind a project to a repository

A project administrator binds a registered project to one GitHub repository. The binding names the owner and repository, the Git remote that points at it, the default target branch, the merge method (`merge` or `squash`), and a reviewer map from GitHub login to Waymark username. Binding checks that `gh` is logged in and that the login can push to the repository. A project without a binding works exactly as today. Forks are out of scope. The pull request's head and base are in the same repository.

## Delivery flow

1. Admission, the build, and the local checks run as today.
2. When every required local check passes on the candidate, the service pushes the candidate to `refs/heads/waymark/<job>` and opens a draft pull request against the target branch. The title is the work item's title. The service writes the body: the acceptance criteria, each check's result, the requirements no locked test covers, the job ID, and a line asking reviewers to request changes instead of pushing to the branch.
3. The service watches the project's required host checks on the pull request's head commit. The job stays `verifying` until every required check, local and host, has passed on that commit.
4. When every check has passed, the service marks the pull request ready for review and the card moves to Approve. The pull request is ready for review exactly when the card waits for the person's approval. It goes back to draft when a repair starts.
5. A failed check or a trusted change request sends the job to repair. A repair builds on the previous candidate, so the new candidate is a descendant and the service fast-forwards the same branch. Local checks, then host checks, run again on the new head.
6. The person's Approve, with the merge switch on, merges the pull request through GitHub. See [Merge through the host](#merge-through-the-host).
7. When a job ends without merging, because it was cancelled, rejected, or superseded, the service closes its pull request with a one-line reason and keeps the branch.

The service pushes only to `refs/heads/waymark/*`. Every push is a compare-and-swap against the remote tip the service last pushed, and the first push expects the branch to be absent. It never force-pushes over a commit it did not push.

### The head must be Waymark's commit

The pull request's head must be a commit the service pushed for this job. Another head, from a teammate's push, GitHub's "Update branch" button, or a committed review suggestion, stops the service from counting host checks, recording host reviews, or merging. The card says the branch has commits Waymark did not build. Adopting such a commit as a candidate would skip the protected-path check and the implementation attribution, so it is a follow-up and not part of this plan.

## Host checks

An installation administrator registers a host check profile: its key, the GitHub app that must report it (for example `github-actions`), the check run name or commit status context, the paths that define it, and how long to wait for it (30 minutes by default). A project requires it by key in its required checks, like any other check, and admission freezes it.

Project policy refuses a host check unless the project's protected paths cover its definition paths. For GitHub Actions these are `.github/workflows/**` and any action or script those workflows run from the repository. GitHub runs `pull_request` workflows from the pull request's own version of those files. An unprotected workflow therefore lets the implementer edit the check that grades it.

The service observes host checks itself, under a dedicated worker-kind observer identity, and polls every 30 seconds. Webhooks are out of scope because the service binds to loopback by default. Each observation is a verification attempt for one check key on the exact head commit. It records the check result with the head commit as its candidate revision, the requirement fingerprint, the observer, the check run ID, the reporting app, the conclusion, the details URL, the completion time, and an artifact holding the host's JSON and its digest. Acceptance publishes this evidence with the rest, so its origin and revision travel with the Acceptance record.

| GitHub reports | Waymark records |
| --- | --- |
| check run `success`, commit status `success` | `passed` |
| check run `failure` or `timed_out`, commit status `failure` | `failed` |
| check run `neutral` or `skipped` | `skipped` |
| check run `cancelled`, `stale`, or `action_required`, commit status `error` | `unavailable` |
| queued, in progress, waiting, or pending | still pending; keep observing |
| no matching check for the head commit when the wait expires, or the host unreachable until then | `unavailable` |
| a matching name from a different app | ignored, and noted on the attempt |

Waymark repairs code only after a definite failure. `timed_out` counts as failed because code can cause it. A commit status `error` usually means the reporter broke, not the code, so it does not start a repair.

Only `passed` satisfies a required check. `failed` sends the job to `verification_failed`, and the repair brief names the check, its summary, and its details URL. `skipped` and `unavailable` leave the job not verified. The autopilot does not repair it, and the card says "Not verified" with the check and what GitHub reported. The person can rerun the check on GitHub and ask Waymark to check again, ask for a fix, or cancel. Checking again starts a new observation, and the latest one decides, as for local checks.

This rule applies to every required check, local or host. Today a local check that reports `skipped` or `unavailable` counts as a failure, and the autopilot repairs code that did not fail.

SuperPlane's evaluator counts neutral, skipped, and cancelled conclusions as nonfailing. SWE-AF's CI gate counts a pull request with no reported checks as passing. Waymark adopts neither. It does keep two of their protections: SuperPlane keys each check by app and name, and SWE-AF refuses to judge until it has seen checks for the expected head commit.

## Review feedback

A mapped reviewer is a GitHub login in the binding's reviewer map whose Waymark user is human, holds `review` on the project, and passes the existing independence rule.

A `CHANGES_REQUESTED` review from a mapped reviewer, whose commit is the current head, becomes one Waymark `changes_requested` review. It is attributed to that user, carries the review's URL as its source, and is recorded once per GitHub review ID. If the head's checks are still running, the review waits and applies when they conclude. The job then goes to `verification_failed` and repairs within the admitted limit, automatically under the autopilot. Several change requests on one head produce one repair. When the limit is used up, the card asks for the person as it does today.

The repair brief quotes the review's body and its inline comments with their paths and lines. A size cap applies, and the brief marks truncation. It labels the text as a reviewer's request that cannot change the rules, the frozen requirements, or the protected paths. A request outside the requirements leads the agent to the existing dispute path.

The service imports nothing else as a decision. Approvals, plain comments, `COMMENTED` reviews, reviews on older heads, and feedback from unmapped logins or bots appear as a count on the card and as links in the Approve view. The person can act on them through Ask for a fix. Waymark never resolves review threads, replies on anyone's behalf, or dismisses reviews; the reviewer owns their thread. SWE-AF resolves threads its own agent claims to have addressed, and SuperPlane runs its agent on any comment that mentions it. Both let unreviewed text drive work, and Waymark avoids both.

GitHub does not let a pull request's author approve or request changes. With the operator's `gh` login the operator authors every factory pull request, so only teammates can request changes there. The operator gives feedback in Waymark through Ask for a fix.

## Repairs say why

A repair attempt currently tells the agent nothing about why it is repairing. `assignmentFor` in `internal/factory/workers.go` adds only a judge directive. The failed check and the reviewer's summary never reach the prompt, so the agent repairs blind. Host feedback needs that path, and local repairs need it too.

Every repair assignment carries a repair brief. The brief lists the previous candidate's failed required checks, each with its status and a bounded excerpt of its retained output or host summary, and every change request on that candidate, each with its reviewer, summary, findings, and quoted comments. The brief is attempt data. It is not a requirement and does not change the admission fingerprints.

## Merge through the host

Approve with the merge switch on, or `jobs merge`, for a bound project:

1. Rechecks as today: the acceptance is valid, and the request matches the candidate and job version.
2. Reads the pull request. It must be open, and its head must be a commit the service pushed for this job.
3. Fetches the target branch's tip.
4. If the head contains that tip, asks GitHub to merge the pull request with the binding's method and the expected head commit. GitHub refuses the merge if the head moved.
5. If the target moved and the head no longer contains its tip, builds the merge commit on the fetched tip exactly as today and runs the local merge checks on it. It then fast-forwards the pull request branch to that commit, waits for the host checks on it, and asks GitHub to merge with that commit as the expected head. Merge checks never change the job's own status, as today.
6. Records the merge with mode `pull_request`, the pull request URL, and GitHub's merge commit. It checks that the merge commit's tree equals the verified head's tree and that its first parent is the fetched tip. If the target moved again in between, the trees differ. The record says so, the merge does not count toward the trusted merges that unlock merging without asking, and the card shows it.
7. Fetches the merge commit and fast-forwards the local target branch under the existing checkout and ref rules when the local tip is the fetched tip. Otherwise it leaves the local branch alone and the card says to update it. Builds start from the local `HEAD`, and admission refuses dependent work whose base lacks a merged prerequisite.

A refusal from GitHub, such as a branch protection that needs an approval, a check GitHub itself requires, or a conflict, fails the merge with GitHub's message and leaves every branch as it was. The existing retry rules apply. A pull request merged on GitHub before Waymark's acceptance is recorded as observed. The job stays unaccepted, and the card says it merged before approval. A person can still approve it, and the merge then records as contained. A pull request closed on GitHub asks the person to reopen it or cancel the job.

The autopilot's approvals under the ask policy merge through the same path.

## Credentials and the two decisions

The service uses the operator's `gh` login. API calls go through `gh api`. Pushes run `git -c credential.helper= -c 'credential.helper=!gh auth git-credential' push`, so the token stays inside `gh`. The service never reads the token into its own memory or environment. It never writes the token into assignments, events, artifacts, receipts, or logs. The worker and solo processes remove `GH_TOKEN`, `GITHUB_TOKEN`, and `GH_ENTERPRISE_TOKEN` from the environment of every agent and check process they start.

This does not confine the login. An agent or a candidate's own test code runs as the operator and can use `gh` if its tools reach a shell. The shipped `claude-agent` profile allows only `go test` and `go vet`, but verification runs whatever the candidate's tests do. That exposure already exists for every credential the operator holds, and closing it belongs to the worker environment. Within Waymark, the head rule keeps a commit the service did not push from being merged.

Everything the factory does on GitHub appears as the operator. Branch protection that requires an approval therefore needs a teammate, because the author's own approval does not count. A team pilot should revisit this with a GitHub App; the binding names its credential source in one field, so adding an App does not change the flow.

The person still makes two decisions per change. Binding a project, choosing its host checks, and mapping reviewers are project configuration, like the ask policy. Pushing, opening, marking, and closing pull requests, observing host checks, importing trusted change requests, and repairing are steps that need no decision. A teammate's change request is that teammate's decision and can only hold work back. The person's approval merges through GitHub, and that click remains the only route to a merge.

## Recovery

The service records its intent before each external effect: the push (ref, expected remote tip, new commit), the pull request creation (head branch and base), each draft or ready change, each close, and the merge (pull request and expected head). After a restart or an uncertain reply, it reconciles from GitHub's state. It reads the remote ref, finds the pull request by its head branch, and reads the merge state and merge commit. It never repeats an effect blindly. While GitHub cannot be read, the step stays pending and visible. This follows the [controller decision](../../docs/adr/0005-factory-controller-owns-operational-state.md): a transaction does not establish an external effect.

## Board

Cards gain short facts: "PR #12", "GitHub checks 2 of 3", "Changes requested by alice", "Not verified: lint skipped on GitHub", "3 other comments". Commit IDs, app names, and raw host replies stay in Run and History. The two human decisions do not change.

## Out of scope

Each of these is a possible follow-up, to take up when use shows the need:

- A GitHub App credential, for a bot author and narrower scope.
- Webhooks instead of polling.
- Fork pull requests and other hosts.
- Adopting a commit someone else pushed to a `waymark/*` branch as a candidate.
- Waiting for a host approval that branch protection needs, instead of failing the merge.
- Checking GitHub's merge commit when its tree differs from the verified one.
- The `rebase` merge method.
- Selecting other pull request comments into Ask for a fix.
- Pushing a local verified merge straight to an unprotected branch without a pull request.

## Verification

Exercise every ticket offline with a fake `gh` that records its calls and replies from scripted JSON, and a bare repository as the remote. Cover each host conclusion, a check from the wrong app, a missing check, results for an older head, a foreign head, interrupted pushes and merges, a refused merge, and a target that moves. Verify the real browser views. Keep generated trial data and acceptance evidence out of Git.

A live run against a disposable GitHub repository that the user names is reported separately from the fixture checks. It publishes to GitHub, so it needs the user's go-ahead at the time.
