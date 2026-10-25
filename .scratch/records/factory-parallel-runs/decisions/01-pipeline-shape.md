---
type: Decision
id: 28a85815-7551-4021-a084-0fd9330399fa
title: Choose how a project shapes its factory pipeline
decisionState: open
---

# Choose how a project shapes its factory pipeline

## Basis

Every work item runs one fixed line: sharpen, locked tests, agree, build, checks, approve, merge. Project policy already switches parts of it. The ask policy decides when a person is asked. Protected paths turn on the test author. Required checks decide what verification runs. The refiner runs only when a criterion looks vague. Fabro and StrongDM Attractor let projects author graphs in a DOT subset with human gates anywhere, and SWE-AF sends some issues to deeper QA. The [research note](../../../../docs/research/software-factory-2026-10-22.md) and its pinned sources describe them.

No trial has asked for a different stage order or an extra gate. Sibling factory plans written on 2026-10-25 propose new stages: an agent review after the checks, a draft pull request with host checks, and observation of merged work in an environment. Each fits the fixed line as a stage that configuration turns on, and none needs a different order. The likeliest near-term need is a check that runs only when certain paths change, such as browser checks for the board's assets. On 2026-10-25 the user saw examples of each option and wasn't sure, and no ticket in the [specification](../../../factory-parallel-runs/spec.md) depends on the answer.

## Options

- **Fixed line with policy switches.** Keep the line. Turn stages on through project policy, and add a field when a trial needs one, for example a required check with a list of paths that trigger it. Two decisions per change hold by construction.
- **DOT graph per class of work.** Projects author Attractor-compatible graphs, which admission validates and freezes per job. This needs a parser, a validator, a renderer, a graph runner, and a rule rejecting human gates beyond the two decisions. Attractor's success conventions, such as partial success at goal gates, would have to be excluded explicitly.
- **YAML stage list per class.** An ordered list from a fixed menu of stages per class, plus conditional checks. A class chosen by changed paths is only known after the build, so it can only select checks, not earlier stages.

## Recommendation

Keep the fixed line with policy switches. Add the first switch when a trial project shows a check being wasted on unrelated changes or skipped when it mattered. Revisit a graph or stage list only when a project needs a stage order or a gate that a switch cannot express.
