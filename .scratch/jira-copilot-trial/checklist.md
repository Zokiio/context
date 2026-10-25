# Jira and Copilot trial checklist, 26 October 2026

This checklist covers trying `ctx` on a new project whose tracker is a self-hosted Jira Data Center, with tickets selected by project and label, and GitHub Copilot CLI as the agent. On 25 October every command was run on a preparation computer against a local Jira Data Center 10.3.9 container and a throwaway project. The exceptions are marked. The trial itself runs on another computer.

Waymark's factory is not part of this trial. It has no Jira import and no Copilot builder yet.

This repository is public. Keep the trial project's ticket content, Jira URL and trial log in that project, not here.

## Set up the trial computer

You need bash or zsh, Git, Go 1.25 or later, Python 3, curl 7.55 or later, and Copilot CLI signed in.

- [ ] Get this branch and build `ctx` from it. The branch is `dev` at `afe9989` plus this checklist.

```bash
git clone --branch feat/jira-copilot-trial https://github.com/Zokiio/context.git ~/code/waymark
```

```bash
mkdir -p ~/bin && go -C ~/code/waymark build -o ~/bin/ctx ./cmd/ctx
```

```bash
~/bin/ctx version
```

  The output names this branch's commit and should say `modified: false`.
- [ ] **Always call it as `~/bin/ctx`.** The installed skills call the binary by its absolute path. If `which ctx` finds an older copy, a bare `ctx` runs that copy instead.

## Before you start

- [ ] **Jira version.** Check Help → About Jira. Personal access tokens need 8.14 or later.
- [ ] **Personal access token.** Create one under Profile → Personal Access Tokens, with a short expiry.
- [ ] **Filter and field names.** Note the project key, the label you filter on, and the exact name of the acceptance-criteria field. It is usually a custom field, and could also be a section of the description.

## Prepare the project

Work on a branch in the project's repository.

- [ ] Run init from the repository root:

```bash
~/bin/ctx init --title 'PROJECT' --tracker 'Jira KEY at https://jira.example.com' --skills-dir .github/skills --docs-dir docs/agents/ctx --allow-source docs
```

- [ ] Paste the pointer block it prints into `.github/copilot-instructions.md`, or into `AGENTS.md`.
- [ ] Confirm Copilot sees the guidance:

```bash
copilot --no-auto-update instruction list
```

```bash
copilot --no-auto-update skill list
```

- [ ] **Decide where snapshots live before the first one.** A snapshot inside `.context/records` gets committed. The installed `tracker.md` says to keep disposable snapshots out of Git. For a shared work repository, either commit the snapshots on your branch only, or keep them in a separate records directory passed with `--records`.

Init writes absolute paths into the installed skills, and it adds rules to `.gitignore`. Review both before committing.

## Find and fetch tickets

Load the token without echoing it, from the project root:

```bash
read -rs JIRA_PAT; export JIRA_PAT JIRA_BASE=https://jira.example.com
```

```bash
H() { printf 'Authorization: Bearer %s\n' "$JIRA_PAT"; }; D=.context/trial/jira; mkdir -p $D
```

List the candidate tickets. This loop pages through all results:

```bash
s=0; while :; do curl -fsS -H @<(H) -G "$JIRA_BASE/rest/api/2/search" --data-urlencode 'jql=project = KEY AND labels = "your-label" AND statusCategory != Done ORDER BY key' -d fields=summary,status,updated -d startAt=$s -d maxResults=50 -o $D/search-$s.json || break; s=$(python3 -c 'import json,sys;d=json.load(open(sys.argv[1]));n=d["startAt"]+len(d["issues"]);print(n if d["issues"] and n<d["total"] else "")' $D/search-$s.json); [ -n "$s" ] || break; done
```

Fetch one ticket and record when you fetched it:

```bash
K=KEY-123; date -u +%Y-%m-%dT%H:%M:%S+00:00 > $D/$K.snapshotAt; curl -fsS -H @<(H) -G "$JIRA_BASE/rest/api/2/issue/$K" -d expand=names,renderedFields -o $D/$K.json
```

Print the fields the snapshot needs:

```bash
python3 -I ~/code/waymark/.scratch/jira-copilot-trial/snap.py $D/$K.json
```

What the container test showed:
- **Label filter.** The JQL returned exactly the open, labelled tickets in the one project. It excluded an unlabelled ticket, a done ticket, and a labelled ticket in another project. Quoted and unquoted `agent-ready` behaved the same. Quote labels anyway.
- **A missing or expired token gives a misleading error.** The search returns 400 with "The value 'KEY' does not exist for the field 'project'", not 401. Check the token first when the project "does not exist".
- **Blocker direction.** `snap.py` finds blockers by reading `issuelinks` from the ticket's own side, and that reports "is blocked by" correctly. Creating links through the API is easy to get backwards, but you only read them.
- **Acceptance criteria.** A custom field shows up by name through `expand=names`. In the test it was `customfield_10112`. Yours will have a different id.
- **`/rest/api/2/search` is the right endpoint on Data Center.** Cloud's newer `search/jql` does not exist there.
- **Corporate networks.** The test Jira ran on plain HTTP on localhost. If your Jira uses an internal certificate authority that curl doesn't trust, point curl at it with `--cacert`. Don't use `-k`.

Atlassian's MCP server and `acli` serve Cloud only. The community `mcp-atlassian` server supports Data Center but was not evaluated. Use curl, because it keeps the raw response that the snapshot convention requires.

## Write the snapshot

Create `.context/records/jira/KEY-123.md`. Generate the `id` once with `uuidgen` and keep it on every refresh.

```markdown
---
type: WorkItem
id: <uuidgen>
title: "KEY-123: <summary>"
triage: ready-for-agent
sourceURL: https://jira.example.com/browse/KEY-123
sourceUpdatedAt: "2026-10-25T22:25:18.892+00:00"
snapshotAt: "2026-10-25T22:26:04+00:00"
---

# KEY-123: <summary>

## Source description

<description, converted from wiki markup without rewording>

## Acceptance criteria

- <copied from the AC field>

## Blocked by

None

## Blocked by decisions

None

## Local adaptations

- Raw capture: `.context/trial/jira/KEY-123.json`.
- `triage: ready-for-agent` maps the Jira label `<your-label>`.
- `execution` is omitted; Jira status does not establish local execution.
```

Rules the rehearsal hit:
- **Leave `execution` out.** Snapshots then never reach orient's eligible shortlist, so pick the ticket by path. Don't invent `unstarted` to make it show up.
- **Blockers.** If the blocker is another ticket you also snapshot, link the local file, for example `- [KEY-122](./KEY-122.md)`. `ctx context` then reads complete, exit 0, with the blocker included. A link to a remote Jira URL leaves context incomplete (exit 1), and the task-context skill then tells the agent to stop. For the first run, pick a ticket with no open blockers, or snapshot its blocker too.
- **Convert by hand.** Jira's `updated` is written `+0000`, so write it as `+00:00`. Description wiki markup (`h3.`, `*`, `{{code}}`) has to be converted to Markdown.
- **The ticket path is relative to the records directory:**

```bash
~/bin/ctx context --ticket jira/KEY-123.md
```

- [ ] Add the ticket to Current commitments in `.context/records/project.md`.
- [ ] Run `~/bin/ctx orient`. Diagnostics such as `execution_unknown` and `readiness_unknown` are expected for snapshots.

## Run Copilot

- [ ] **Run Copilot interactively from the project root.** When it asks to run the `ctx` binary, approve it for the session. Copilot also appends `; echo "exit=$?"` to its commands, which may need its own approval.
- [ ] Give it the ticket path, for example "Follow this project's instructions and implement `jira/KEY-123.md`."
- [ ] **Check the model.** Copilot silently fell back from an unavailable configured model in the rehearsal. Check it with `/model`.

Non-interactive mode, `copilot -p` with Copilot CLI 1.0.96-2, could not run `ctx` with narrow allow rules. Both a path rule (`shell(<absolute path>:*)`) and a name rule (`shell(ctx:*)`) were refused, and Copilot's help says `--allow-all-tools` is required for non-interactive mode. Copilot was honest about the refusals. In the rehearsal it stopped without a summary. In the second test it read the record files directly and said its summary wasn't reader output. Use `--allow-all-tools` only if you accept that it can run any command.

## What to record

Add a row to a trial log for each friction point, as in the [Mukabi trial](../ctx-init/mukabi-trial.md):

| What to record | Why it matters |
| --- | --- |
| Minutes from ticket chosen to snapshot that `ctx context` reads | Evidence for the intake plan's Jira Data Center import ticket |
| Hand edits per snapshot: wiki markup, offsets, blockers | What an importer must do |
| Whether Copilot used the ctx skills and ran `ctx`, and how many approvals it asked for | Evidence for the harnesses plan's Copilot builder ticket |
| Where Copilot diverged from the ticket's criteria | Whether the snapshot carried enough context |
| Every `ctx` diagnostic you didn't expect | Reader fixes |

## How the commands were tested

The preparation computer ran Jira Software Data Center 10.3.9 from the official `atlassian/jira-software` image, on PostgreSQL 15, with Atlassian's three-hour test licence. Jira 10 offers no built-in evaluation database, so it needs an external one. The setup had:
- two projects;
- three tickets labelled `agent-ready`, one of them done, and one unlabelled ticket;
- a labelled ticket in the second project, to check the project filter;
- a custom Acceptance Criteria field and a "Blocks" link.

The commands ran with a personal access token. Two fetched tickets became snapshots, and `ctx context` read them complete. None of this setup is needed on the trial computer.
