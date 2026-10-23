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
| `/v1/projects/{id}/board` | GET one card per open work item: its column, state, the decision it waits for, and its cost so far |
| `/v1/projects/{id}/plan` | POST `ApprovePlanRequest`: answer, accept a refinement or locked tests, and start the next step; `continue` gives a go-ahead to build |
| `/v1/projects/{id}/ask` | PUT `SetAskPolicyRequest`; requires a human project administrator who holds the delegated capabilities |
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

The API returns JSON errors with `code`, `message`, and optional safe `details`. Invalid input uses 400, missing authentication uses 401, denied capabilities use 403, hidden or absent resources use 404, and stale versions or conflicting claims use 409. Unmet gates and exhausted bounds use 422. Unsupported media uses 415, oversized bodies use 413, and a locked installation uses 503. Worker `no_work` is an explicit 404 error code. Internal errors return a generic 500 without their underlying cause.

The typed `factoryclient.Client` keeps the token in headers, rejects redirects, and permits cleartext HTTP only on loopback. Its response limit is 16 MiB. It returns `*factoryclient.APIError` for a service rejection. A timeout is an uncertain transport result, not proof that a mutation failed. Retry an unchanged operation with the same request ID.

## Browser behavior

The workspace opens on the project's board, with a list view of the same cards. The two decisions open in a side panel over the board, and each also has its own page. A job page shows its stage, the change, and the merge. Its captured requirements, attempts, check evidence, events, reviews, and acceptance sit under technical details. An administrator can register existing projects and fixed executable profiles, create distinct users, issue or revoke tokens, disable users, and grant project capabilities.

The board checks for changes every three seconds while the tab is visible and no panel or field is in use. It does not re-render when nothing changed. Interface text uses Plus Jakarta Sans and code uses JetBrains Mono, both served from the installation with their licenses. The theme follows the system setting, and the toggle stores light or dark in local storage. Review and submission forms keep their input while a request fails. Connection errors identify uncertain outcomes. Cancellation remains pending until the core records a terminal observation. Missing or skipped evidence remains visible and cannot pass through the browser.

Independent verification is queued by the controller and executed through the worker protocol. The browser does not offer a button that turns a check into a pass. Fixture projects and fixture operation are labelled. The core prohibits fixture evidence from accepting an ordinary project.

The worker protocol carries service-owned workspace paths. A worker must run with access to those paths and follow the core's durable dispatch and terminal-receipt contract. Authentication over a network does not provide Git workspace transport or prove that a remote process stopped.
