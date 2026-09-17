import os, pathlib, tempfile, subprocess, json, hashlib, stat
ROOT=pathlib.Path(tempfile.mkdtemp(prefix='preservation-',dir='/private/tmp/ctx-spec-setup-c59d12e-e3jqn0tc'))
for n in ('home','checkout','old-records','new-records','docs-a','docs-b'):(ROOT/n).mkdir()
for n in ('old','new'):
 (ROOT/(n+'-records/project.md')).write_text('---\ntype: Project\nid: '+n+'\ntitle: '+n+'\n---\n')
for n in ('a','b'):(ROOT/('docs-'+n+'/guide.md')).write_text('# Guide '+n+'\n')
(ROOT/'new-records/task.md').write_text('# Task\n\n## Context\n- [A](../docs-a/guide.md)\n- [B](../docs-b/guide.md)\n')
p=ROOT/'home/.context/config.md';p.parent.mkdir()
body='\r\n# Personal registry notes\r\n\r\nPreserve **these bytes**.\r\n'.encode()
p.write_bytes(('---\ntype: ContextConfig\nversion: 1\nprojects:\n  - &selected\n    key: stable-key\n    alias: keep\n    directory: '+str(ROOT/'checkout')+'\n    records: '+str(ROOT/'old-records')+'\n    allowSources: ['+str(ROOT/'docs-a')+', '+str(ROOT/'docs-b')+']\n    custom: !!timestamp 2026-09-17T12:00:00Z\n  - key: other-key\n    alias: occupied\n    directory: '+str(ROOT/'unavailable-checkout')+'\n    records: '+str(ROOT/'unavailable-records')+'\nunrelated: *selected\nworkspaces: [{key: workspace-key, alias: team, id: team, title: Team, members: []}]\nencoding: !!binary SGVsbG8=\n---\n').encode()+body)
p.chmod(0o640)
env=dict(os.environ,HOME=str(ROOT/'home'))
results=[]
def snap():
 return {str(q.relative_to(ROOT)):(q.lstat().st_mode,q.lstat().st_ino,q.lstat().st_mtime_ns,hashlib.sha256(q.read_bytes()).hexdigest() if q.is_file() else None) for q in ROOT.rglob('*')}
def run(label,args,writes=False):
 before=snap()
 c=subprocess.run(['/private/tmp/ctx-spec-setup-c59d12e-e3jqn0tc/ctx',*args],cwd=ROOT/'checkout',env=env,capture_output=True,text=True)
 after=snap();result={'label':label,'args':args,'status':c.returncode,'stdout':c.stdout,'stderr':c.stderr,'changed':[k for k in sorted(before.keys()|after.keys()) if before.get(k)!=after.get(k)]};results.append(result)
 if not writes:assert before==after,result
 return c
args=['setup','--personal','--records',str(ROOT/'new-records'),'--replace','--allow-source',str(ROOT/'docs-b')]
d=run('dry run preserves all paths',args+['--dry-run']);assert d.returncode==0
c=run('alias collision rejected without writes',args+['--alias','occupied']);assert c.returncode==2
c=run('replacement preserves alias key tagged metadata prose and modes',args,True);assert c.returncode==0
assert 'Removes allowed source: '+str(ROOT/'docs-a') in c.stdout
content=p.read_bytes();assert content.endswith(body)
assert '!!timestamp' in content.decode() and '!!binary' in content.decode()
assert p.stat().st_mode&0o777==0o640
assert content.decode().count('stable-key')==2
assert content.decode().count(str(ROOT/'old-records'))==1
assert str(ROOT/'unavailable-records') in content.decode()
c=run('identical binding no-op',args);assert c.returncode==0 and 'unchanged' in c.stdout
c=run('retained alias selects new records with reduced sources',['context','--project','@keep','--ticket','task.md']);assert c.returncode==1
j=json.loads(c.stdout);assert not j['complete']
assert any(d['code']=='source_outside_scope' and 'docs-a' in d['path'] for d in j['diagnostics'])
assert any(s['path']==str(ROOT/'docs-b/guide.md') for s in j['sources'])
pathlib.Path('/private/tmp/ctx-spec-setup-c59d12e-e3jqn0tc/preservation-evidence.json').write_text(json.dumps({'revision':'c59d12e6886874a00d42b821be24615a93c95ef6','root':str(ROOT),'results':results},indent=2)+'\n')
print('Five independent preservation/source-authorization probes passed',ROOT)
