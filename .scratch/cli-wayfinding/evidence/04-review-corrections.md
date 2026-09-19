# Retained checkpoint review corrections

Actor: Codex /root. Date: 2026-09-19.
Tested revision: `2fc3a524a436018bcfd240090d348d28e0dd7b9a`.

Independent review found two defects after initial acceptance. Timestamp offsets now require minutes 00 through 59, including negative offsets. Retained root validation now trims the type field consistently with current orientation, so exact context output for an effective WorkItem is accepted.

Targeted resumption and CLI suites passed with regression coverage for both findings. A fresh build passed all eleven independent checkpoint probes, including the JSON contract and unchanged-file checks. These corrections preserve the other verified criteria from [initial checkpoint verification](04-retained-checkpoint.md).

- [Independent findings](04-spec-review.md)
- [Regression suite output](04-review-fix-tests.txt)
- [Repeated CLI probes](04-review-fix-probes.json)

This new acceptance supersedes the initial decision without rewriting its evidence. The multi-observation restriction remains intentional until ticket 05.
