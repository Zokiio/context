from pathlib import Path
import tempfile,subprocess,json,os,hashlib,shlex
B='/private/tmp/ctx-spec-readers-review'
R=Path(tempfile.mkdtemp(prefix='ctx-independent-review-',dir='/private/tmp'))
H=R/'home';H.mkdir();env={**os.environ,'HOME':str(H)}
results=[]
def put(p,t):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(t)
def cfg(p,d):put(p,'---\n'+json.dumps({'type':'ContextConfig','version':1,**d})+'\n---\nKept prose\n')
def bundle(p,id='project-a',doc=None):
 put(p/'project.md',f'---\ntype: Project\nid: {id}\ntitle: {id}\n---\n\n## Goals\n\nNone\n\n## Current commitments\n\nNone\n\n## Open decisions\n\nNone\n')
 put(p/'ticket.md','# Ticket\n\n## Acceptance criteria\n\nWorks.\n'+(f'\n## Context\n\n[Doc]({doc})\n' if doc else ''))
 return p

def run(label,args,cwd=None,want=0,check=None):
 p=subprocess.run([B,*args],cwd=cwd or R,env=env,text=True,capture_output=True,timeout=10)
 try:d=json.loads(p.stdout)
 except:d=None
 ok=p.returncode==want and (check(d,p) if check else True)
 results.append({'label':label,'ok':ok,'exit':p.returncode,'args':args,'cwd':str(cwd or R),'stdout':p.stdout,'stderr':p.stderr})
 print(label,'PASS' if ok else 'FAIL',p.returncode, p.stderr[:220] if not ok else '')
 return d,p

D=R/'docs';put(D/'guide.md','Supporting document\n')
A=bundle(R/'records-a',doc='../docs/guide.md');C=R/'checkout';(C/'nested').mkdir(parents=True)
cfg(C/'.context/config.md',{'project':{'records':str(A),'allowSources':[str(D)]}})
d,p=run('configured roots + separate records',['context','--ticket','ticket.md'],C,want=0,check=lambda d,p:len(d['sources'])==2)
e,_=run('explicit path matches cwd',['context','--project',str(C/'nested'),'--ticket','ticket.md'],want=0,check=lambda d,p:d==json.loads(results[-1]['stdout']))
run('bundle excludes configured roots',['context','--bundle',str(A),'--ticket','ticket.md'],C,want=1,check=lambda d,p:any(x['code']=='source_outside_scope' for x in d['diagnostics']))
run('byte budget remains partial',['context','--ticket','ticket.md','--max-bytes','1'],C,want=1,check=lambda d,p:not d['complete'] and not d['traversalComplete'] and d['sources']==[])
run('file budget whole source',['context','--ticket','ticket.md','--max-files','1'],C,want=1,check=lambda d,p:not d['complete'] and len(d['sources'])==1)
run('repeated selector fails',['context','--project',str(C),'--project',str(C),'--ticket','ticket.md'],want=2,check=lambda d,p:p.stdout=='')
run('no scope outside repo',['orient','--json'],want=2,check=lambda d,p:p.stdout=='' and 'filesystem root' in p.stderr and '--bundle' in p.stderr)
M=R/'manifestless';put(M/'ticket.md','# Old ticket\n')
run('manifestless direct context',['context','--bundle',str(M),'--ticket','ticket.md'],want=0)
run('manifestless orientation partial',['orient','--bundle',str(M),'--json'],want=1,check=lambda d,p:d['project'] is None)
run('manifestless project no fallback',['context','--project',str(M),'--ticket','ticket.md'],want=2,check=lambda d,p:p.stdout=='')
# Inner workspace prevents implicit outer project selection.
W=C/'nested';cfg(W/'.context/config.md',{'workspace':{'id':'workspace-a','title':'Workspace A','members':[]}})
run('nearer workspace stops context',['context','--ticket','ticket.md'],W,want=2,check=lambda d,p:'requires project scope' in p.stderr and p.stdout=='')
run('explicit project passes workspace',['context','--project',str(W),'--ticket','ticket.md'],want=0)
run('empty workspace navigation',['orient','--json'],W,want=0,check=lambda d,p:d['kind']=='workspace-navigation' and d['members']==[] and d['workStatus']=='unevaluated')
put(W/'project.md','malformed reserved marker\n')
run('malformed marker no fallback',['orient','--json'],W,want=2,check=lambda d,p:'project marker' in p.stderr and p.stdout=='')
run('workspace ignores malformed marker',['orient','--workspace',str(W),'--json'],want=0)
# Personal bindings at equal depth conflict with authored roots.
cfg(H/'.context/config.md',{'projects':[{'key':'a','alias':'saved','directory':str(C),'records':str(A),'allowSources':[]}]})
run('equal-depth roots conflict',['orient','--json'],C,want=2,check=lambda d,p:'conflicting project' in p.stderr and str(H/'.context/config.md') in p.stderr and str(C/'.context/config.md') in p.stderr)
run('alias concrete entry independent of local',['context','--project','@saved','--ticket','ticket.md'],C,want=1,check=lambda d,p:any(x['code']=='source_outside_scope' for x in d['diagnostics']))
# Missing checkout is independent of alias records, and unselected record targets are lazy.
cfg(H/'.context/config.md',{'projects':[{'key':'a','alias':'saved','directory':str(R/'missing-checkout'),'records':str(A),'allowSources':[str(D)]},{'key':'unrelated','directory':str(R/'unrelated'),'records':str(R/'missing-records')}]})
run('alias missing checkout + unrelated missing records',['context','--project','@saved','--ticket','ticket.md'],want=0)
put(H/'.context/config.md','broken personal config\n')
run('malformed registry blocks discovery',['context','--project',str(C),'--ticket','ticket.md'],want=2,check=lambda d,p:p.stdout=='')
run('bundle bypasses malformed registry',['context','--bundle',str(A),'--allow-source',str(D),'--ticket','ticket.md'],want=0)
(H/'.context/config.md').unlink()
# Physical start and symlinked config use their encountered bases.
physical=R/'physical';(physical/'child').mkdir(parents=True);lexical=R/'lexical';lexical.mkdir();(lexical/'jump').symlink_to(physical/'child',target_is_directory=True)
other=bundle(R/'records-b','project-b');cfg(physical/'.context/config.md',{'project':{'records':str(A),'allowSources':[str(D)]}});cfg(lexical/'.context/config.md',{'project':{'records':str(other)}})
run('symlink parent selects physical ancestry',['orient','--project',str(lexical)+'/jump/..','--json'],want=0,check=lambda d,p:d['project']['id']=='project-a')
external=R/'external';external.mkdir();linked=R/'linked-config';linked.mkdir();localrecords=bundle(linked/'records','encountered-base');bundle(external/'records','wrong-base');cfg(external/'settings.md',{'project':{'records':'../records'}})
(linked/'.context').mkdir();(linked/'.context/config.md').symlink_to(external/'settings.md')
run('symlinked config keeps encountered base',['orient','--project',str(linked),'--json'],want=0,check=lambda d,p:d['project']['id']=='encountered-base')
# Member command preserves whitespace, quote, comma, newline and member roots.
odd=R/"member 'quotes', spaced\nline";bundle(odd,'member-one', '../docs/guide.md')
# Member is shallow-available even with a malformed manifest; copied command remains partial.
shallow=R/'shallow';put(shallow/'project.md','invalid manifest\n')
nav=R/'navigation';nav.mkdir();cfg(nav/'.context/config.md',{'workspace':{'id':'nav','title':'Navigation','members':[{'key':'odd','records':str(odd),'directory':str(R/'absent-checkout'),'allowSources':[str(D)]},{'key':'missing','records':str(R/'absent-records')},{'key':'shallow','records':str(shallow)}]}})
navdata,_=run('workspace ordered shallow availability',['orient','--json'],nav,want=0,check=lambda d,p:d['complete'] and [m['key'] for m in d['members']]==['odd','missing','shallow'] and [m['availability'] for m in d['members']]==['available','unavailable','available'])
run('member array command preserves roots',['context',*navdata['members'][0]['selectionArgs'][2:],'--ticket','ticket.md'],want=0,check=lambda d,p:len(d['sources'])==2)
_,human=run('workspace human output',['orient'],nav,want=0,check=lambda d,p:'Work status was not evaluated.' in p.stdout)
# Human shell text may contain literal newlines inside quotes; parse the select span by known next member.
select=human.stdout.split('    select: ',1)[1].split('\n  missing',1)[0].rstrip('\n')
roundtrip=shlex.split(select)
results.append({'label':'human member POSIX quoted roundtrip','ok':roundtrip==navdata['members'][0]['selectionArgs']})
print('human member POSIX quoted roundtrip','PASS' if results[-1]['ok'] else 'FAIL')
# Changing a selected mapping to missing records must not find the broader project.
inner=C/'broken';inner.mkdir();cfg(inner/'.context/config.md',{'project':{'records':str(R/'absent-records')}})
run('selected missing records no fallback',['orient','--json'],inner,want=2,check=lambda d,p:p.stdout=='' and 'selected directory is unavailable' in p.stderr)
# Manifest marker and agreeing authored mapping gain roots.
cfg(A/'.context/config.md',{'project':{'records':'..','allowSources':[str(D)]}})
run('marker + mapping supplies roots',['context','--project',str(A),'--ticket','ticket.md'],want=0,check=lambda d,p:len(d['sources'])==2)
# Source symlinks may not escape an allowed root.
outside=R/'outside';put(outside/'hidden.md','outside\n');(D/'escape.md').symlink_to(outside/'hidden.md');put(A/'escape.md','# Escape\n\n## Context\n\n[Hidden](../docs/escape.md)\n')
run('source symlink escape rejected',['context','--project',str(C),'--ticket','escape.md'],want=1,check=lambda d,p:any(x['code']=='source_outside_scope' for x in d['diagnostics']))
# Capture byte hashes around read-only commands.
def hashes():return {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in R.rglob('*') if p.is_file()}
before=hashes();run('read only verification',['orient','--workspace',str(nav),'--json'],want=0);after=hashes();results[-1]['ok'] &= before==after
out=Path('/private/tmp/ctx-spec-readers-probes.json');out.write_text(json.dumps({'fixtures':str(R),'results':results},indent=2))
print('SUMMARY',sum(r['ok'] for r in results),'/',len(results),'retained',out,'fixtures',R)
