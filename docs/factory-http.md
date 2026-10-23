# Factory HTTP and browser access

The HTTP handler and embedded browser use the same `factory.Service` as command-line clients. `factoryhttp.NewHandler(service)` receives an already-open controller. It does not create another database, workspace owner, or worker loop. The service composition owns its listener, TLS configuration, data directory, and shutdown.

## Authentication and trust

Every `/v1/` request requires `Authorization: Bearer <token>`. The handler calls `Service.Authenticate` for each request and derives the acting user from the returned database identity. Request bodies cannot supply an actor. The core checks project visibility and capabilities again for every operation.

The browser stores its token in `sessionStorage` for the current tab. Sign out clears that storage and the in-memory credential. Browser session restore may retain the token. This first release does not provide an HttpOnly session cookie or enterprise SSO. Scripts served by this installation can access its browser token, so installation operators must control the static assets and use HTTPS for shared network access.

The API accepts no query-string credentials and provides no CORS access. Browser requests with another Origin or a cross-site fetch context fail. Responses use a restrictive Content Security Policy, `nosniff`, no referrer, and `Cache-Control: no-store`. The browser inserts project content, context, errors, worker output, and evidence through text nodes. It does not execute Markdown HTML.

User and executable-profile administration requires installation administration. Project grants govern coordination, verification, review, acceptance, and project administration. `GET /v1/projects/{id}/capabilities` returns only the caller's own capabilities. Hiding a browser action is a convenience; the service remains the permission authority.

## JSON operations

Requests and responses use the JSON-tagged domain structs in `internal/factory/types.go`. Bodies are limited to 2 MiB, must use `application/json`, and must contain one object with known fields. Mutations carry `requestId`; job mutations also carry `ifVersion`. A body subject must match the URL subject when both are supplied.

| Path | Methods and domain operation |
| --- | --- |
| `/v1/me` | GET authenticated principal |
| `/v1/users` | GET users, POST `CreateUserRequest` |
| `/v1/users/{id}/disabled` | PUT `SetUserDisabledRequest` |
| `/v1/users/{id}/tokens` | POST `IssueTokenRequest`; plaintext appears only in the original response |
| `/v1/tokens/{id}/revoke` | POST `RevokeTokenRequest` |
| `/v1/projects` | GET accessible projects, POST `RegisterProjectRequest` |
| `/v1/projects/{id}/orientation` | GET current reader result, project policy, and advisory `criterionSignals` when a decision model is configured |
| `/v1/projects/{id}/metrics` | GET measured outcomes, lead times, human and delegated decisions, findings, and agent cost |
| `/v1/projects/{id}/board` | GET one card per open work item: its column, state, the decision it waits for, its cost so far, short facts, its step among six, and the work items it depends on |
| `/v1/projects/{id}/work-items` | GET all task records, including drafts and archives; POST `CreateWorkItemRequest`, with `queue: false` to save an unqueued draft (omitting it preserves the original queued behavior) |
| `/v1/projects/{id}/work-items/reconcile` | POST `{}` to resume retained task saves under coordinator permission; only the exact retained before/after file contents are accepted |
| `/v1/projects/{id}/work-items/{item}/run` | GET the item's run as eight steps with status, summary, model, turns, time, cost, and who decided |
| `/v1/projects/{id}/work-items/{item}` | GET the editable `WorkItemRecord`, PUT `UpdateWorkItemRequest` with its `ifTicketSHA256`; refused during active or uncertain refinement, test authoring, or execution |
| `/v1/projects/{id}/work-items/{item}/disposition` | POST `SetWorkItemDispositionRequest` with `ifTicketSHA256` and `action`: `queue`, `unqueue`, `archive`, `restore`, or restricted `delete` |
| `/v1/projects/{id}/work-items/{item}/activity` | GET retained jobs, attempts, checks, decisions, proposals, and participant names independently of the board; includes cancelled jobs and artifact metadata |
| `/v1/projects/{id}/criteria/score` | POST a title and requirements; returns the decision model's advisory signals for a draft |
| `/v1/projects/{id}/plan` | POST `ApprovePlanRequest`: answer, accept a refinement or locked tests, and start the next step; `continue` gives a go-ahead to build |
| `/v1/projects/{id}/ask` | PUT `SetAskPolicyRequest`; requires a human project administrator who holds the delegated capabilities |
| `/v1/projects/{id}/instructions` | PUT `SetAgentInstructionsRequest`, replacing the refiner, test author and builder instructions; requires a human project administrator |
| `/v1/projects/{id}/expectations` | GET a project's holdout expectations, optionally `?workItemId=`, POST `StartExpectationsRequest` |
| `/v1/expectations/{id}` | GET one suite with its files, patch, and notes |
| `/v1/expectations/{id}/accept`, `/v1/expectations/{id}/reject` | POST `DecideExpectationsRequest`; accepting requires a human |
| `/v1/projects/{id}/capabilities` | GET caller's capabilities |
| `/v1/projects/{id}/grants` | GET project grants for an authorized administrator |
| `/v1/projects/{id}/grants/{user}` | PUT `GrantProjectRequest` |
| `/v1/executors`, `/v1/checks` | GET profiles, POST the matching registration request |
| `/v1/jobs` | GET accessible jobs, POST `SubmitRequest` |
| `/v1/jobs/{id}` | GET retained job, admission, attempts, checks, reviews, and events |
| `/v1/jobs/{id}/events?after=0` | GET events after a nonnegative sequence |
| `/v1/jobs/{id}/artifacts/{artifact}` | GET an artifact attached to that job's implementation or verification attempt |
| `/v1/jobs/{id}/actions` | POST `JobActionRequest` for cancel, repair, reconcile, or explicit resolution |
| `/v1/jobs/{id}/reviews` | POST `ReviewRequest` for an exact candidate and requirement fingerprint, with optional `findings` |
| `/v1/jobs/{id}/approve` | POST `ApproveRequest`: review, acceptance, and an optional merge as one decision |
| `/v1/jobs/{id}/change` | GET the changed files and the patch from the base to the candidate, capped at 200 KB |
| `/v1/jobs/{id}/problems` | POST `ReportProblemRequest` on merged work; lowers the project's ask policy |
| `/v1/jobs/{id}/merge` | POST `MergeRequest` for the exact accepted candidate and job version; requires a human with `merge`. The merge is `verifying` until the required checks pass on the merge commit |
| `/v1/jobs/{id}/acceptance` | POST `AcceptRequest`; the core validates and publishes the authored record |
| `/v1/projects/{id}/refinements` | GET a project's refinements, optionally `?workItemId=`, POST `StartRefinementRequest` |
| `/v1/refinements/{id}` | GET one refinement with its proposal and before and after scores |
| `/v1/refinements/{id}/accept` | POST `AcceptRefinementRequest`; rewrites the ticket through an expected-revision write |
| `/v1/refinements/{id}/reject` | POST `RejectRefinementRequest` |
| `/v1/worker/claims` | POST `WorkerClaimRequest` |
| `/v1/worker/heartbeat` | POST `HeartbeatRequest` with exact attempt and dispatch IDs and an optional `progress` note |
| `/v1/worker/reports` | POST `WorkerReportRequest` with exact attempt and dispatch IDs |

Job queries support `projectId`, `status`, `after`, and a `limit` between 1 and 500. Submission performs admission and acquires the claim in the core. There is no separate preview command that silently starts work. The browser first displays live orientation and administrator-owned policy, then explicitly submits. Submitters cannot remove required checks or supply arbitrary command text.

Task disposition is separate from authored completion and execution. Queue membership comes from the project's Current commitments. Archive retains the record and removes that commitment; restore keeps it unqueued. Delete requires an unqueued, unreferenced draft with no execution history. An explicit stop holds the work item until it is queued again. A task operation that writes both the record and project links retains a publication plan and reconciles it after interruption. Restart resumes safe plans automatically. A conflicting external edit leaves the project frozen with `publicationPending`; resolve the named file before using the authorized reconcile operation. Retry an uncertain command with the same request ID and payload.

Task activity returns at most 100 jobs, refinements, and expectation proposals of each kind, newest first. `truncated` identifies additional retained history; it does not imply deletion. Polling omits artifact contents; fetch each authenticated artifact endpoint to read its output. Captured admission and requirement revisions belong to the original run even when the current task changes.

The API returns JSON errors with `code`, `message`, and optional safe `details`. Invalid input uses 400, missing authentication uses 401, denied capabilities use 403, hidden or absent resources use 404, and stale versions or conflicting claims use 409. Unmet gates and exhausted bounds use 422. Unsupported media uses 415, oversized bodies use 413, and a locked installation uses 503. Worker `no_work` is an explicit 404 error code. Internal errors return a generic 500 without their underlying cause.

The typed `factoryclient.Client` keeps the token in headers, rejects redirects, and permits cleartext HTTP only on loopback. Its response limit is 16 MiB. It returns `*factoryclient.APIError` for a service rejection. A timeout is an uncertain transport result, not proof that a mutation failed. Retry an unchanged operation with the same request ID.

## Browser behavior

The workspace opens on the project's board, with list and graph views of the same cards. Each task has a permanent page with Task, Run, and History tabs. Task provides a spacious editor, safe Markdown preview, criteria, dependencies, and spec/context links. Run shows the stage path, available worker progress and output, checks, and authorized intervention. History retains previous jobs and proposals with their revisions and evidence. An administrator can register existing projects and fixed executable profiles, create distinct users, issue or revoke tokens, disable users, and grant project capabilities.

The task page polls run activity independently of the board and editor. Follow-active-step is optional; manual inspection and expanded output stay in place during updates. A stale save keeps the local draft and offers the latest record for comparison. The page identifies its last update and connection errors. Worker contact and task polling are separate observations: a successful poll does not prove a worker is alive. Usage is shown only when reported; captured output may be available only after an attempt ends.

Interface text uses Plus Jakarta Sans and code uses JetBrains Mono, both served from the installation with their licenses. The theme follows the system setting, and the toggle stores light or dark in local storage. Review and submission forms keep their input while a request fails. Cancellation remains pending until the core records a terminal observation. Missing or skipped evidence remains visible and cannot pass through the browser.

Independent verification is queued by the controller and executed through the worker protocol. The browser does not offer a button that turns a check into a pass. Fixture projects and fixture operation are labelled. The core prohibits fixture evidence from accepting an ordinary project.

The worker protocol carries service-owned workspace paths. A worker must run with access to those paths and follow the core's durable dispatch and terminal-receipt contract. Authentication over a network does not provide Git workspace transport or prove that a remote process stopped.
