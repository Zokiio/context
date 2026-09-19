# Standards review: `7b93da4...88b7e23`

## Findings

1. **Documented-contract violation — `internal/cli/cli.go:89-128`.** The hunk adds `--detail` and routes the no-format case to the new compact renderer:

   ```go
   &urfave.BoolFlag{Name: "detail", Usage: "Write the full text orientation report"},
   ...
   } else if cmd.Bool("detail") {
       renderDetailedOrientation(...)
   } else if err := renderOrientation(...)
   ```

   This conflicts with the checked-in reader contract at `docs/readers.md:16-18`, which lists the single-use flags without `--detail` and says the **default** text report contains backlog, decisions, per-item readiness/eligibility, digests, and relationship reasons. Those facts now require `--detail`, while the compact default intentionally omits several of them. Update the public reference and command examples with the new default/detail split and `--detail` validation in the same change.

2. **Possible Duplicated Code (judgment) — `internal/cli/orientation_compact.go:48-78` and `internal/cli/orientation.go:13-39`.** The new compact renderer repeats the detailed renderer’s project identity, completeness conversion, goals known/none/present switch, authored goal text, source, and references. For example, both contain:

   ```go
   case result.Project == nil || !result.Project.GoalsKnown:
       fmt.Fprintln(..., "Goals: unknown")
   case len(result.Goals) == 0:
       fmt.Fprintln(..., "Goals: none")
   ```

   The change also introduced `completeness` beside the detailed renderer’s identical local `completion` closure. Extract the shared project/goals writers and completeness formatter so future wording or state changes cannot make compact and detail disagree.

## Verification

`go test ./...` passes at pinned HEAD `88b7e23`. No additional non-tooling standards findings were found in the shared-capture changes.
