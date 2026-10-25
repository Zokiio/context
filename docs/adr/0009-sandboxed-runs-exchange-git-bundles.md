# Sandboxed runs exchange code as Git bundles and stop with proof

## Context

[ADR 0005](0005-factory-controller-owns-operational-state.md) gives one service process ownership of an installation's operational store and workspaces. Workers receive service-owned worktree paths and run with the operator's permissions. The platform design states that service authorization does not confine a native agent's filesystem, commands or network, and the operator trusts each worker environment.

That trust no longer holds once agents run unattended. The builder's `go test`, the expectation author's tests and every required check execute code that an agent wrote. That code runs as the operator's OS user, with the worker's environment. It can read the operator's token file and approve its own change. Every cancelled or timed-out native run also ends `unknown`, because killing a process group does not establish that the agent's work stopped.

A worktree cannot simply be mounted into a sandbox. It shares hooks, refs, objects and config with the project repository. A confined agent with write access to it could plant a hook that runs outside the sandbox. Host Git commands run in a directory the sandbox wrote would also obey its config.

## Decision

On 25 October 2026 the user selected Docker-compatible containers as the first sandbox, deny-by-default egress through a Waymark proxy, worker-held secrets, and remote workers specified now and built when needed.

A sandboxed run never mounts a service worktree or the project's `.git`. The service sends the input revision as a Git bundle. The worker clones it inside the container and streams the candidate back out as a size-limited bundle. The service verifies the bundle, fetches its objects and materializes the candidate in a worktree only the service writes. Verification, review and merge keep their current inputs.

A worker owns only disposable sandboxes and its receipts. The service keeps ownership of the repository, issued bundles, fetched candidates and every worktree used for decisions. The same transport serves workers on other hosts.

A sandboxed run counts as stopped only when the container runtime reports that the container's processes ended. The worker attaches that observation as a stop proof, and only such a proof turns a non-fixture cancellation into `cancelled`. Without it, the outcome stays `unknown` and the claim stays held, as before. A worker may settle its own `unknown` attempt by finding the attempt's labelled container and proving it stopped. The service never moves an uncertain attempt to another worker.

## Alternatives considered

An OS sandbox wrapper (Seatbelt or bubblewrap) needs no daemon or image and keeps the macOS keychain login. It cannot confirm a stop on macOS, because a detached process can leave the wrapper's control. A dedicated OS user protects token files and can confirm a stop by checking that the user has no processes left. It offers no network control and needs privileged setup.

Mounting worktrees into containers would keep the current path-based protocol. It exposes the project repository's hooks and refs to the sandbox, and it ties remote workers to a shared filesystem.

## Consequences

Docker becomes an optional runtime dependency for sandboxed profiles. Unsandboxed execution remains for fixtures and trusted commands, and a project can require sandboxes. Sandboxed Claude Code runs authenticate with a named secret instead of the operator's keychain login.

ADR 0005 still holds: the service owns all authoritative workspaces and operational state. Sandbox copies are disposable worker property that carry no authority.

Full bundles cost transfer time on large repositories. Incremental bundles are a later optimization.

Confinement covers the sandboxed process. It does not cover the container runtime's own operator, and it does not yet cover the read-only refiner and judge. A stop proof establishes that the sandbox's processes ended and its egress closed. Effects already made through allowed hosts remain in the run's egress log.
