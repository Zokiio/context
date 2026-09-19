# Ticket 06 supplemental instruction recheck

The P2 finding in `06-instruction-review.md` is resolved by `625fe74c95a86aff5754f9c8d8308d2d79395f62`.

Step 3 now explicitly compares the retained collection with resume's current source paths and digests. A mismatch requires assessment plus collection and reading of new exact task-context output before action. The instruction to retain the collection governing the work, and never publish the older collection after using newer requirements, closes the A/B baseline gap. Publication still avoids silently fetching a newer baseline at checkpoint time.

Also inspected the PROFILE pointer correction and `ce8b183645dbaf5869e0b68cde9202f2d8fe218e`. Namespace selection now requires known, unambiguous identities from a current JSON report without incorrectly excluding partial reports. No obvious regression found.

This was a narrow prose recheck against the original finding. The parent's strengthened filesystem trial remains separate verification. The initial review is preserved; no product or record files changed.
