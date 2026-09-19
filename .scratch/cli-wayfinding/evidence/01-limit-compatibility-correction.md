# Standalone reader compatibility correction

Actor: Codex /root. Date: 2026-09-19.
Tested revision: `3f966925076e07c2cfe46a63981fb729764e8067`.
Baseline binary was built at `c3d37e4`, before milestone product changes.

A later comparison on the expanded real record store exposed repeated file-limit diagnostics for repeated references to an already rejected source. The retained capture bytes needed by composed resumption caused orientation to repeat admission. The fix preserves omission tracking per physical source. A separate entry flag preserves standalone manifest-byte-limit behavior while retaining the composed operation's inventory omission.

Seven actual old/new command comparisons now have byte-identical stdout and stderr and equal exit statuses: task context normally, at one file, and at one byte; orientation at defaults, full explicit limits, one file, and one byte. Commands read the same current files in the main checkout.

Focused orientation, recordread, resumption, and CLI suites passed. The new filesystem regression exercises repeated references to an oversized unadmitted acceptance snapshot and one later pending source. Existing shared-limit and manifest-first tests pass. Independent review found no hidden composed-behavior issue.

- [Suite output](01-compatibility-fix-tests.txt)
- [Seven comparisons](01-final-standalone-compatibility.json)
- [Independent recheck](01-compatibility-recheck.md)

This supplements earlier [shared-source verification](01-share-captured-sources.md) and replaces the current acceptance decision without changing prior evidence.
