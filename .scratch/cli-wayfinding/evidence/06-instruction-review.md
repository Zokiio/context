# Ticket 06 independent instruction usability review

Reviewed committed `c8bb9d1` in `/tmp/context-cli-instructions`, following the AGENTS pointer into recovery instructions, the local profile, and task-context guidance. Excluded the already requested PROFILE wording correction and stronger filesystem trials.

One remaining finding:

- **[P2] Preserve the baseline actually used after the startup refresh.** `.agents/skills/recovery-notes/SKILL.md:38` instructs the agent to read a newly collected `resume.context` after step 2 saved standalone context. Line 45 then directs publication to retain step 2's file. If another writer changes requirements from A to B between these commands, the agent can follow B during resumed work but publish A as its baseline. The approved spec requires: "Retain the requirement collection actually used for the work described by the checkpoint" (`.scratch/cli-wayfinding/spec.md:135`); ticket 06 likewise requires "the exact context output actually used" (line 22). Add an explicit branch when refreshed source identities or digests differ: assess the change, collect and read a new exact standalone context output before using it, and retain the collection that governed the described work. Keep the prohibition on silently refreshing at publication time.

No other remaining findings. A fresh agent can discover the workflow through AGENTS, find scoped notes through resume, distinguish historical checks from evidence, preserve competing predecessors, record authoritative blockers, and locate the stopped-writer quarantine/reset procedures. The documented no-overwrite rules and unknown/live refusal are clear; their pending filesystem trials remain the author's separate verification work.

No product or record files changed.
