"""Independent public CLI graph fixtures using actual reader snapshot bytes."""
import argparse
import json
from pathlib import Path
import subprocess
import tempfile
import uuid
from checkpoint_probe import fixture, tree_bytes, load_checker, digest


def add_note(f, predecessors, observed_at='2020-01-01T00:00:00Z'):
    identity = str(uuid.uuid4())
    directory = f['observation'].parent / identity
    directory.mkdir()
    data = (f['observation'] / 'context.json').read_bytes()
    (directory / 'context.json').write_bytes(data)
    header = dict(f['header'], id=identity, predecessors=predecessors, observedAt=observed_at)
    body = (f['observation'] / 'note.md').read_text().split('\n---\n', 1)[1]
    (directory / 'note.md').write_text('---\n' + json.dumps(header) + '\n---\n' + body)
    return identity, directory


def run(binary, output):
    check = load_checker()
    rows = []
    cases = ['linear', 'multiple-roots', 'competing-successors', 'reconciled', 'dangling',
             'cycle', 'unfinished', 'quarantined', 'superseded-missing-snapshot',
             'candidate-invalid-snapshot', 'limited', 'invalid-and-incomplete']
    for case in cases:
        with tempfile.TemporaryDirectory(prefix='ctx-graph-probe-') as temporary:
            base = Path(temporary).resolve()
            f = fixture(base, binary)
            root = f['observation_id']
            left, left_dir = add_note(f, [] if case == 'multiple-roots' else [root])
            candidates = [left]
            if case in ('multiple-roots',):
                candidates = sorted([root, left])
            if case in ('competing-successors', 'reconciled', 'unfinished', 'quarantined'):
                right, _ = add_note(f, [root], '2040-01-01T00:00:00Z')
                candidates = sorted([left, right])
                if case == 'reconciled':
                    merged, _ = add_note(f, [left, right], '2010-01-01T00:00:00Z')
                    candidates = [merged]
            if case in ('dangling', 'invalid-and-incomplete'):
                note = left_dir / 'note.md'
                note.write_text(note.read_text().replace(root, str(uuid.uuid4())))
            if case == 'cycle':
                f['header']['predecessors'] = [left]
                f['publish_header']()
            if case in ('unfinished', 'quarantined', 'invalid-and-incomplete'):
                partial = f['observation'].parent / str(uuid.uuid4())
                partial.mkdir()
                (partial / 'context.json').write_text('{}\n')
                if case == 'quarantined':
                    # This fixture writer is this sequential process and has stopped writing.
                    assert not (partial / 'note.md').exists()
                    archive = partial.parent.parent / 'quarantine' / str(uuid.uuid4())
                    archive.parent.mkdir()
                    partial.rename(archive)
                    (archive / 'recovery.md').write_text('Fixture writer completed; partial publication retained.\n')
            if case == 'superseded-missing-snapshot':
                (f['observation'] / 'context.json').unlink()
            if case == 'candidate-invalid-snapshot':
                (left_dir / 'context.json').write_text('{}\n')
            if case == 'limited':
                f['args'] += ['--max-cache-files', '1']
            before = tree_bytes(base)
            result = subprocess.run(f['args'], capture_output=True, text=True)
            incomplete = case in ('dangling', 'cycle', 'unfinished', 'candidate-invalid-snapshot', 'limited', 'invalid-and-incomplete')
            assert result.returncode == int(incomplete), (case, result.returncode, result.stderr)
            assert not result.stderr, (case, result.stderr)
            report = json.loads(result.stdout)
            check(report)
            assert tree_bytes(base) == before, case + ' wrote files'
            recovery = report['recovery']
            invalid_graph = case in ('dangling', 'cycle', 'invalid-and-incomplete')
            partial_graph = case in ('unfinished', 'limited')
            if invalid_graph or partial_graph:
                assert recovery['status'] == 'unknown'
                assert recovery['graphStatus'] == ('invalid' if invalid_graph else 'incomplete')
                assert recovery['candidates'] == []
                assert all(n['snapshot']['status'] == 'not_loaded' for n in recovery['observations'])
            else:
                assert recovery['graphStatus'] == 'valid'
                assert recovery['candidates'] == candidates, (case, recovery['candidates'], candidates)
                assert recovery['status'] == ('conflicting' if len(candidates) > 1 else 'available')
                assert len(report['comparison']['candidates']) == len(candidates)
                assert all(n['snapshot']['status'] == 'not_loaded' for n in recovery['observations'] if n['id'] not in candidates)
            rows.append(dict(case=case, exit=result.returncode, status=recovery['status'],
                             graphStatus=recovery['graphStatus'], complete=report['complete'],
                             candidateCount=len(recovery['candidates']), observationCount=len(recovery['observations']),
                             diagnostics=[d['code'] for d in report['diagnostics']], readOnly=True,
                             reportSHA256=digest(result.stdout.encode())))
    Path(output).write_text(json.dumps(rows, indent=2) + '\n')
    for row in rows:
        print(row['case'], row['exit'], row['status'], row['graphStatus'])


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('binary')
    parser.add_argument('output')
    args = parser.parse_args()
    run(args.binary, args.output)
