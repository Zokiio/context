{{if .Development}}<!-- Generated from internal/initialization/templates/cli.md. Run go generate ./internal/initialization after editing the template. -->

{{end -}}
# Use ctx for project work

Use the CLI when selecting work, reporting project status, implementing a known ticket, or continuing work. The reader commands supply current records and evaluations. For an assigned factory job, use the service commands below to inspect execution and request authorized actions.

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

## Work on an assigned factory job

When the user supplies a factory job or asks for queue inspection, cancellation, repair, review, or acceptance, use `ctx factory`. Use the operator's service URL, private token file, and job ID. These are separate from the local project binding used by the readers.

Inspect the service command help and the assigned job:

```sh
{{if .Development}}"$ctx_agent_bin_dir/ctx"{{else}}{{.Command}}{{end}} factory --help
{{if .Development}}"$ctx_agent_bin_dir/ctx"{{else}}{{.Command}}{{end}} factory --url "$wm_factory_url" --token-file "$wm_factory_token_file" \
	jobs show --job "$wm_factory_job"
```

Read the admitted source text, candidate, required checks, attempts, events, reviews, and remaining actions. Use the supplied role's credential. For each authorized mutation, retain a stable `--request-id`. Use the current `--if-version`, candidate revision, and `admission.ticketSHA256` when a command requires them. Inspect the JSON and stderr when a command returns nonzero.

Run `factory worker` only for an assigned worker role with a durable receipt directory. Implementation and verification use distinct worker identities. Verification executes the administrator's required checks against the assigned candidate. Review and acceptance remain separate authorized decisions attributed to the authenticated account. An account with kind `human` can be operated by an agent; that category does not prove a person's approval.

If a receipt or service outcome is `unknown`, preserve the receipts and workspace, then inspect the exact attempt before requesting reconciliation. A missed heartbeat or cancellation acknowledgement does not establish that external execution stopped.

{{if .Development}}Follow [Run a local factory with offline workers](../factory-platform.md) for service setup, user grants, the complete fixture flow, and recovery.{{else}}The operator configures the service, project policy, executable profiles, and project access. Use `factory <command> --help` for the installed command's flags.{{end}}

## Interpret reader results

Exit `0` reports a complete read. Exit `1` retains a partial report or incomplete evaluation; inspect its diagnostics and completeness fields. Exit `2` requires correcting the invocation or execution failure. Resolve missing required context before dependent work, while keeping unrelated gaps visible.

All three readers are read-only. A complete report establishes neither permission to start nor acceptance. Follow [Record acceptance]({{.AcceptanceLink}}) for a completion decision. Invoke `$orchestrate`, or `/orchestrate` in Copilot, only when the user explicitly requests checkpoint orchestration or approves a plan that invokes it.
