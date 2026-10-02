{{if .Development}}<!-- Generated from internal/initialization/templates/cli.md. Run go generate ./internal/initialization after editing the template. -->

{{end -}}
# Use ctx for project work

Use the CLI when selecting work, reporting project status, implementing a known ticket, or continuing work. These readers supply current records and evaluations.

{{if .Development}}Build the current reader in a unique temporary directory from this checkout:

```sh
ctx_agent_repo=$(git rev-parse --show-toplevel)
ctx_agent_bin_dir=$(mktemp -d)
go -C "$ctx_agent_repo" build -o "$ctx_agent_bin_dir/ctx" ./cmd/ctx
```

Keep these variables in the same shell for the commands below. Direct bundle access works without a saved project binding.
{{else}}Use the supplied binary at `{{.Command}}` and this installation's project binding, `{{.Scope}}`. The binding selects `{{.Records}}` and its authorized source roots. These commands work from another directory without a ctx source checkout.
{{end}}
Preserve narrower scope supplied by the caller. Inspect the command's diagnostics when a required source is unavailable.

## Select work or report project status

Run orientation before recommending the next work item. Read goals, current commitments, work readiness, unresolved decisions, and acceptance limits:

```sh
{{if .Development}}"$ctx_agent_bin_dir/ctx" orient --bundle "$ctx_agent_repo/.scratch/records" --allow-source "$ctx_agent_repo"{{else}}{{.Command}} orient {{.Scope}}{{end}} --json
```

The report describes recorded project state. Check the authoritative [tracker]({{.TrackerLink}}) and source freshness before relying on it. Readiness does not authorize implementation.

## Implement or verify a known ticket

Read [task-context]({{.TaskContextLink}}), then collect the ticket with the same scope:

```sh
{{if .Development}}"$ctx_agent_bin_dir/ctx" context --bundle "$ctx_agent_repo/.scratch/records" --allow-source "$ctx_agent_repo"{{else}}{{.Command}} context {{.Scope}}{{end}} \
	--ticket <ticket-path-relative-to-records>
```

Read every selected source's actual text, path, and inclusion reasons. Retain the exact JSON used when the task-context or recovery procedure requires it.

## Continue work

Read [recovery-notes]({{.RecoveryNotesLink}}), then inspect current context and recovery candidates:

```sh
{{if .Development}}"$ctx_agent_bin_dir/ctx" resume --bundle "$ctx_agent_repo/.scratch/records" --allow-source "$ctx_agent_repo" --checkout "$ctx_agent_repo"{{else}}{{.Command}} resume {{.Scope}}{{end}} \
	--ticket <ticket-path-relative-to-records> --json
```

Compare retained notes and reported checks with current requirements and code before continuing. An absent note does not prevent inspection of the current ticket.

## Interpret the result

Exit `0` reports a complete read. Exit `1` retains a partial report or incomplete evaluation; inspect its diagnostics and completeness fields. Exit `2` requires correcting the invocation or execution failure. Resolve missing required context before dependent work, while keeping unrelated gaps visible.

All three readers are read-only. A complete report establishes neither permission to start nor acceptance. Follow [Record acceptance]({{.AcceptanceLink}}) for a completion decision. Invoke `$orchestrate` only when the user explicitly requests checkpoint orchestration or approves a plan that invokes it.
