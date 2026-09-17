# Independent setup Spec recheck

Actor: Codex `/root/merge_review_spec_setup`. Date: 2026-09-17. Tested source: `c59d12e6886874a00d42b821be24615a93c95ef6`, exported independently from Git. All 321 exported files remained unchanged.

**Clean: no remaining Spec findings in the setup correction.** The original PR #9 shared/personal race and permission-loss findings are resolved, including the later Linux equal-ctime ACL gap. The correction satisfies spec line 86: “Use atomic replacement and detect changes between reading and writing. Preserve existing file permissions and report failures without leaving truncated files.”

Native macOS discovery/setup/CLI and permission tests passed under `-race`, including independent same-process and separate-process conflict probes in both commit orders. Each race produced one commit, one conflict, and valid discovery. Matching declarations coexist. Group, mode, ACL, inherited-ACL removal, stale permissions, unreadable ACL, and protection-flag checks passed.

Native Linux/aarch64 discovery/setup/permission and CLI suites passed in an isolated Docker runtime. Independent checks preserved UID 1001, GID 1002, mode 0640, and the exact POSIX ACL; capability loss failed safely. The original clock-collision probe passed. A stronger independent probe tested 200 ACL revocations, including 79 equal-ctime cases before Apply and 58 during Sync. Every stale write failed, retaining the revoked grant, original bytes, and inode.

Black-box CLI checks passed for invalid-input/dry-run/no-op immutability, group/ACL preservation, YAML tags and unrelated alias values, personal key/alias identity, exact body bytes, and reduced source authorization. The permission and sidecar documentation at `f880dc3` matches this behavior. Earlier docs/trial evidence remains separately pinned.

Current-reader contexts for tickets 05/06 were complete without diagnostics, with explicit bundle/source roots and limits 500 files/8388608 bytes. Full selected texts matched previously delivered requirements.

Evidence is retained in `/private/tmp/ctx-spec-setup-c59d12e-e3jqn0tc/`: `native-macos-runtime.json`, `native-linux-runtime.json`, `native-linux-cli-runtime.json`, `cli-permission-recheck.json`, `preservation-evidence.json`, and `final-integrity-and-results.json`, plus full contexts, exact commands, image/binary/source/probe hashes, and probe sources. Original red evidence is unchanged. Windows and other platforms were not runtime-tested here; actual alternate ownership was verified on Linux, not macOS.
