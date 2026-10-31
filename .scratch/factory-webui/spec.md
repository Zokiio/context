# Web workspace rework

On 30 October 2026 the user asked for a full rework of the factory web workspace. The assessment behind this plan ran the `dev` build against the Opsbase grooming trial records: fifteen work items, sixteen refinement proposals, no jobs. The visual tokens chosen on 23 October stay. The screen structure and the way the frontend is built change.

## Observed problems

- The only pending decisions were invisible. Every work item had a proposed refinement with flagged criteria. Its card read "Draft", the decision view read "No decision is waiting for you", the run view read "Sharpen requirements · Needs you", and the proposal was reachable only as a collapsed row under History. The board attaches refinements to queued items only.
- "Up next" held fifteen unqueued drafts, each with a cost and an empty six-step bar, and the task library under the board listed the same fifteen. The list view joined their titles into one paragraph.
- The plan and approve decisions render in three places with three layouts: the board drawer, the decision tab and the separate job page.
- The run view drew an eight-step grid, repair included, for an item that had never run.
- At phone width the top bar wrapped to three rows, the navigation scrolled sideways, and the board scrolled sideways across six columns.
- The frontend is one 1,510-line script with its own element helper, hash router and DOM patching, plus a 580-line stylesheet. The browser regression runs against a fake server.
- Board cards carried a zero `updatedAt`, and columns sorted by it.

## Outcome

A person opens the workspace and sees what needs them, in order, and nothing else at the same weight. Opening an item shows the pending decision as the page, with the task text, the run and the history under it. Each decision has one home. The screens work at phone width. A change to a screen is made in that screen's component and checked in a browser against the real service.

## Screens

### Home

- The page opens with the decisions waiting for the signed-in person, oldest first, one row each: what is asked (agree what done means, answer the agent's questions, approve the change, merge), the item title, and how long it has waited. One action opens the item.
- Below, the rest of the project's work grouped by stage: Running, Up next, Drafts, Done. Each group is collapsed to a count with its latest items visible. Drafts are tasks saved outside the queue; they never appear under Up next.
- Cards show title, state and one line of status. Costs, revisions, scores and step bars are not on cards.
- The graph view stays reachable from Home. The list view, the task library and the board drawer go away.
- Project switching, the theme toggle, settings and sign-out stay in the top bar. The top bar fits one row at phone width, with secondary items behind one menu.

### Item page

- One route per work item. When a decision is pending it is the first section and holds the actions. The stored records decide which one shows: a proposed refinement asks for review of the proposal; open refinement questions ask for answers; accepted expectations or no refinement ask to agree and start; a job ready for review or acceptance asks for approval; an accepted unmerged job asks to merge; otherwise the section names the current stage and who acts next.
- The decision shows whether or not the item is queued. Queueing is part of the agree step, not a precondition for seeing the proposal.
- Task text, requirements, dependencies and links follow, with the existing editor and its save, stale-save and freeze rules.
- The run section shows only the steps that exist for this item, with the live step open. Steps that have not started are one line each. Worker output, artifacts, check results, disputes and intervention controls keep their current behaviour.
- History lists previous jobs, attempts, proposals and decisions, one row each, expandable.
- The separate job page is removed. A link to a job opens its item with the run section expanded.

### Settings

How much to ask you, Agents, Administration and project access keep their current content on the new shell. Insights stays as it is.

## Construction

- The browser workspace gets a build step. Screens are written as components in TypeScript with a small component library. [Choose the browser workspace stack](../records/factory-webui/decisions/01-frontend-stack.md) records the choice. The built output is embedded in the `ctx` binary as today, so `ctx factory serve` and `ctx factory solo` keep serving the workspace with no runtime dependency.
- The Gups tokens stay as the shared stylesheet variables: cool neutrals, violet as the only action colour, status colours that brighten in dark mode, Plus Jakarta Sans and JetBrains Mono served locally with their licences. Light and dark follow the system setting with a toggle.
- The service stays the permission authority; hiding a control remains a convenience. Project content, worker output and errors render as text. The same-origin rule, the no-token-in-URL rule and the Content Security Policy stay, so the build may not depend on inline scripts or styles.
- Until every screen is ported, the legacy workspace stays reachable under `/legacy/` and the new shell links to it for screens not yet ported. The legacy script, stylesheet and fake-server regression are removed in the last step.
- Browser checks run against the real service with fixture workers, as the offline platform demonstration does.

## Service changes

- The board response carries the pending refinement or expectation for every work item, queued or not, and the decision the person is asked to make.
- Cards carry a real `updatedAt`. Items without one sort by title.

## Out of scope

- New factory behaviour. The rework shows what the service already records.
- Changes to the CLI or its JSON contracts.
- A visual redesign of the tokens or typography.

## Order

1. [Show every work item's pending decision on the board](../records/factory-webui/issues/01-board-pending-decisions.md). Service only. It unblocks the item page and can ship first.
2. [Build the browser workspace from components](../records/factory-webui/issues/02-frontend-toolchain.md), after the stack decision.
3. [Open the workspace on what needs you](../records/factory-webui/issues/03-home.md).
4. [Decide what done means on the item page](../records/factory-webui/issues/04-item-page-plan.md).
5. [Approve and merge from the item page](../records/factory-webui/issues/05-item-page-approve-merge.md).
6. [Move settings to the new shell and retire the legacy workspace](../records/factory-webui/issues/06-settings-and-cleanup.md).

Each step is one pull request with its own browser check.
