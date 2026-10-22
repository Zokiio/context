# One durable controller owns factory operations

## Context

The user selected the complete factory platform before target-project pilot work on 22 October 2026. Existing readers and agent guidance provide context and workflow conventions. They do not supply transactional claims, authenticated shared operations, or execution supervision.

Several users and clients must see the same jobs, claims, attempts, evidence, and acceptance decisions. External worker execution and authored record writes cannot occur atomically with an operational database transaction.

## Decision

Use one Go controller service with a private SQLite operational store per installation. CLI and browser clients use its application operations. The controller coordinates one active implementation claim per work item, idempotent commands, authenticated project access, bounded attempts, and reconciliation.

Keep OKF project records authoritative for intent, work requirements, relationships, and recorded acceptance. Store operational job state and immutable execution snapshots separately. Register existing project identities and locations without creating a second backlog.

Retain external-effect intent before dispatch or record publication. Uncertain execution holds its claim and requires reconciliation before another invocation. SQLite transactions establish local command and claim coordination; they do not establish exactly-once external effects.

Project policy owns required check definitions and permitted execution modes. A submitting operator cannot remove required checks, and fixture evidence is valid only for a fixture-designated project. Verification, review, and acceptance bind to the exact candidate and admitted requirements.

Acceptance publishes immutable evidence and an attributed Acceptance record, updates the ticket through an expected-revision write, and evaluates the result with the existing reader. Operational job completion alone cannot satisfy a prerequisite. Preserve the publication plan so interrupted writes can be reconciled by content identity.

Existing coding-agent environments remain execution adapters. Gups is a later development target, not an execution prerequisite. No third-party workflow engine is selected by this decision.

## Alternatives considered

A custom file journal could provide durable local operation without a database dependency. It requires implementing transaction framing, replay, corruption recovery, snapshots, and idempotency storage. SQLite supplies that transaction boundary behind a smaller maintenance surface.

An external workflow engine may later provide execution machinery. It does not remove Waymark's responsibility for requirement versions, acceptance policy, and project identity, and none has been selected.

Several active controllers with a network database would support a larger deployment. One installation authority already serves multiple users and clients, so active replicas add coordination requirements without establishing a needed first-platform outcome.

## Consequences

One service process owns an installation's operational store and workspaces. Shared clients use that service. Independent installations and Git synchronization do not share claims.

SQLite becomes a runtime dependency. The first implementation preserves the repository's Go 1.25 baseline through a compatible pure-Go driver.

The controller must handle the gap between transactions and external effects explicitly. Failed, pending, cancelled, and unknown outcomes remain distinct. Authentication of service calls does not by itself confine an external worker's filesystem or provider tools.

The original reader commands remain read-only and usable without running the service. Generated evidence and Acceptance records follow each project's retention policy.
