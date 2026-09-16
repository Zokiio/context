import hashlib, json, pathlib, struct, subprocess
ROOT = pathlib.Path(__file__).parent
PROJECT = ROOT/'project'
FRONT = '---\ntype: WorkItem\nid: work\ntitle: Work\ntriage: ready-for-agent\nexecution: unstarted\n---\n'
def digest(tag, parts):
    data = tag.encode()+b'\0'+struct.pack('>Q',len(parts))
    for p in parts:
        p=p.encode()
        data+=struct.pack('>Q',len(p))+p
    return hashlib.sha256(data).hexdigest()
cases=[]
def put(name,criteria,definition):
    cases.append((name,criteria+'## Comments\n'+definition,[criteria,definition],[criteria,definition]))
put('quoted-criteria','> ## Acceptance criteria\n> - [Use][r]\n\n','[r]: target.md\n')
put('list-criteria','- ## Acceptance criteria\n  - [Use][r]\n\n','[r]: target.md\n')
put('multiline-use','## Acceptance criteria\n- [Use][one\n two]\n\n','[one two]: target.md\n')
put('mixed-line-endings','## Acceptance criteria\r\n- [Use][r]\n\r\n','[r]: target.md\r\n  "Title"\n')
first='## Acceptance criteria\n- [Use][r]\n\n[r]: first.md\n\n'
second='## Acceptance criteria\n- [Use][r]\n'
cases.append(('repeated-criteria-retained-definition',first+'## Comments\n[r]: ignored.md\n'+second,[first+second],[first,second]))
results=[]
for name,body,tp,cp in cases:
    (PROJECT/'work.md').write_bytes((FRONT+body).encode())
    run=subprocess.run([str(ROOT/'ctx'),'orient','--project',str(PROJECT),'--json'],text=True,capture_output=True)
    try:
        w=json.loads(run.stdout)['workItems'][0]
        got=[w.get('ticketSHA256'),w.get('criteriaSHA256')]
    except Exception: got=None
    expected=[digest('ctx.ticket-body.v1',tp),digest('ctx.acceptance-criteria.v1',cp)]
    results.append(dict(name=name,body=body,exitStatus=run.returncode,got=got,expected=expected,match=got==expected,stderr=run.stderr))
(ROOT/'boundary-results.json').write_text(json.dumps(results,indent=2))
print(json.dumps({'cases':len(results),'differences':[r for r in results if not r['match']]},indent=2))
