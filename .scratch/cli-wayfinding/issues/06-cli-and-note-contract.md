# What command and note-format contract implements the agreed local loop?

Type: grilling
Labels: wayfinder:grilling
Status: resolved
Blocked by: 02, 03, 04
Parent: [Direct and resume agent work with the local CLI](../map.md)

## Question

Draft and review one implementation-ready specification translating the resolved orientation, handoff, and resumption behavior into a concrete CLI and skill-authoring contract. The agent should propose routine technical choices using existing source and conventions, rather than ask the user to design a storage format.

Specify the command entry points, compact and expanded text behavior, full JSON and compatibility rules, diagnostics and exit statuses, and reuse of existing scope/context/orientation operations. Define task and checkout identification, cache location for embedded or separately stored records, note and snapshot representation, and discovery of absent, incomplete, or conflicting observations.

Specify bounded, authorized reading of retained and current source content, including uncommitted observations. Define how skills retain observations and avoid silently overwriting competing notes without claiming exclusive agent ownership or introducing managed CLI writes. Keep cache absence recoverable and authoritative project knowledge outside the cache.

Include the agreed real-task acceptance scenario and necessary implementation checks. Identify any contradiction or genuinely new product decision for human review; do not silently enlarge the milestone. Publish the proposed specification as `.scratch/cli-wayfinding/spec.md`, with links to decision owners rather than a second record of their rationale.

Inputs: [map and resolved decisions](../map.md), [reader contracts](../../../docs/readers.md), [discovery contracts](../../../docs/discovery.md), [tracker conventions](../../../docs/agents/issue-tracker.md).

## Specification

[Local orientation and resumption](../spec.md) translates the agreed behaviors into concrete commands, output rules, scope selection, note/checkpoint encoding, bounded reading, skill-authoring conventions, and acceptance checks. Its initial draft was presented for review; the completed contract and review basis are recorded below.

The main technical proposals are `orient --detail`, a new read-only `resume` command, an explicit checkout selector for direct bundle access, and immutable recovery checkpoints with predecessor links. Checkpoint branches expose competing notes without managed CLI writes or exclusive work claims. The draft preserves existing context/orientation JSON evaluation semantics and defines a separate resumption result.

2026-09-17: Draft reviewed against the current reader/discovery contracts and source types. It also identifies the stale direct `--project` example in the existing task-context skill for correction during implementation. Local document links and whitespace are checked; no product changes, tests, or implementation acceptance are claimed.

## Answer

The user said the proposed commands looked okay, clarified that text is the operator presentation and JSON is the agent presentation, and authorized continued contract review. The remaining routine technical choices were specified against the current reader/discovery contracts; no new product-scope decision was identified. The [specification](../spec.md) is the implementation handoff. Its exact protocol details are agent-authored technical design, not a claim that the user individually reviewed every field.

Use compact `orient` text with `--detail` for expansion and preserve full orientation JSON. Add read-only `resume --ticket` with explicit checkout selection for direct bundle access. Skills publish immutable, checkout-local recovery observations with predecessor references and retained task-context bytes; the CLI reads the observed graph and compares sources. Conflicting successors remain visible rather than being resolved by timestamp.

The final review tightened three correctness boundaries: supersede only observations actually incorporated into the work; retain the requirements actually used rather than silently replacing the baseline at publication; and require complete source sets before reporting additions or removals. It also specifies working-directory path resolution, note-field validation, cache budgets, and the distinction between prose check claims and structured acceptance evidence.

Review used the current `internal/cli/scope.go`, `internal/discovery/resolve.go`, `internal/taskcontext/context.go`, `internal/orientation/orientation.go`, public report types, and their documentation. Shared captured reads and a shared resumption budget require implementation work; they do not already exist merely because they are specified. Local links and whitespace were checked. No implementation tests or real-task acceptance trial ran in this planning session.

The wayfinding destination is reached: no known decision remains before implementation planning. Create implementation tickets from the specification in the normal tracker workflow; do not treat the discovery tickets as implementation commitments.
