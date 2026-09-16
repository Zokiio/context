# Use recorded acceptance to satisfy dependencies

Session orientation relies on an authored acceptance decision, retained evidence, and record checks when deciding whether a completed prerequisite satisfies a dependency. Authorized workflow skills may record acceptance, with the decision's author and any human approval separately attributed. This lets a read-only operation explain the basis for satisfaction while leaving the judgment about whether evidence meets the criteria with the acceptance decision's author.

Acceptance is tied to fingerprints of its requirements and evidence. Changes to those sources require reassessment, while the tested revision remains historical provenance and need not equal the current Git HEAD. The [session orientation specification](../../.scratch/session-orientation/spec.md) defines the record checks and fingerprint encoding. Its whole-body ticket coverage began as assumption A1 and was confirmed by the user during the ticket-breakdown review.
