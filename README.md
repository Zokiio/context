# Evidence for Zokiio/context#47

Screenshots and metrics from real runs of the factory's solo mode, captured on 2026-10-22 and 2026-10-23 against a disposable Go repository. Claude Code (claude-sonnet-5-5) did the agent work, and Kev 0.8B (kev:0.8b-mxfp8 through Ollama) did the decision-model scoring.

This branch holds only these assets so the PR can show them; it is not meant to be merged.

## `ui/`: the board and the two decisions

Screenshots and metrics from a second trial on 2026-10-23, run against the rebuilt web UI in solo mode at 1440×900 and 2x scale, in light and dark. Claude Code did the agent work: `claude-sonnet-5-5` for refining and test writing, and Claude Code's default model, `claude-fable-5-1`, for builds. Kev 0.8B scored the criteria at `--decision-threshold 0.32`. Pad and Truncate were decided by a person through the board. Slugify ran under a saved ask policy and reached `main` without a human decision. `ui/metrics.json` is the project's measured metrics at the end.
