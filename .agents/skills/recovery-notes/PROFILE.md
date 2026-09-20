# RecoveryNote version-1 profile

Use this reference when publishing a checkpoint. It is the local template for the `ctx resume` cache format.

## Namespace

Read `scope.projectId`, `scope.taskId`, and `scope.workingDirectory` from a current `ctx resume` JSON report. Publish only when the selected Project and WorkItem identities are known and unambiguous. Do not guess an unknown ID. For each ID, apply the reader's effective identity rule: trim surrounding whitespace and preserve the remaining case. Encode that effective value as UTF-8 with no newline. SHA-256 each byte sequence as lowercase hexadecimal.

Store the new observation here:

```text
<scope.workingDirectory>/.context-cache/resume-v1/
  <sha256(utf8(effective-project-id))>/
  <sha256(utf8(effective-task-id))>/
  observations/<observation-id>/
```

`<observation-id>` is a newly generated random UUID in lowercase hyphenated spelling. Create its directory as a new directory. An existing path belongs to another write attempt; generate another UUID.

## Files

Copy the exact bytes of the task-context JSON used by this session to `context.json`. Do not reformat, regenerate, or replace it with a later collection. Calculate `contextSHA256` from those exact file bytes with SHA-256, lowercase hexadecimal, and no added newline.

Write this frontmatter, filling every placeholder with the required value:

```yaml
---
type: RecoveryNote
version: 1
id: <observation-id>
projectId: <effective-project-id>
taskId: <effective-task-id>
observedAt: '<RFC-3339 timestamp with offset>'
actor: <nonempty session identifier>
predecessors: []
ticketPath: <nonempty ticket path>
checkoutRevision: null
contextFile: context.json
contextSHA256: <64 lowercase hexadecimal characters>
---
```

All shown fields are required. `type` is exactly the string `RecoveryNote`; `version` is the integer `1`; `id`, `projectId`, `taskId`, `actor`, and `ticketPath` are nonempty strings. `id` is the same lowercase UUID as the directory name. `projectId` and `taskId` use the effective identities above. `observedAt` is an RFC 3339 timestamp. `predecessors` is an array of distinct lowercase UUID strings. It can be empty, and cannot contain `id`.

`checkoutRevision` is the only nullable field. Use null when no checkout provenance was observed. Otherwise use an object with nonempty string `origin` and `revision` fields. `contextFile` is exactly `context.json`. Do not repeat YAML keys.

Use each of these level-two headings exactly once. `None` or an empty section records no observation; it does not establish completion.

```markdown
## Approach

## Completed

## Remaining

## Checks

## Questions

## Failed approaches

## Next step
```

Write `note.md` last through a unique same-directory temporary file, then rename that file to `note.md`. Never replace an existing finalized note.
