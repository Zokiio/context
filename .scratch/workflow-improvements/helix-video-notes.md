# AI LABS checkpoint workflow

Research note, 2026-09-29, based on the full English automatic captions. AI LABS describes its own Claude Code recreation of Helix. These observations do not change the [agreed trial design](helix-design.md).

## What the video describes

- Fresh-context agents and an orchestrator skill divide the work. [01:25](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=85s), [02:40](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=160s).
- Separate agents plan tests, write them before implementation, and run them afterward. [04:43](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=283s), [08:43](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=523s).
- A Stop hook prompts continued work. [05:51](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=351s).
- One critic and one fixer replace Shopify's two reviewers. [12:09](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=729s).
- Humans approve the plan and final result. Feedback creates new checkpoints and shared learnings. [13:26](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=806s), [14:00](https://www.youtube.com/watch?v=bBMp5tLxShQ&t=840s).

## Implications for our trial

The user's individual checkpoint files can support deliberate context selection. The orchestrator must supply relevant requirements and dependency contracts without inheriting its whole conversation into the implementing agent. The [design notes](helix-design.md) track the decisions about that responsibility and the context supplied.

An independent test author needs the agreed behavior and constraints. A separate test runner needs the actual test command, code revision, exit status, and output. Distinct agents alone do not prove independent verification. Any change to a test's expected behavior needs reassessment against its source requirement.

Our agreed standards and specification reviews remain separate obligations. A fixer supplies corrections, not a second independent approval. The first trial still asks for human review after each checkpoint.

We have not inspected the hook implementation. The transcript cannot establish whether it validates evidence, trusts agent-written status, or prevents checkpoint advancement before a Stop event. Our [vision](../../docs/vision.md#agent-execution-model) limits enforcement claims to execution paths actually controlled. Unavailable checks and stalled repairs need an explicit unresolved outcome.

Retain feedback with its source and scope under the agreed design. Only adopted guidance becomes a broader requirement. Select relevant observations for each agent so retained feedback does not recreate the full-plan context problem.

Combined verification must address behavior that no individual checkpoint establishes. The [design notes](helix-design.md) record the agreed responsibility for that review.
