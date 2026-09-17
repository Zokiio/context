# PR 7 README recheck

Reviewed the uncommitted `readme.md` patch over `5c2148234dbd9ddd653ccda73b089cb9348a2967` in `discovery-reader-cli`. The inspected diff identified the README blob as `ad16c45`. Only the README was modified. This addendum preserves the original standards report unchanged.

The original P3 selector-migration finding is resolved by the patch. Direct examples use `--bundle`; the text distinguishes cwd discovery, directory selectors, aliases, and manifestless direct access.

Two P3 wording issues remain in this reviewed version:

- `readme.md:65` says repeated `--explain-scope` returns exit status 2. That flag has no repetition check. A real invocation with the flag twice returned exit 0 and complete orientation JSON. Remove it from the once-only list while retaining the sentence about stderr output.
- `readme.md:234` still says, "Reader commands do not use it yet," about `internal/discovery`. PR 7 calls `discovery.Resolve` from `resolveReadScope`. Describe the package as parsing configuration and resolving project and workspace scope. The adjacent orientation description can say "record inventory and project evaluation" to distinguish its responsibility.

These are reference inaccuracies under the technical-writing skill's requirement to "state facts, options, limits, and errors." They do not require source changes.

Checked `git rev-parse HEAD`, `git status --short`, `git diff -- readme.md`, the CLI command and scope implementations, and README selector and package descriptions. Ran this read-only probe:

```sh
go run ./cmd/ctx orient --bundle .scratch/records --allow-source . --max-files 500 --max-bytes 8388608 --json --explain-scope --explain-scope
```

The probe exited 0, emitted one scope explanation on stderr, and returned `complete: true`. No tracked files were edited by this reviewer. The setup concurrency and permission findings are outside this recheck.
