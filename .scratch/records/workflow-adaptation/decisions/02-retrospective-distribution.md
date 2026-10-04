---
type: Decision
id: a5486e68-20f2-4650-899a-025d944e4bc7
title: Choose where the retrospective skill is installed
decisionState: resolved
---

# Choose where the retrospective skill is installed

## Resolution

2026-10-04: The user chose to install the retrospective skill through `ctx init`, explicit-only like orchestrate. [Ticket 03](../issues/03-retrospective.md) implements it.

## Basis

The [specification](../../../workflow-adaptation/spec.md) proposes an explicitly invoked `retro` skill. The need comes from one manual inspection of two Axpilot sessions, run from the Waymark checkout on 2026-10-04.

## Options

- **Install it through `ctx init`.** Every prepared project receives the skill, explicit-only like orchestrate. Projects where orchestrated work runs can review it in place.
- **Keep it in this repository first.** Retrospectives run from the Waymark checkout against other projects' sessions, as on 2026-10-04, until a second use shows that projects need their own copy.

## Recommendation

Install it through `ctx init`. The sessions worth reviewing run in prepared projects, whose users are not Waymark developers. The skill writes nothing without approval, so an unused copy costs little.

The second option follows the rule that repeated observations set scope. Choose it if one manual inspection is not enough evidence for distribution.
