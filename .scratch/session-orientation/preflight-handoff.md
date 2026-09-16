# Retained reader handoff

Start with project orientation, choose from the observed eligible list, and then request that work item's full task context. The reader reports facts and reasons; the workflow actor chooses and records execution.

## Verified build and scope

- Reader: `/var/folders/hd/nxsy37l9371g67_scvcnpxb00000gn/T/ctx-orientation-kppqsre4/ctx-preflight`
- Reader SHA-256: `70d37e8989db00f305e05d0ce0f498b7fa00dfca45ab1a6ee74d1f93ff789a21`
- Matching source commit: `4ccb1f7171082fa97af7455b0d86f56c5228b71d`.
- Project bundle: `/Users/zoki/code/context/.scratch/records`.
- Allowed source root: `/Users/zoki/code/context`.
- Default limits: 100 inspected files and 1,048,576 source bytes.

The [successful program preflight](evidence/07-preflight-verification.json) records the exact source manifest, commands, results, toolchain versions, and actual tested working-tree identity. Every source-manifest entry was compared with its corresponding blob in the matching commit. The [independent review](evidence/review.md) records resolved findings; its rebuilt reader has the same digest. Record observations and acceptance decisions retain their own revisions and times.

## Pickup workflow

Verify the retained binary's digest. Make orientation the first project query. Run both formats with the explicit scope above, from any working directory:

```sh
/var/folders/hd/nxsy37l9371g67_scvcnpxb00000gn/T/ctx-orientation-kppqsre4/ctx-preflight orient \
  --project /Users/zoki/code/context/.scratch/records \
  --allow-source /Users/zoki/code/context

/var/folders/hd/nxsy37l9371g67_scvcnpxb00000gn/T/ctx-orientation-kppqsre4/ctx-preflight orient \
  --project /Users/zoki/code/context/.scratch/records \
  --allow-source /Users/zoki/code/context --json
```

Select work from the report's eligible list and explain its execution, commitment, triage, and readiness. Use the selected source path with `ctx context --project ... --ticket ... --allow-source ...`. Read every selected source's full text, path, and inclusion reasons. Require complete context before recording execution as in progress.

Keep command captures outside the project during read-only comparisons. Preserve exact invocations, exit statuses, reader and source identities, observed source bytes or digests, and the facts used for selection. Compare project files and Git state across the reads before making explicit workflow edits. Retain the actual pickup state in evidence, including any failures and corrections. Relevant documents without authored links remain separate observations.

Complete the selected work through the authored [acceptance procedure](../../docs/agents/acceptance.md). Historical acceptance proves matching retained assertions and snapshots; it does not by itself certify the current checkout, grade the evidence, or establish human approval.
