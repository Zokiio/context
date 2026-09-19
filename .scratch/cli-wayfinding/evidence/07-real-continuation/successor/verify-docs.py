import json,subprocess,hashlib,re,pathlib,datetime
root=pathlib.Path.cwd();out=pathlib.Path('/tmp/ctx-doc-successor.803zto');binary=str(out/'ctx')
limit=['--max-files','500','--max-bytes','8388608'];ticket='cli-wayfinding/issues/07-verify-workflow.md'
cases=[('compact',[binary,'orient']+limit),('detail',[binary,'orient','--detail']+limit),('orientation-json',[binary,'orient','--json']+limit),('context-example',[binary,'context','--ticket','project-discovery/issues/06-document-and-trial.md']),('resume-text',[binary,'resume','--ticket',ticket]+limit),('resume-json',[binary,'resume','--ticket',ticket,'--json']+limit),('resume-direct',[binary,'resume','--bundle','.scratch/records','--checkout','.','--ticket',ticket,'--allow-source','.','--json']+limit),('invalid-detail-json',[binary,'orient','--detail','--json']),('invalid-direct-checkout',[binary,'resume','--bundle','.scratch/records','--ticket',ticket])]
results=[]
for name,args in cases:
 p=subprocess.run(args,capture_output=True)
 (out/(name+'.stdout')).write_bytes(p.stdout);(out/(name+'.stderr')).write_bytes(p.stderr)
 r={'name':name,'argv':args,'exit':p.returncode,'stdoutSHA256':hashlib.sha256(p.stdout).hexdigest(),'stderrSHA256':hashlib.sha256(p.stderr).hexdigest()}
 if name in ['orientation-json','context-example','resume-json','resume-direct']:
  d=json.loads(p.stdout);r['complete']=d['complete']
  if name.startswith('resume'):
   r['recoveryStatus']=d['recovery']['status'];r['comparisonComplete']=d['comparison']['complete'];r['taskReadiness']=next(w['readiness'] for w in d['orientation']['workItems'] if w['source'].endswith(ticket))
  assert d['complete']
 expected=2 if name.startswith('invalid-') else 0
 assert p.returncode==expected,(name,p.returncode,p.stderr)
 results.append(r)
files=['readme.md','docs/readers.md','docs/discovery.md','docs/resuming-work.md'];links=[]
for file in files:
 text=(root/file).read_text()
 for target in re.findall(r'\]\(([^)]+)\)',text):
  if '://' in target or target.startswith('#'):continue
  path=target.split('#')[0]
  if not path:continue
  assert (root/file).parent.joinpath(path).exists(),(file,target)
  links.append({'from':file,'target':target})
manifest={str(p.relative_to(root)):hashlib.sha256(p.read_bytes()).hexdigest() for prefix in ['cmd','internal'] for p in sorted((root/prefix).rglob('*.go'))}
for file in ['go.mod','go.sum']+files:manifest[file]=hashlib.sha256((root/file).read_bytes()).hexdigest()
identity={'baseCommit':subprocess.check_output(['git','rev-parse','HEAD']).decode().strip(),'sourceSHA256':manifest}
(out/'source-identity.json').write_text(json.dumps(identity,indent=2)+'\n')
report={'actor':'codex-documentation-successor','time':datetime.datetime.now(datetime.timezone.utc).isoformat(),'cwd':str(root),'commands':results,'links':links,'sourceIdentity':'source-identity.json','limitations':['Documentation examples and link checks only. Repository test/race/vet/build suite and acceptance reassessment belong to coordinator.','Shared reference edits stale prerequisite requirements until reassessed.']}
(out/'documentation-checks.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({'commands':[(r['name'],r['exit']) for r in results],'checkedLinks':len(links),'report':str(out/'documentation-checks.json')},indent=2))
