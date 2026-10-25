---
type: WorkItem
id: 1d283680-8b63-40b7-a90c-19cc15b00281
title: Queue matching work on a person's behalf and keep the first agreement with a person
triage: ready-for-agent
execution: unstarted
---

This is the ticket that lets work start without anyone queueing it. A trigger a person saved queues its matches as that person, as the ask policy already delegates decisions. Imported text stays away from agents that write files or run commands until a person agrees what done means.

## Spec

- [Factory intake](../../../factory-intake/spec.md)

## Context

- [Tracker intake decision](../../../../docs/adr/0006-tracker-intake-ownership-and-delegated-pickup.md)

## Blocked by

- [Keep agreed requirements when the tracker changes](03-keep-agreed-requirements.md)
- [Poll bound trackers and import what pickup triggers match](04-poll-and-match-pickup-triggers.md)

## Blocked by decisions

None.

## Acceptance criteria

- A trigger's action can be `queue`. Only a human project administrator who holds coordinate can save it, it must name at least one condition, and it records who saved it and when.
- The service adds each match of a queueing trigger to Current commitments as the person who saved it. It rechecks that person's capabilities each time and records the decision as delegated, with the reason "pickup trigger <name>". If the check fails, the item stays unqueued and the trigger shows why.
- When a queueing trigger stops matching an unstarted item that it queued, the service removes the item from the queue. An item a person queued stays queued. Active work gets only a "No longer matches <name>" fact.
- For an imported item without `requirementsAgreedAt`, the autopilot may start the read-only refiner. It does not accept a refinement, start the test author or start a build, whatever the ask policy says. The card asks a person to agree what done means. After agreement the ask policy governs the item as it governs native work.
- Folded card details show "Queued by trigger <name> for <person>". Metrics count delegated queue decisions apart from a person's own decisions.
- Tests run autopilot passes with the ask policy set to never ask, before or after the build. An imported item stops at the plan decision, and a native item still runs unattended. Further tests cover the capability recheck, unqueueing when a trigger stops matching, and a person-queued item that stays queued.
