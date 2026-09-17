import hashlib,json,os,pathlib,stat,subprocess,tempfile
base=pathlib.Path(__file__).resolve().parent.parent
root=pathlib.Path(tempfile.mkdtemp(prefix='cli-permissions-',dir=base))
for name in ['checkout','home','records','docs']:(root/name).mkdir()
(root/'records/project.md').write_text('---\ntype: Project\nid: probe-project\ntitle: Probe project\n---\n')
path=root/'checkout/.context/config.md';path.parent.mkdir();path.write_bytes(b'---\ntype: ContextConfig\nversion: 1\nproject: {records: ../../records}\n---\nRetained **body**.\r\n')
path.chmod(0o640)
old=path.stat();groups=[gid for gid in os.getgroups() if gid!=old.st_gid]
assert groups,'Need different permitted group for original regression'
os.chown(path,-1,groups[0]);subprocess.run(['chmod','+a','everyone deny write',str(path)],check=True)
env=dict(os.environ,HOME=str(root/'home'))
def acl(p):return subprocess.run(['ls','-le',str(p)],text=True,capture_output=True,check=True).stdout.split('\n',1)[1]
def perms():
 s=path.stat();return {'uid':s.st_uid,'gid':s.st_gid,'mode':oct(stat.S_IMODE(s.st_mode)),'acl':acl(path)}
def snapshot():
 state={}
 for p in root.rglob('*'):
  s=p.lstat();state[str(p.relative_to(root))]={'mode':s.st_mode,'inode':s.st_ino,'mtime':s.st_mtime_ns,'ctime':s.st_ctime_ns,'uid':s.st_uid,'gid':s.st_gid,'sha256':hashlib.sha256(p.read_bytes()).hexdigest() if p.is_file() else None}
 return state
before=perms();results=[]
args=[str(base/'ctx'),'setup','--records',str(root/'records'),'--allow-source',str(root/'docs'),'--replace']
for label,extra,writes,wanted in [('invalid flags',['--alias','invalid'],False,2),('dry-run',['--dry-run'],False,0),('apply',[],True,0),('no-op',[],False,0)]:
 prior=snapshot();command=args+extra;r=subprocess.run(command,cwd=root/'checkout',env=env,text=True,capture_output=True);after=snapshot()
 result={'label':label,'command':command,'status':r.returncode,'stdout':r.stdout,'stderr':r.stderr,'changed':[p for p in sorted(prior.keys()|after.keys()) if prior.get(p)!=after.get(p)]};results.append(result)
 assert r.returncode==wanted,result
 if not writes:assert prior==after,result
 if label=='no-op':assert 'unchanged' in r.stdout,result
 assert perms()==before,(before,perms(),result)
assert path.read_bytes().endswith(b'Retained **body**.\r\n')
assert not (root/'home/.context/config.md').exists()
assert (root/'home/.context/config.md.lock').is_file()
assert (root/'checkout/.context/config.md.lock').is_file()
(base/'cli-permission-recheck.json').write_text(json.dumps({'revision':'c59d12e6886874a00d42b821be24615a93c95ef6','fixture':str(root),'before':before,'after':perms(),'results':results},indent=2)+'\n')
print('PASS: CLI invalid, dry-run, replacement, and no-op preserve metadata; uid/gid/mode/ACL:',perms())
