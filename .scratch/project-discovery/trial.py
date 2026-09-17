#!/usr/bin/env python3
"""Exercise a built ctx against this checkout and disposable independent projects.

Usage: python3 .scratch/project-discovery/trial.py BINARY BUILD_EVIDENCE NEW_OUTPUT
The output must not exist. Setup connects this checkout to its existing records.
The child processes use a disposable personal home; the caller's registry is untouched.
"""

import datetime
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile


REPO = Path(__file__).resolve().parents[2]
BINARY, BUILD_EVIDENCE, OUTPUT = (Path(value).resolve() for value in sys.argv[1:])
assert not OUTPUT.exists(), "Evidence output already exists; choose a new filename"
BUILD = json.loads(BUILD_EVIDENCE.read_text())
assert BUILD["exitCode"] == 0 and BUILD["sourcesUnchangedDuringRun"]
assert hashlib.sha256(BINARY.read_bytes()).hexdigest() == BUILD["binarySHA256"]
FIXTURE = Path(tempfile.mkdtemp(prefix="discovery-trial-", dir=BINARY.parent)).resolve()
PERSONAL_HOME = FIXTURE / "personal-home"
PERSONAL_HOME.mkdir()
ENVIRONMENT = dict(os.environ)
ENVIRONMENT["HOME"] = str(PERSONAL_HOME)
BIN_DIR = FIXTURE / "bin"
BIN_DIR.mkdir()
(BIN_DIR / "ctx").symlink_to(BINARY)
ENVIRONMENT["PATH"] = str(BIN_DIR) + os.pathsep + ENVIRONMENT["PATH"]
RESULTS = []
STARTED = datetime.datetime.now(datetime.timezone.utc).isoformat()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def program_sources():
    return {path: digest((REPO / path).read_bytes()) for path in BUILD["programSources"]}


def snapshot(roots):
    observed = {}
    for root in roots:
        for directory, folders, files in os.walk(root, followlinks=False):
            folders[:] = [name for name in folders if name != ".git"]
            links = [Path(directory) / name for name in folders if (Path(directory) / name).is_symlink()]
            for path in [Path(directory), *links, *[Path(directory) / name for name in files if name != ".git"]]:
                info = path.lstat()
                item = {"mode": info.st_mode, "uid": info.st_uid, "gid": info.st_gid,
                        "mtimeNS": info.st_mtime_ns, "inode": info.st_ino}
                if stat.S_ISLNK(info.st_mode):
                    item["link"] = os.readlink(path)
                elif stat.S_ISREG(info.st_mode):
                    item["sha256"] = digest(path.read_bytes())
                observed[str(path)] = item
    return observed


def write(path, text):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text)


def config(path, **fields):
    # JSON mappings are valid YAML frontmatter and keep fixture paths literal.
    write(path, "---\n" + json.dumps({"type": "ContextConfig", "version": 1, **fields}, indent=2) + "\n---\n\nTrial declaration.\n")


def project(records, guide, identity="independent-project", title="Independent project"):
    write(records / "project.md", f"---\ntype: Project\nid: {identity}\ntitle: {title}\n---\n\n# {title}\n\n## Goals\nExercise an independent project without Git.\n\n## Current commitments\n- [Read the guide](ticket.md)\n\n## Open decisions\nNone\n")
    relative = os.path.relpath(guide, records)
    write(records / "ticket.md", f"---\ntype: WorkItem\nid: {identity}-task\ntitle: Read the guide\ntriage: ready-for-agent\nexecution: unstarted\n---\n\n## Acceptance criteria\n- [ ] Read the linked guide.\n\n## Blocked by\nNone\n\n## Blocked by decisions\nNone\n\n## Context\n- [Guide]({relative})\n")
    write(guide, "# Independent guide\n\nThis document lives outside the record store.\n")


def run(label, args, cwd, expected=0, *, writes=(), watch=None, command=None):
    roots = watch or [FIXTURE]
    before = snapshot(roots)
    argv = command or [str(BINARY), *map(str, args)]
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    result = subprocess.run(argv, cwd=cwd, env=ENVIRONMENT, input="", capture_output=True, text=True, timeout=30)
    after = snapshot(roots)
    changed = [path for path in sorted(before.keys() | after.keys()) if before.get(path) != after.get(path)]
    changed_files = [path for path in changed if any("sha256" in value or "link" in value for value in (before.get(path, {}), after.get(path, {})))]
    item = {"label": label, "command": argv, "cwd": str(cwd), "startedAt": started, "exitCode": result.returncode,
            "expectedExitCode": expected, "stdoutSHA256": digest(result.stdout.encode()), "stdoutBytes": len(result.stdout.encode()),
            "stderr": result.stderr, "changedPaths": changed, "changedFiles": changed_files}
    try:
        parsed = json.loads(result.stdout)
    except json.JSONDecodeError:
        parsed = None
    if len(result.stdout) < 60000:
        item["stdout"] = result.stdout
    elif parsed is not None:
        item["structuredSummary"] = {key: parsed[key] for key in ("schemaVersion", "complete", "traversalComplete", "inventoryComplete", "project", "diagnostics") if key in parsed}
        item["structuredSummary"]["sourceCount"] = len(parsed.get("sources", []))
        item["structuredSummary"]["workItems"] = [{key: work[key] for key in ("id", "source", "execution", "readiness", "acceptance") if key in work} for work in parsed.get("workItems", []) if "/project-discovery/" in work["source"]]
    RESULTS.append(item)
    assert result.returncode == expected, (label, result.returncode, result.stderr, result.stdout[:1000])
    if writes:
        assert set(changed_files) <= {str(path) for path in writes}, (label, changed_files)
        allowed_paths = {str(path) for target in writes for path in (target, target.parent, target.parent.parent)}
        assert set(changed) <= allowed_paths, (label, "unexpected setup directory changes", changed)
    else:
        assert before == after, (label, "reader or dry-run changed the filesystem", changed)
    item["filesystemCheck"] = "only intended registration and coordination sidecars changed" if writes else "unchanged paths, contents, modes, ownership, inodes, and mtimes"
    item["passed"] = True
    print(label, "PASS", flush=True)
    return result.stdout, result.stderr, parsed


def setup(label, cwd, records, directory, roots, personal=False, alias=None, dry=False, replace=False):
    args = ["setup", "--records", records, "--directory", directory]
    for root in roots:
        args += ["--allow-source", root]
    if personal:
        args += ["--personal"]
    if alias:
        args += ["--alias", alias]
    if dry:
        args += ["--dry-run"]
    if replace:
        args += ["--replace"]
    destination = (PERSONAL_HOME if personal else Path(cwd) / directory) / ".context/config.md"
    destination = destination.resolve()
    personal_lock = (PERSONAL_HOME / ".context/config.md.lock").resolve()
    writes = () if dry else [destination, Path(str(destination) + ".lock"), personal_lock]
    return run(label, args, cwd, writes=writes, watch=[FIXTURE, REPO] if destination.is_relative_to(REPO) else None)


def exercise():
    assert program_sources() == BUILD["programSources"]
    hub = FIXTURE / "workspace"
    checkout = hub / "independent's project $(literal)"
    nested = checkout / "src/nested"
    nested.mkdir(parents=True)
    records, docs = FIXTURE / "separate-records", FIXTURE / "separate-docs"
    project(records, docs / "guide.md")
    assert not (checkout / ".git").exists() and not (records / ".git").exists()
    relative_records = os.path.relpath(records, nested)
    relative_docs = os.path.relpath(docs, nested)
    dry, _, _ = setup("shared setup dry-run from nested cwd", nested, relative_records, "../..", [relative_docs], dry=True)
    proposal = dry.split("Proposed configuration:\n", 1)[1]
    setup("shared setup changes only registration", nested, relative_records, "../..", [relative_docs])
    assert (checkout / ".context/config.md").read_text() == proposal
    setup("identical shared setup is a no-op", nested, relative_records, "../..", [relative_docs])
    assert RESULTS[-1]["changedPaths"] == []
    _, _, report = run("non-Git project routine JSON orientation", ["orient", "--json"], nested)
    assert report["complete"] and report["project"]["id"] == "independent-project"
    human, _, _ = run("non-Git project routine human orientation", ["orient"], nested)
    assert "Independent project" in human
    _, _, context = run("configured roots supply task context", ["context", "--ticket", "ticket.md"], nested)
    assert context["complete"] and context["traversalComplete"] and any(source["path"] == str(docs / "guide.md") for source in context["sources"])
    ordinary, _, _ = run("explicit descendant project override", ["orient", "--project", nested, "--json"], FIXTURE)
    explained, explanation, _ = run("scope explanation keeps stdout identical", ["orient", "--project", nested, "--json", "--explain-scope"], FIXTURE)
    assert ordinary == explained and str(checkout / ".context/config.md") in explanation
    setup("personal alias coexists with identical shared binding", FIXTURE, records, checkout, [docs], personal=True, alias="independent")
    _, _, report = run("personal project alias", ["orient", "--project", "@independent", "--json"], FIXTURE)
    assert report["project"]["id"] == "independent-project"
    temporarily_moved = checkout.with_name("checkout temporarily absent")
    checkout.rename(temporarily_moved)
    _, _, report = run("alias works with missing checkout", ["orient", "--project", "@independent", "--json"], FIXTURE)
    assert report["complete"]
    temporarily_moved.rename(checkout)

    portable = FIXTURE / "portable-checkout"
    portable.mkdir()
    project(portable / "records", portable / "docs/guide.md", title="Second checkout of the same project")
    setup("portable shared checkout setup", portable, "records", ".", ["docs"])
    moved = FIXTURE / "relocated/portable-checkout"
    moved.parent.mkdir()
    portable.rename(moved)
    _, _, report = run("moved shared checkout discovers relative records", ["orient", "--json"], moved)
    assert report["complete"] and report["project"]["id"] == "independent-project"

    repo_dry, _, _ = setup("repository setup dry-run", REPO, ".scratch/records", ".", ["."], dry=True)
    setup("repository setup changes only registration", REPO, ".scratch/records", ".", ["."])
    assert (REPO / ".context/config.md").read_text() == repo_dry.split("Proposed configuration:\n", 1)[1]
    budgets = ["--max-files", "500", "--max-bytes", "8388608"]
    _, _, report = run("repository routine JSON orientation", ["orient", "--json", *budgets], REPO, watch=[REPO, FIXTURE])
    assert report["complete"] and report["inventoryComplete"] and report["project"]["id"] == "ca6a73e0-ae93-49a3-b287-12071f7446fd"
    human, _, _ = run("repository routine human orientation", ["orient", *budgets], REPO, watch=[REPO, FIXTURE])
    assert "Project and context management" in human
    _, _, context = run("repository task context without selector", ["context", "--ticket", "project-discovery/issues/06-document-and-trial.md"], REPO, watch=[REPO, FIXTURE])
    assert context["complete"] and context["traversalComplete"]
    _, _, limited = run("repository default budgets remain bounded", ["orient", "--json"], REPO, expected=1, watch=[REPO, FIXTURE])
    assert not limited["complete"] and any("limit" in diagnostic["code"] for diagnostic in limited["diagnostics"])

    manifestless = FIXTURE / "manifestless"
    write(manifestless / "ticket.md", "# Manifestless task\n")
    members = [
        {"key": "repo", "title": "This repository", "records": str(REPO / ".scratch/records"), "directory": str(REPO), "allowSources": [str(REPO)]},
        {"key": "independent", "records": str(records), "directory": str(checkout), "allowSources": [str(docs)]},
        {"key": "same-id-checkout", "records": str(moved / "records"), "directory": str(moved), "allowSources": [str(moved / "docs")]},
        {"key": "unavailable", "records": str(FIXTURE / "absent-records")},
        {"key": "absent-checkout", "records": str(records), "directory": str(FIXTURE / "absent-checkout"), "allowSources": [str(docs)]},
        {"key": "manifestless-available", "records": str(manifestless)},
    ]
    config(hub / ".context/config.md", workspace={"id": "trial-workspace", "title": "Trial workspace", "members": members})
    _, _, navigation = run("workspace JSON membership", ["orient", "--json", "--allow-source", FIXTURE / "not-inherited"], hub)
    assert navigation["kind"] == "workspace-navigation" and navigation["schemaVersion"] == 1 and navigation["complete"] and navigation["workStatus"] == "unevaluated"
    assert [member["key"] for member in navigation["members"]] == [member["key"] for member in members]
    assert [member["availability"] for member in navigation["members"]] == ["available", "available", "available", "unavailable", "available", "available"]
    for authored, returned in zip(members, navigation["members"]):
        wanted = ["ctx", "orient", "--bundle", authored["records"]]
        for root in authored.get("allowSources", []):
            wanted += ["--allow-source", root]
        assert returned["selectionArgs"] == wanted
    human, _, _ = run("workspace human membership", ["orient"], hub)
    assert "Work status was not evaluated" in human
    copied = [line.split("select: ", 1)[1] for line in human.splitlines() if "select: " in line][1]
    _, _, selected = run("copied POSIX member command", [], hub, command=[shutil.which("sh"), "-c", copied + " --json"])
    assert selected["complete"] and selected["project"]["id"] == "independent-project"
    _, _, selected = run("structured workspace member arguments", [*navigation["members"][2]["selectionArgs"][1:], "--json"], hub)
    assert selected["complete"] and selected["project"]["id"] == "independent-project"
    _, _, selected = run("available member can have invalid project report", [*navigation["members"][5]["selectionArgs"][1:], "--json"], hub, expected=1)
    assert not selected["complete"] and selected["project"] is None
    _, error, _ = run("context never picks a workspace member", ["context", "--ticket", "ticket.md"], hub, expected=2)
    assert "requires project scope" in error
    _, _, selected = run("project scope wins below enclosing workspace", ["orient", "--json"], nested)
    assert selected["project"]["id"] == "independent-project"
    write(checkout / "project.md", "Broken reserved Project marker.\n")
    _, _, selected = run("explicit workspace ignores malformed project marker", ["orient", "--workspace", nested, "--json"], FIXTURE)
    assert selected["kind"] == "workspace-navigation"
    (checkout / "project.md").unlink()
    _, _, selected = run("explicit project overrides cwd workspace", ["orient", "--project", REPO, "--json", *budgets], hub, watch=[FIXTURE, REPO])
    assert selected["project"]["id"] == "ca6a73e0-ae93-49a3-b287-12071f7446fd"
    registry = PERSONAL_HOME / ".context/config.md"
    old_registry = registry.read_text()
    # A workspace alias without a directory creates no implicit binding.
    write(registry, old_registry.replace("---\n", "---\nworkspaces:\n  - key: alias-only\n    alias: empty\n    id: empty-workspace\n    title: Empty workspace\n    members: []\n", 1))
    _, _, empty = run("workspace alias without cwd binding", ["orient", "--workspace", "@empty", "--json"], FIXTURE)
    assert empty["complete"] and empty["members"] == []

    no_scope = FIXTURE / "unconfigured"
    no_scope.mkdir()
    _, error, _ = run("missing scope suggests setup or direct access", ["orient"], no_scope, expected=2)
    assert "ctx setup" in error and "--bundle" in error
    run("manifestless project selector has no raw fallback", ["context", "--project", manifestless, "--ticket", "ticket.md"], FIXTURE, expected=2)
    _, _, direct = run("manifestless migration through bundle", ["context", "--bundle", manifestless, "--ticket", "ticket.md"], FIXTURE)
    assert direct["complete"] and direct["traversalComplete"]
    malformed = FIXTURE / "malformed"
    write(malformed / ".context/config.md", "---\ntype: ContextConfig\nversion: [bad]\n---\n")
    _, error, _ = run("malformed local configuration", ["orient"], malformed, expected=2)
    assert str(malformed / ".context/config.md") in error
    broken = FIXTURE / "broken-parent"
    config(broken / ".context/config.md", project={"records": str(records), "allowSources": [str(docs)]})
    inner = broken / "inner"
    config(inner / ".context/config.md", project={"records": "../absent-records"})
    _, error, _ = run("broken nearest mapping never falls back", ["orient"], inner, expected=2)
    assert str(inner / ".context/config.md") in error and "project.records" in error
    _, _, direct = run("bundle recovers past broken mapping", ["orient", "--bundle", records, "--allow-source", docs, "--json"], inner)
    assert direct["complete"]
    write(registry, "---\ntype: ContextConfig\nversion: wrong\n---\n")
    _, error, _ = run("malformed personal registry blocks discovery", ["orient"], nested, expected=2)
    assert str(registry) in error
    _, _, direct = run("bundle bypasses malformed personal and local configuration", ["context", "--bundle", records, "--allow-source", docs, "--ticket", "ticket.md"], malformed)
    assert direct["complete"]
    assert program_sources() == BUILD["programSources"]


failure = None
try:
    exercise()
except Exception as error:
    failure = repr(error)
    raise
finally:
    evidence = {"actor": "Codex coordinating agent /root", "startedAt": STARTED,
                "finishedAt": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                "testedRevision": BUILD["testedRevision"], "programSources": BUILD["programSources"],
                "binarySHA256": BUILD["binarySHA256"], "trialScriptSHA256": digest(Path(__file__).read_bytes()),
                "buildEvidence": str(BUILD_EVIDENCE), "buildEvidenceSHA256": digest(BUILD_EVIDENCE.read_bytes()),
                "fixture": str(FIXTURE), "repository": str(REPO), "childPersonalHome": str(PERSONAL_HOME),
                "callerPersonalRegistryUsed": False, "passed": failure is None, "failure": failure,
                "results": RESULTS, "criteria": []}
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(json.dumps(evidence, indent=2) + "\n")
