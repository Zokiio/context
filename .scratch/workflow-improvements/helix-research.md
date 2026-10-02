# Helix and our agent workflow

Research note, 2026-09-29. These are proposals for discussion. They do not change the product vision, ADRs, or implementation scope.

## What Shopify describes

Helix uses tools and skills to migrate mobile features through small checkpoints. Each checkpoint passes behavior tests, visual comparison, isolated code reviews, and engineer approval by default. Feedback persists, and autonomy is configurable. Its CLI exercises application behavior. Shopify also describes designs and product documents as references for new features. The article does not expose the enforcement implementation. [Shopify, published September 21, 2026](https://shopify.engineering/helix).

## Where the overlap is strongest

My read is that the closest match is our agent workflow. The [implementation skill](../../.agents/skills/implement/SKILL.md) already requires review, corrections, repeated affected checks, and retained acceptance evidence. The [review skill](../../.agents/skills/code-review/SKILL.md) separates standards and specification review into parallel agents. The [acceptance procedure](../../docs/agents/acceptance.md) records criteria evidence, source identity, tested revision, and decision attribution.

The product has a broader responsibility. Our [vision](../../docs/vision.md) centers durable project records, context, readiness, and optional integrations. Agent execution remains separate. The [reader reference](../../docs/readers.md) documents existing `ctx orient` and `ctx context` operations. Their CLI exposes project information rather than application behavior.

One boundary matters when borrowing the approach. [ADR 0003](../../docs/adr/0003-recorded-acceptance-for-readiness.md) permits unchanged historical acceptance to remain valid after HEAD moves. Orientation checks retained requirements and evidence. It does not rerun tests or certify the current application checkout. A future execution gate would need its own observed code identity and check results.

## Practical inspirations to trial

1. Make each slice reviewable on its own. Show its intended outcome, relevant criteria, changed code, and evidence together. Existing work items and evidence documents can support a trial. A slice inside a larger work item must not acquire partial acceptance that satisfies the whole prerequisite. This follows the [acceptance procedure](../../docs/agents/acceptance.md#acceptance-profile).

2. Make the correction loop explicit. For each slice, identify applicable checks, their observed results, and which need another run after a correction. Distinguish a failed check from unavailable evidence or an invalid comparison. Keep these observations separate from work readiness and the acceptance decision. Escalate repeated failures or conflicting reviews with the remaining question. This would make the existing [implementation workflow](../../.agents/skills/implement/SKILL.md) easier to inspect.

3. Derive verification from the agreed outcome independently of the implementation. In the [vision's Wi-Fi example](../../docs/vision.md#first-users-working-scenario), backend queueing and device configuration need different observations. A reference must identify the behavior and environment being compared. Unresolved success semantics require grooming before equivalence testing can settle them. Our existing CLI cannot substitute for a device test interface.

4. Retain feedback with its scope and source. An engineer's correction can be useful for later work without immediately becoming a project rule. Preserve the observation, propose a focused change, and record adoption separately. The existing [workflow discovery](discovery.md#keep-observations-and-commitments-distinct) already proposes this distinction. Its [criterion-to-evidence view](discovery.md#explain-what-the-evidence-establishes) and [continuation trial](discovery.md#make-interrupted-work-cheap-to-resume) provide a natural place to test it.

I would extend that trial with one small implementation slice and an explicit check-and-correction record. A fresh reviewer should be able to identify the intended behavior, the tested code, unresolved checks, and why acceptance was recorded. Measure review effort and repeated investigation before proposing a new record type or agent runtime. The [vision's enforcement boundary](../../docs/vision.md#agent-execution-model) still applies: skill instructions cannot guarantee that an agent follows them.
