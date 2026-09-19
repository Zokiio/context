"""Independent CLI checkpoint fixtures; no production recovery helpers used."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import uuid


def digest(content):
    return hashlib.sha256(content).hexdigest()


def load_checker():
    source = Path(__file__).with_name('check_resume_contract.py')
    spec = importlib.util.spec_from_file_location('contract_checker', source)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.verify


def fixture(base, binary):
    records, working, docs = [base / name for name in ('records', 'working', 'docs')]
    for path in (records, working, docs):
        path.mkdir()
    (records / 'project.md').write_text('''---
type: Project
id: checkpoint-project
title: Checkpoint fixture
---
## Goals
Verify retained source comparisons.
## Current commitments
[Task](task.md)
## Open decisions
None
''')
    (records / 'task.md').write_text('''---
type: WorkItem
id: checkpoint-task
title: Compare retained context
triage: ready-for-agent
execution: in-progress
---
## Acceptance criteria
- [ ] Preserve old and current requirement text.
## Blocked by
None
## Blocked by decisions
None
## Spec
[Requirements](../docs/requirements.md)
''')
    (docs / 'requirements.md').write_text('Original requirement before uncommitted editing.\n')
    context_args = [binary, 'context', '--bundle', str(records), '--ticket', 'task.md', '--allow-source', str(docs)]
    captured = subprocess.run(context_args, capture_output=True)
    assert captured.returncode == 0, captured.stderr
    collection = json.loads(captured.stdout)
    assert collection['complete'] and collection['traversalComplete']
    namespace = working / '.context-cache/resume-v1' / digest(b'checkpoint-project') / digest(b'checkpoint-task')
    observation_id = str(uuid.uuid4())
    observation = namespace / 'observations' / observation_id
    observation.mkdir(parents=True)
    (observation / 'context.json').write_bytes(captured.stdout)
    header = {
        'type': 'RecoveryNote', 'version': 1, 'id': observation_id,
        'projectId': 'checkpoint-project', 'taskId': 'checkpoint-task',
        'observedAt': '2026-09-19T00:00:00+02:00', 'actor': 'independent-cli-fixture',
        'predecessors': [], 'ticketPath': 'task.md', 'checkoutRevision': None,
        'contextFile': 'context.json', 'contextSHA256': digest(captured.stdout),
    }
    body = '''## Approach
Exercise the public command with actual captured context bytes.
## Completed
Collected the original task requirements.
## Remaining
Inspect source differences.
## Checks
Controlled fixture claim: tests passed, without retained evidence or tested revision.
## Questions
None
## Failed approaches
None
## Next step
Refresh the current requirements and assess the report.
'''
    def publish_header():
        # JSON is a YAML subset, preserving required scalar types exactly.
        (observation / 'note.md').write_text('---\n' + json.dumps(header) + '\n---\n' + body)
    publish_header()
    args = [binary, 'resume', '--bundle', str(records), '--checkout', str(working), '--ticket', 'task.md', '--allow-source', str(docs), '--json']
    return dict(records=records, working=working, docs=docs, observation=observation,
                header=header, publish_header=publish_header, args=args,
                baseline=collection, observation_id=observation_id)


def tree_bytes(root):
    return {str(p.relative_to(root)): p.read_bytes() for p in root.rglob('*') if p.is_file() and not p.is_symlink()}


def run(binary, output):
    verify = load_checker()
    rows = []
    for case in ('valid', 'changed', 'invalid-snapshot', 'partial-snapshot', 'invalid-note', 'missing-snapshot',
                 'authorization-loss', 'current-source-missing', 'root-relocation', 'entry-limit', 'byte-limit'):
        with tempfile.TemporaryDirectory(prefix='ctx-checkpoint-probe-') as temporary:
            base = Path(temporary).resolve()
            f = fixture(base, binary)
            if case == 'changed':
                (f['docs'] / 'requirements.md').write_text('Updated uncommitted requirement.\n')
            elif case == 'invalid-snapshot':
                (f['observation'] / 'context.json').write_bytes(b'{}\n')
            elif case == 'partial-snapshot':
                partial = subprocess.run([binary, 'context', '--bundle', str(f['records']), '--ticket', 'task.md', '--allow-source', str(f['docs']), '--max-files', '1'], capture_output=True)
                assert partial.returncode == 1 and not json.loads(partial.stdout)['complete']
                (f['observation'] / 'context.json').write_bytes(partial.stdout)
                f['header']['contextSHA256'] = digest(partial.stdout)
                f['publish_header']()
            elif case == 'invalid-note':
                note = f['observation'] / 'note.md'
                note.write_text(note.read_text() + '\n## Checks\nDuplicated required section.\n')
            elif case == 'missing-snapshot':
                (f['observation'] / 'context.json').unlink()
            elif case == 'authorization-loss':
                index = f['args'].index('--allow-source')
                del f['args'][index:index + 2]
            elif case == 'current-source-missing':
                (f['docs'] / 'requirements.md').unlink()
            elif case == 'root-relocation':
                (f['records'] / 'task.md').rename(f['records'] / 'moved.md')
                manifest = f['records'] / 'project.md'
                manifest.write_text(manifest.read_text().replace('(task.md)', '(moved.md)'))
                f['args'][f['args'].index('--ticket') + 1] = 'moved.md'
            elif case == 'entry-limit':
                f['args'] += ['--max-cache-files', '1']
            elif case == 'byte-limit':
                f['args'] += ['--max-cache-bytes', '1']
            before = tree_bytes(base)
            command = subprocess.run(f['args'], capture_output=True, text=True)
            expected = 0 if case in ('valid', 'changed', 'root-relocation') else 1
            assert command.returncode == expected, (case, command.returncode, command.stderr, command.stdout[:1200])
            assert command.stderr == '', (case, command.stderr)
            report = json.loads(command.stdout)
            verify(report)
            assert tree_bytes(base) == before, case + ' changed fixture files'
            if case in ('valid', 'changed', 'root-relocation'):
                assert report['recovery']['status'] == 'available'
                assert report['recovery']['observations'][0]['snapshot']['status'] == 'valid'
                assert report['comparison']['baselineAvailable']
                diffs = report['comparison']['candidates'][0]['sources']
                if case == 'valid':
                    assert all(d['status'] == 'unchanged' for d in diffs)
                elif case == 'changed':
                    change, = [d for d in diffs if d['status'] == 'changed']
                    assert 'Original requirement' in change['previous']['text']
                    assert 'Updated uncommitted' in change['current']['text']
                else:
                    roots = [d for d in diffs if d['previous'] and d['current'] and d['previous']['path'].endswith('/task.md') and d['current']['path'].endswith('/moved.md')]
                    assert len(roots) == 1 and roots[0]['status'] == 'unchanged'
            elif case in ('invalid-snapshot', 'missing-snapshot'):
                assert report['recovery']['status'] == 'available'
                assert report['recovery']['graphStatus'] == 'valid'
                assert not report['comparison']['baselineAvailable']
                assert len(report['comparison']['candidates']) == 1
            elif case == 'authorization-loss':
                assert report['recovery']['status'] == 'available'
                assert report['recovery']['observations'][0]['snapshot']['status'] == 'withheld'
                assert 'Original requirement before uncommitted editing.' not in command.stdout
                assert any(d['code'] == 'recovery_snapshot_outside_scope' for d in report['diagnostics'])
            elif case == 'partial-snapshot':
                assert report['recovery']['status'] == 'available'
                assert report['recovery']['observations'][0]['snapshot']['status'] == 'incomplete'
                assert report['comparison']['baselineAvailable']
                assert any(d['status'] == 'unknown' and d['previous'] is None for d in report['comparison']['candidates'][0]['sources'])
            elif case in ('invalid-note', 'entry-limit', 'byte-limit'):
                assert report['recovery']['status'] == 'unknown'
                assert report['recovery']['candidates'] == []
            rows.append(dict(case=case, exit=command.returncode, complete=report['complete'],
                             recoveryStatus=report['recovery']['status'], graphStatus=report['recovery']['graphStatus'],
                             snapshotStatuses=[n['snapshot']['status'] for n in report['recovery']['observations']],
                             baselineAvailable=report['comparison']['baselineAvailable'],
                             diagnosticCodes=[d['code'] for d in report['diagnostics']], readOnly=True,
                             reportSHA256=digest(command.stdout.encode())))
    Path(output).write_text(json.dumps(rows, indent=2) + '\n')
    for row in rows:
        print(row['case'], row['exit'], row['recoveryStatus'], row['snapshotStatuses'])


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('binary')
    parser.add_argument('output')
    args = parser.parse_args()
    run(args.binary, args.output)
