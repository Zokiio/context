---
type: Decision
id: baf79f05-daa1-4363-9d70-4bcf56eea3c9
title: Where the pilot's records live
decisionState: open
---

# Where the pilot's records live

The service writes project records: tasks, refined criteria, the ticket's Acceptance link and execution, and merge notes. On a shared host these writes must reach the team's remote. A record commit can't ride on a `waymark/<job>` pull request branch, because it would change the head and discard that head's host evidence. A protected target branch refuses a direct push. This decision picks the route. It blocks [publishing record writes](../issues/08-publish-record-writes.md).

## Options

**A separate records repository (recommended).** The bundle lives in its own repository, and the service pushes to that repository's main branch by compare-and-swap. The factory already registers `bundle_path` and `repository_path` separately, and the vision lists separately stored records as a supported configuration. Code, locked tests and merges stay in the monorepo and go through the code-host plan's pull requests. This adds no human decision and doesn't fight branch protection. The cost: teammates clone two repositories side by side so relative Spec and Context links resolve, and record changes aren't reviewed as pull requests.

**A rolling records pull request in the monorepo.** The service commits record writes to one `waymark/records` branch and keeps a pull request open against the target. A teammate, or GitHub auto-merge, merges it. Records stay next to the code. The cost: a protected branch that requires approval turns each records merge into a third human decision per change. The service's checkout also carries unmerged record commits on top of the target, which conflicts with the code-host plan's rule that it fast-forwards the local target only when the tips match.

**A direct push to the target with a bypass.** The service's account gets a branch-protection bypass and pushes record commits straight to main. This is the least code. The cost: the enterprise has to grant a bot a bypass on main, and the bot's pushes race with teammates' merges.

## What the answer changes

Ticket 08's push route and its tests, and the team-host guide's setup steps in ticket 06.
