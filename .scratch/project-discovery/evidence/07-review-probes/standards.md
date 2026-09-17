# Standards review

Reviewed pinned HEAD `2777f087a5e320594e09c5cda1f35d62e66c5a80` against `1eb54f87242502020181b46504d9556e5bea6a6a`, including PRs #4, #6, #7, #8, #9, and #10 separately.

## Documented standards

No hard violation or concrete correctness defect remains at the pinned HEAD. Scope resolution stays outside the existing readers, setup writes stay in the Go core, and authoritative records retain their original evidence and separately attributed workflow decisions.

One intermediate documentation defect is resolved by #10. **P3, PR #7**, `readme.md:153` at `5c21482` still says that `--project` is the bundle root and that both flags are required with no configuration fallback. `internal/cli/scope.go:26` changes that contract. Following the documented command for a manifestless bundle now fails until the user changes to `--bundle`. This contradicts the direct-access distinction in `docs/adr/0004-directory-specific-project-selection.md`, final paragraph, and the technical-writing skill's requirement that reference text state the actual facts. PRs #8 and #9 retain the mismatch; #10 corrects it. No change is needed at the reviewed tip.

## Judgement calls

**P3, possible Duplicated Code, PR #8.** `internal/workspace/navigation.go:120` repeats the shallow directory probe introduced by #7 at `internal/cli/scope.go:118`: `os.Stat(path)`, `os.OpenRoot(path)`, `root.Open(".")`, and `directory.ReadDir(1)` with the same EOF treatment. A shared probe could keep future filesystem fixes consistent while callers retain their own diagnostics. This is a maintenance suggestion, with no demonstrated behavior failure.

## Checks performed

Used `git rev-parse`, `git log`, combined and adjacent three-dot `git diff` reads, `git show` for intermediate README contents, `rg`, and numbered source reads. Slice boundaries were `1eb54f8 → 18728ee → 39483dd → 5c21482 → 573c220 → 933d9c7 → 2777f08`.

Read-only Python probes checked 44 acceptance snapshot digests and 17 public-document local link targets, with no mismatches or missing targets. Per-slice diff inventories found no edits to previously retained evidence files. `git status --short` was clean. I did not run Go tests or the mutating trial; the coordinator owns runtime validation. No tracked files were changed.
