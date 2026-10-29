---
type: WorkItem
id: 8fba2652-0a68-4625-bc2d-b6e99ca7f01d
title: Sign in once per browser
triage: ready-for-agent
execution: unstarted
---

## Spec

- [Run the factory for a small team](../../../factory-team-scale/spec.md)

## Context

- [Factory HTTP and browser access](../../../../docs/factory-http.md)
- [Factory controller decision](../../../../docs/adr/0005-factory-controller-owns-operational-state.md)

## Blocked by

None.

## Blocked by decisions

None.

## Acceptance criteria

- Signing in with a valid bearer token creates a server-side session and sets an HttpOnly, SameSite=Strict, Path=/ cookie, marked Secure on a TLS listener. The browser no longer keeps the token in `sessionStorage` or `localStorage`.
- A new tab, or a task link opened from outside the workspace, loads signed in while the session is valid.
- A session ends at sign-out, seven days after it started, when its token is revoked or expires, or when its user is disabled. The next request returns 401 and the browser shows sign-in without losing an unsaved draft in the open form.
- The service stores only a digest of the session secret. A session authenticates the same principal as its token, and every capability check is unchanged.
- Mutations through a session keep the existing Origin, fetch-context and JSON content-type checks, so a cross-site form post or fetch is refused. CLI, worker and `factoryclient` bearer authentication is unchanged, and query-string credentials stay refused.
- `docs/factory-http.md` describes the session and its limits. Tests cover sign-in in a new browser context, revocation, expiry, a disabled user, sign-out and cross-site refusal.

## Comments

- 2026-10-25: Joakim accepted token-then-session sign-in for the pilot organisation; SSO is not required.
