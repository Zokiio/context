<!-- Generated from internal/initialization/templates/orchestrate-verification.md. Run go generate ./internal/initialization after editing the template. -->

# Verify a checkpoint independently

Use these briefs for the implementing agent, runner, two reviewers, and combined-result verifier. They provide the required procedures without another installed testing or review skill.

## Write tests in the implementation loop

The implementing or repairing agent writes the checkpoint's tests itself, at the approved test seams in its checkpoint file. Report a needed seam that the file lacks to the orchestrator before writing tests that depend on it.

Work test-first in vertical slices. Write one failing test and observe it fail for the expected reason. A missing executable or compile failure does not show that an assertion detects the intended defect. Write the least code that passes the test, then start the next slice. Vertical slices replace horizontal slicing, writing all tests first and then all implementation, because bulk tests check imagined behavior. For authored records or guidance, state a focused check and its observable pass condition before the change.

Test public interfaces and caller-visible behavior. Name each test for its behavior, and cover one coherent case per test. Keep assertions independent of private methods, internal call order, and the implementation's algorithm. Take expected values from the specification, known literals, or worked examples. Retain valid regressions that already pass.

Extend the project's existing test files and helpers. Write the fewest cases that distinguish each requirement and its failure cases. Change an existing test's expected behavior only with a reason from its authoritative source. A failure alone does not justify weakening it.

Implement only the agreed checkpoint. Write each new test's failing command, output, revision, and a SHA-256 digest of every file `git status` reports as changed or untracked at that moment to an evidence file in the project's local evidence location, separate from your reasoning, and report its path. List deleted files by path, and copy each failing test file beside the evidence file. The digests and copies identify the observed sources after later slices change the tree. Give a reason for any new test file or harness.

## Run checks and both review axes

Pin the full candidate SHA and the checkpoint's starting revision as the review base before dispatch. Use a checkout at the candidate revision and confirm that its files match that revision.

Retain the exact commands and commit list:

```sh
git merge-base --is-ancestor <base-sha> <target-sha>
git diff <base-sha>...<target-sha>
git log <base-sha>..<target-sha> --oneline
```

Dispatch the independent runner by pointer at the target SHA, with environment constraints. Require the runner to retain each exact command or procedure, result, exit status, and output in its evidence file. Mark checks that cannot run as unverified.

Run the Standards and Specification reviews in separate fresh contexts, in parallel when the environment supports it. Both inspect the complete identified diff and commit list, including tests. Supply relevant rules and requirements by path and digest. Keep implementation reasoning and the other review report out of each dispatch.

Require the runner, each reviewer, and the combined-result verifier to write the full report to a new evidence file in the project's local evidence location. The evidence file records the observer, time, environment, source identities, and evaluated revision. Each returns a summary of about 400 words or fewer. The summary states the verdict and evaluated revision, cites the criterion identifier or Standards rule source for each finding, and names the evidence file.

### Standards review

Give the reviewer the applicable project standards and this "Standards review" section by path and digest. Ask the reviewer to inspect every changed file and report documented violations with the rule's source, affected lines, and concrete impact. Skip rules already enforced by tooling. Project standards override the baseline. Label baseline smells as judgment calls rather than hard violations.

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

Retain the Standards report and verdict separately. Preferences need adoption before they become mandatory.

### Specification review

Dispatch the reviewer by pointer, as the skill describes. Ask the reviewer to report every missing or partial requirement, unrequested behavior, and apparent implementation error. The reviewer also confirms that every requirement has a test at an approved test seam, or a focused check for authored records or guidance; that expected values come from the specification, known literals, or worked examples rather than restating the implementation; and that the implementer's failure evidence shows each new test failing for a reason that matches its requirement. Cite the requirement text and changed code for each finding. Report unavailable required sources as a verification gap.

Retain the Specification report and verdict separately. One axis cannot compensate for an unresolved finding on the other.

## Preserve failures and renew affected verification

Keep the original failed checks and findings in their evidence files, with their evaluated revisions. Write the source-grounded correction to a findings file that names the failed evidence files rather than copying them, and name it in the dispatch to a fresh repair context for one checkpoint. After the repair produces a new revision, renew affected checks and review axes. Keep unchanged valid observations as historical evidence, with their original revision.

Record each remaining failure or uncertainty. A progress label, changed HEAD, or workflow verdict cannot turn a failed gate into a pass. Escalate unavailable required checks. If repairs stop reducing failures or producing new evidence, diagnose the cause or seek a needed decision before another dependent attempt.
