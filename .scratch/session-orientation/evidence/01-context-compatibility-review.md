# Shared-reader compatibility review

Workflow reviewer: Codex agent `/root/orientation_ticket_audit`, 2026-09-16. This is an automated review, not human approval.

The reviewer compared commit `9666b19` with its parent `5742063`, limiting the review to the task-context wrappers, shared parsing, source authorization and caching, diagnostics, and JSON aliases. No actionable regression was found. Record-source scope is checked before cached document bytes are reused.

The reviewer extracted each commit with `git archive`, built its `./cmd/ctx` executable, and ran both executables against the same temporary project. Each invocation used `context --project <audit-directory>/project --ticket ticket.md`. Both builds succeeded. The selected `doc.md` contained `selected document` followed by one newline.

The probes compared the complete tuple of exit code, parsed JSON stdout, and stderr between revisions. Raw stdout was compared if it was not JSON. The retained reviewer result was `{"cases":11,"differences":[]}`. These comparisons establish compatibility with the baseline, not independent correctness of the baseline's behavior. No test suite was rerun during this review.

## Probe definitions

The following Python string values reconstruct each starting ticket's exact bytes:

```python
cases = {
    "empty_end_heading": "## Spec\n[x](doc.md)\n\n##",
    "empty_middle_heading": "## Spec\n[x](doc.md)\n\n##\n[ignored](absent.md)\n",
    "closed_empty_heading": "## Context\n[x](doc.md)\n\n# #\n[ignored](absent.md)\n",
    "setext": "Spec\n----\n[x](doc.md)\n\nOther\n-----\n[ignored](absent.md)\n",
    "multiline_setext": "Sp\nec\n----\n[x](doc.md)\n",
    "quoted": "> ## Spec\n> [x](doc.md)\n> ## Other\n> [ignored](absent.md)\n",
    "list_nested": "- ## Spec\n  [x](doc.md)\n- ## Other\n  [ignored](absent.md)\n",
    "quoted_repeat": "## Context\n[missing][absent]\n> ## Blocked by\n> [missing][other]\n\n## Context\n[x](doc.md)\n",
    "formatted_headings": "## **Spec**\n[x](doc.md)\n\n## [Context](ignored.md)\n[x](doc.md)\n",
    "reference_conflicts": "## Blocked by\n[x][same]\n## Context\n[y][missing]\n[x][same]\n\n[same]: doc.md\n[same]: absent.md\n",
    "crlf": "## Spec\r\n[x](doc.md)\r\n\r\n##\r\n[ignored](absent.md)\r\n",
}
```

The probe script used an automatically removed temporary directory. Its extracted revisions, binaries, and individual outputs were not retained. This document preserves the reviewer report and exact fixture definitions; the [original ticket verification](01-overview.md) separately retains the full-suite results and tested working-tree manifest.
