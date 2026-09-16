import hashlib, json, pathlib, struct, subprocess
ROOT = pathlib.Path(__file__).parent
PROJECT = ROOT / 'project'
PROJECT.mkdir(exist_ok=True)
(PROJECT / 'project.md').write_text('---\ntype: Project\nid: p\ntitle: P\n---\n## Goals\nNone\n## Current commitments\nNone\n## Open decisions\nNone\n')
FRONT = '---\ntype: WorkItem\nid: work\ntitle: Work\ntriage: ready-for-agent\nexecution: unstarted\n---\n'
def digest(tag, parts):
    data = tag.encode() + b'\0' + struct.pack('>Q', len(parts))
    for p in parts:
        p = p.encode()
        data += struct.pack('>Q', len(p)) + p
    return hashlib.sha256(data).hexdigest()
cases = []
def reference(name, use, definition):
    criteria = '## Acceptance criteria\n- '+use+'\n\n'
    cases.append((name, criteria+'## Comments\n'+definition, [criteria,definition], [criteria,definition]))
reference('multiline-label', '[Use][one two]', '[one\n two]: target.md\n')
reference('escaped-label', '[Use][escaped\\]label]', '[escaped\\]label]: target.md\n')
reference('quoted-definition', '[Use][rule]', '> [rule]: target.md\n')
reference('quoted-multiline-definition', '[Use][rule]', '> [rule]:\n>   target.md\n>   "Multi\n>   line"\n')
reference('list-definition', '[Use][rule]', '- [rule]: target.md\n')
reference('list-multiline-definition', '[Use][rule]', '- [rule]:\n    target.md\n    "Multi\n    line"\n')
reference('empty-destination-title', '[Use][rule]', '[rule]: <>\n  "title"\n')
reference('escaped-title-newline', '[Use][rule]', '[rule]: target.md\n  "Escaped \\"\n  title"\n')
reference('unicode-label', '[Use][CAFÉ]', '[café]: target.md\n')
criteria = '## Acceptance criteria\r- Works.\r\r'
cases.append(('bare-cr-boundaries', criteria+'## Comments\rNote.\r', [criteria], [criteria]))
criteria = '## Acceptance criteria\n- Works.\n\n'
cases.append(('container-exclusion', criteria+'> ## Comments\n> Note.\n\n## Scope\nRetained.\n', [criteria+'## Scope\nRetained.\n'], [criteria]))
results = []
for name, body, ticket_parts, criteria_parts in cases:
    (PROJECT/'work.md').write_bytes((FRONT+body).encode())
    run = subprocess.run([str(ROOT/'ctx'),'orient','--project',str(PROJECT),'--json'],capture_output=True,text=True)
    try:
        work = json.loads(run.stdout)['workItems'][0]
        got = [work.get('ticketSHA256'), work.get('criteriaSHA256')]
    except Exception:
        got = None
    expected = [digest('ctx.ticket-body.v1', ticket_parts), digest('ctx.acceptance-criteria.v1', criteria_parts)]
    results.append(dict(name=name, body=body, exitStatus=run.returncode, got=got, expected=expected, match=got==expected, stderr=run.stderr))
(ROOT/'results.json').write_text(json.dumps(results,indent=2))
print(json.dumps([r for r in results if not r['match']],indent=2))
