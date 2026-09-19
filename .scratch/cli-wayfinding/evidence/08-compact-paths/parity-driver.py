import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

ROOT = Path('/Users/zoki/code/context')
DEMO = ROOT / '.cache/live-cli-demo/demo'
OUT = ROOT / '.scratch/cli-wayfinding/evidence/08-compact-paths'
OUT.mkdir(exist_ok=True)
old = '/tmp/context-compact-before'
new = '/tmp/context-compact-after'
base = ['--bundle', str(ROOT / '.scratch/records'), '--allow-source', str(ROOT), '--max-files', '1000', '--max-bytes', '67108864']
cases = [
    ('demo-detail', DEMO, ['orient', '--detail']),
    ('demo-json', DEMO, ['orient', '--json']),
    ('demo-context', DEMO, ['context', '--ticket', 'resume.md']),
    ('demo-resume', DEMO, ['resume', '--ticket', 'resume.md']),
    ('demo-resume-json', DEMO, ['resume', '--ticket', 'resume.md', '--json']),
    ('repo-detail', ROOT, ['orient', *base, '--detail']),
    ('repo-json', ROOT, ['orient', *base, '--json']),
    ('repo-context', ROOT, ['context', *base, '--ticket', 'cli-wayfinding/issues/08-compact-paths.md']),
    ('repo-resume', ROOT, ['resume', *base, '--checkout', str(ROOT), '--ticket', 'cli-wayfinding/issues/08-compact-paths.md']),
    ('repo-resume-json', ROOT, ['resume', *base, '--checkout', str(ROOT), '--ticket', 'cli-wayfinding/issues/08-compact-paths.md', '--json']),
]
with tempfile.TemporaryDirectory(prefix='ctx08-workspace-') as tmp:
    workspace = Path(tmp)
    (workspace / '.context').mkdir()
    (workspace / '.context/config.md').write_text(f'''---
type: ContextConfig
version: 1
workspace:
  id: compact-path-parity
  title: Compact path parity
  members:
    - key: demo
      records: {DEMO / 'records'}
      directory: {DEMO}
      allowSources:
        - {DEMO / 'docs'}
---
''')
    for name, flags in [('text', []), ('detail', ['--detail']), ('json', ['--json'])]:
        cases.append(('workspace-' + name, workspace, ['orient', '--workspace', str(workspace), *flags]))
    report = []
    for name, cwd, args in cases:
        outputs = [subprocess.run([binary, *args], cwd=cwd, capture_output=True) for binary in [old, new]]
        a, b = outputs
        identical = (a.returncode, a.stdout, a.stderr) == (b.returncode, b.stdout, b.stderr)
        report.append({'case': name, 'cwd': str(cwd), 'args': args, 'identical': identical, 'exit': b.returncode,
                       'stdoutBytes': len(b.stdout), 'stdoutSHA256': hashlib.sha256(b.stdout).hexdigest(),
                       'stderrBytes': len(b.stderr), 'stderrSHA256': hashlib.sha256(b.stderr).hexdigest()})
        assert identical, name
        assert b.returncode == 0, (name, b.stderr)
    compact = []
    for name, binary in [('before', old), ('after', new)]:
        result = subprocess.run([binary, 'orient'], cwd=DEMO, capture_output=True)
        assert result.returncode == 0
        (OUT / f'demo-compact-{name}.txt').write_bytes(result.stdout)
        compact.append({'version': name, 'bytes': len(result.stdout), 'lines': len(result.stdout.splitlines()), 'exit': result.returncode})
    result = {'baselineRevision': '4f480b9f1f013bd09c17e18f5a7264fcef47a8ef', 'cases': report, 'compactDemo': compact}
    (OUT / 'parity.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps({'parityCases': len(report), 'allIdentical': all(r['identical'] for r in report), 'compactDemo': compact}))
