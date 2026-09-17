# Detect ACL changes without relying on timestamps

Independent reviewer `Codex /root/merge_review_spec_setup` tested source `a0ecc4cf143886e687c072ceffc9c8d1a0963a86` on a native Linux 6.8.0-64-generic/aarch64 filesystem inside an isolated container. Ownership and ACL preservation passed, but a concurrent ACL change could share the original change time.

The retained reproduction revoked an ACL read grant for fixture UID 1234 during temporary-file sync. The file's owner, group, mode, contents, and change time were unchanged. Setup returned success and restored the revoked read grant during replacement. The ACL permission changed from 4 to 0 and then back to 4.

The setup specification requires detecting changes between reading and writing and preserving permissions. The correction must compare a captured permission snapshot and restore only captured values. Timestamp comparison alone does not establish that ACLs are unchanged.

[The failed runtime checks](pinned-linux-runtime-recheck-v2.json) and [the revoked-grant reproduction](linux-acl-clock-collision-evidence.json) retain commands, exact source revision, binary identity, and output. The [manifest](manifest.json) records immutable probe sources and digests. A later independent recheck will record the correction; this evidence remains the original failed observation.
