# Project and context management

Projects retain their identity across record locations and repository arrangements. Workspaces group projects without duplicating their authoritative records.

## Language

**Project**:
A logical initiative with its own identity, goals, specifications, work, and decisions. A project can span repositories, and one repository can contribute to several projects.
_Avoid_: Repository, workspace

**Project record**:
The authoritative collection of a project's authored information, including its specifications, work, decisions, and evidence.
_Avoid_: Cache, workspace

**Record store**:
The location of a project's authoritative records. Its location can change without changing the project's identity.
_Avoid_: Project, repository binding

**Repository binding**:
The relationship between a project and a contributing code repository, optionally limited to specified paths within that repository.
_Avoid_: Record store

**Workspace**:
A grouping of project locations and checkout information. A project can participate in several workspaces without duplicating its authoritative records.
_Avoid_: Project, record store

**Work item**:
A unit of work tracked within a project. It can carry its own requirements and acceptance criteria or refer to a separate specification.

**Pickup trigger**:
A project-configured condition that identifies a work item as a candidate for pickup.
_Avoid_: Work readiness, assignment, permission to start

**Work readiness**:
Whether a work item has explicit acceptance criteria, available required context, satisfied dependencies, and no unresolved decision marked as blocking.
_Avoid_: Pickup trigger, workflow state, assignment, permission to start

**Implementation claim**:
The record identifying the agent responsible for active implementation of a work item. A work item has at most one active implementation claim, distinct from its human assignment.
_Avoid_: Human assignee, work readiness

**Specification**:
A description of the desired behavior for a feature or change that one or more work items implement.

**Grooming**:
Investigation and clarification of a work item to establish its scope and identify missing information. Its output is a proposal of findings, affected components, open questions, and any suggested ticket splits.

**Acceptance criterion**:
A checkable condition that a work item must satisfy to be accepted.

**Completion evidence**:
Recorded results or observations that support a claim that a work item satisfies its acceptance criteria. Evidence retains its origin and the revision it concerns.

**Task context**:
The source content selected for one work item, with the origin of each included record and the reason for its inclusion.
_Avoid_: Summary

**Session orientation**:
A project-level view of goals, current commitments, ready work, and unresolved decisions that helps an agent identify suitable work.
_Avoid_: Task context

**Handoff**:
A durable account of completed and unfinished work, relevant code revisions, verification results, and unresolved questions for a successor continuing the work.

**Context completeness**:
Whether every source selected for a task context is present in the result. A dependency cycle can exist even when the context is complete.
_Avoid_: Work readiness, document validity
