# Acceptance authoring clarification

Workflow actor: Codex coordinating agent, 2026-09-16. No human approval or new historical test run is asserted.

A public-CLI probe using the retained ticket 03 reader showed that adding a separating blank line before a first Acceptance heading changes retained requirement bytes. Preparing the empty section before obtaining fingerprints makes subsequent insertion of the current link stable. The [boundary observation](05-authoring-boundary.json) retains exact body/suffix bytes, observed hashes, reader digest, and the reader's tested source identity.

The acceptance guide now tells authors to prepare the empty section and its surrounding line breaks first. This clarifies the existing raw-byte encoding; it does not change the operation's encoding or the agreed acceptance boundary.

Tickets 01 through 04 explicitly select that guide as context. Their earlier Acceptance records therefore require reassessment when its whole-file digest changes. Preserve those decisions and their original evidence. New decisions can retain the original application tests because this change concerns the authoring procedure; the new boundary probe supplies evidence for the clarified procedure. The coordinating agent will verify both stale detection and fresh replacement through the new application validator before closing ticket 05.
