<!-- Generated from internal/initialization/templates/orchestrate-verification.md. Run go generate ./internal/initialization after editing the template. -->

# Verify a checkpoint independently

Use these briefs for the expectation author, runner, and two reviewers. They provide the required procedures without another installed testing or review skill.

## Prepare independent expectations

Give an author separate from implementation the agreed requirement text, source paths and digests, checkpoint outcome, approved seam, starting revision, and applicable constraints. Record existing authorization for the seam. Resolve a genuinely undecided seam before tests that depend on it.

Ask the author to derive observable expectations from requirements before production edits. Expected values come from the specification, known literals, or independently worked examples. Choose executable tests for missing executable behavior and focused procedures for authored records or guidance. Prepare each procedure independently before the change and state the observable pass condition.

When executable tests are appropriate, observe public interfaces and caller-visible behavior. Test names describe the behavior, and each test covers one coherent case. Refactoring internals alone should not break a test. Keep assertions independent of private methods, internal call order, and the implementation's algorithm.

Prepare expectations one checkpoint at a time. For missing executable behavior, establish a targeted failing test before implementation. Retain valid regressions that already pass. A missing executable or compile failure is baseline evidence, but does not show that a behavior assertion detects the intended defect. Preserve the exact command or procedure, output, result, exit when applicable, author, environment, time, code identity, and requirement identities for each baseline.

Supply the cases and reusable tests to the implementing agent after preparation. Implement only the agreed checkpoint. Renew expectations when adopted requirements change. Explain any proposed test correction against its authoritative source. A failure alone does not justify weakening the expected behavior.

## Run checks and both review axes

Pin the full candidate SHA and review base before dispatch. Use a checkout at the candidate revision and confirm that its files match that revision. Include independently authored reusable tests in the complete comparison. If the preparation commit already contains those tests, use a prior review base that includes them. Record that choice.

Retain the exact commands and commit list:

```sh
git merge-base --is-ancestor <base-sha> <target-sha>
git diff <base-sha>...<target-sha>
git log <base-sha>..<target-sha> --oneline
```

Give the independent runner the checkpoint requirements, required checks, target SHA, source identities, and environment constraints. Require the runner to retain each exact command or procedure, result, exit status, output, observer, time, environment, and evaluated revision. Mark checks that cannot run as unverified.

Run the Standards and Specification reviews in separate fresh contexts, in parallel when the environment supports it. Both inspect the complete identified diff and commit list, including tests. Supply actual relevant rules and requirements. Keep implementation reasoning and the other review report out of each dispatch.

### Standards review

Give the reviewer the applicable project standards and the full baseline below. Ask the reviewer to inspect every changed file and report documented violations with the rule's source, affected lines, and concrete impact. Skip rules already enforced by tooling. Project standards override the baseline. Label baseline smells as judgment calls rather than hard violations.

The baseline covers these possible smells and corrections:

- Mysterious Name: a name hides its purpose. Rename it or clarify the design.
- Duplicated Code: the same logic appears in several places. Extract the shared logic.
- Feature Envy: a method uses another object's data more than its own. Move the behavior to its owner.
- Data Clumps: the same fields travel together. Consider a type that holds them.
- Primitive Obsession: a primitive represents a domain concept that needs its own type. Introduce that type when warranted.
- Repeated Switches: the same condition cascade repeats. Centralize the decision or use polymorphism.
- Shotgun Surgery: one change requires scattered edits. Gather the behavior into one module.
- Divergent Change: one module changes for unrelated reasons. Separate those responsibilities.
- Speculative Generality: abstractions support no agreed requirement. Remove them until needed.
- Message Chains: callers navigate through several objects. Hide the navigation behind an owning interface.
- Middle Man: a function mostly delegates without adding useful behavior. Call the actual owner directly.
- Refused Bequest: an implementation ignores most of its inherited contract. Prefer a fitting interface or composition.

Retain the Standards verdict separately. Preferences need adoption before they become mandatory.

### Specification review

Give the reviewer the checkpoint requirements, acceptance criteria, source identities, and dependency contract. Ask the reviewer to report every missing or partial requirement, unrequested behavior, and apparent implementation error. Cite the requirement text and changed code for each finding. Report unavailable required sources as a verification gap.

Retain the Specification verdict separately. One axis cannot compensate for an unresolved finding on the other.

## Preserve failures and renew affected verification

Keep the original failed checks and findings with their evaluated revisions. Send the relevant evidence and source-grounded correction to a fresh repair context for one checkpoint. After the repair produces a new revision, renew affected checks and review axes. Keep unchanged valid observations as historical evidence, with their original revision.

Record each remaining failure or uncertainty. A progress label, changed HEAD, or workflow verdict cannot turn a failed gate into a pass. Escalate unavailable required checks. If repairs stop reducing failures or producing new evidence, diagnose the cause or seek a needed decision before another dependent attempt.
