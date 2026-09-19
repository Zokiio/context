"""Controlled workflow probes. Separate from the real fresh-agent trial."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import uuid
from checkpoint_probe import fixture, tree_bytes, load_checker, digest
from graph_probe import add_note


def command(args, cwd=None):
    result = subprocess.run(args, cwd=cwd, capture_output=True, text=True,
                            env=dict(os.environ, GIT_OPTIONAL_LOCKS='0'))
    assert result.returncode == 0, (args, result.stderr, result.stdout)
    return result.stdout


def git(directory, *args):
    return command(['git', '-C', str(directory), *args])


def initialize_git(directory):
    git(directory, 'init', '-q')
    git(directory, 'config', 'user.name', 'CLI workflow fixture')
    git(directory, 'config', 'user.email', 'fixture@example.invalid')


def git_output(repository, *args):
    result = subprocess.run(['git', '-C', str(repository), *args], capture_output=True, text=True)
    assert result.returncode == 0, (args, result.stderr, result.stdout)
    return result.stdout


def source_identity(repository):
    repository = Path(repository).resolve()
    paths = []
    for root in ('cmd', 'internal'):
        directory = repository / root
        paths.extend(path for path in directory.rglob('*') if path.is_file())
    paths.extend(repository / name for name in ('go.mod', 'go.sum'))
    manifest = [dict(path=str(path.relative_to(repository)), sha256=digest(path.read_bytes()))
                for path in sorted(paths) if path.is_file()]
    manifest_bytes = json.dumps(manifest, sort_keys=True, separators=(',', ':')).encode()
    return dict(repository=str(repository), head=git_output(repository, 'rev-parse', 'HEAD').strip(),
                dirtyDiffSHA256=digest(git_output(repository, 'diff', '--binary', 'HEAD', '--', 'cmd', 'internal', 'go.mod', 'go.sum').encode()),
                dirtyStatus=git_output(repository, 'status', '--porcelain=v1', '--', 'cmd', 'internal', 'go.mod', 'go.sum').splitlines(),
                sourceManifestSHA256=digest(manifest_bytes), sourceManifest=manifest)


def build_binary(binary, repository):
    binary = Path(binary).resolve()
    before = source_identity(repository)
    build_command = ['go', 'build', '-mod=readonly', '-o', str(binary), './cmd/ctx']
    result = subprocess.run(build_command, cwd=before['repository'], capture_output=True, text=True)
    assert result.returncode == 0, (build_command, result.stderr, result.stdout)
    after = source_identity(repository)
    assert after == before, 'building the CLI changed its source identity'
    assert binary.is_file(), 'go build did not produce the requested binary'
    return dict(buildFrom=before['repository'], buildCommand=build_command,
                sourceBefore=before, sourceAfter=after, binarySHA256=digest(binary.read_bytes()))


def binary_identity(binary):
    binary = Path(binary).resolve()
    assert binary.is_file(), 'CLI binary is missing: ' + str(binary)
    return dict(buildFrom=None, buildCommand=None, sourceBefore=None, sourceAfter=None,
                binarySHA256=digest(binary.read_bytes()))


def has_diagnostic(report, code, path):
    return any(d['code'] == code and d.get('path') == str(path) for d in report['diagnostics'])


def work_by_id(report, identity):
    work, = [work for work in report['orientation']['workItems'] if work['id'] == identity]
    return work


def check_by_name(work, name):
    check, = [check for check in work['checks'] if check['name'] == name]
    return check


def selected_fields(report):
    if report is None:
        return None
    if report.get('kind') != 'task-resumption':
        return dict(kind=report.get('kind'), complete=report.get('complete'),
                    diagnostics=[dict(code=d['code'], path=d.get('path')) for d in report.get('diagnostics', [])])
    return dict(complete=report['complete'], contextComplete=report['context']['complete'],
                orientationComplete=report['orientation']['complete'],
                orientationInventoryComplete=report['orientation']['inventoryComplete'],
                recoveryInventoryComplete=report['recovery']['inventoryComplete'],
                recoveryGraphStatus=report['recovery']['graphStatus'],
                diagnostics=[{'code': d['code'], 'path': d['path'], 'from': d['from'], 'link': d['link'],
                              'observationId': d['observationId']} for d in report['diagnostics']])


def run(binary, output, identity):
    check = load_checker()
    rows = []
    with tempfile.TemporaryDirectory(prefix='ctx-workflow-probe-') as temporary:
        base = Path(temporary).resolve()
        f = fixture(base, binary)
        records, docs, app = f['records'], f['docs'], f['working']
        initialize_git(records)
        initialize_git(app)
        (app / '.gitignore').write_text('.context-cache/\n')
        (app / 'local_check.py').write_text('assert 2 + 2 == 4\nprint("local fixture check passed")\n')
        git(app, 'add', '.gitignore', 'local_check.py')
        git(app, 'commit', '-qm', 'Record local fixture check')
        tested_revision = git(app, 'rev-parse', 'HEAD').strip()
        check_output = command(['python3', str(app / 'local_check.py')])
        (docs / 'local-evidence.md').write_text(
            '# Historical local fixture result\n\nActor: workflow probe.\n\n'
            + 'Command: `python3 local_check.py`. Tested revision: `' + tested_revision
            + '`.\n\nOutput: ' + check_output + '\nNo device or deployment check was performed.\n')
        (records / 'decision.md').write_text('''---
type: Decision
id: fixture-runtime-decision
title: Select the fixture runtime target
decisionState: open
---
## Question
Which device/runtime target is authorized for the remaining verification?
''')
        task = records / 'task.md'
        task.write_text(task.read_text().replace('## Blocked by\nNone', '## Blocked by\n[Local prerequisite](prerequisite.md)')
                        .replace('## Blocked by decisions\nNone', '## Blocked by decisions\n[Runtime question](decision.md)')
                        + '\n## Context\n[Runtime question](decision.md)\n'
                        + '\n## Verification gap\nDevice/runtime verification has not run. The dependent outcome remains unfinished.\n')
        prerequisite = records / 'prerequisite.md'
        prerequisite.write_text('''---
type: WorkItem
id: fixture-local-prerequisite
title: Verify local fixture arithmetic
triage: ready-for-agent
execution: in-progress
---
## Acceptance criteria
- [x] Run the retained local arithmetic check.
## Blocked by
None
## Blocked by decisions
None
## Acceptance

''')
        (records / 'outcome.md').write_text('''---
type: WorkItem
id: fixture-broader-outcome
title: Finish the broader runtime outcome
triage: ready-for-agent
execution: unstarted
---
## Acceptance criteria
- [ ] Complete backend and runtime verification.
## Blocked by
[Backend](task.md)
## Blocked by decisions
None
''')
        manifest = records / 'project.md'
        manifest.write_text(manifest.read_text().replace('## Open decisions\nNone', '## Open decisions\n[Runtime question](decision.md)'))
        common = ['--bundle', str(records), '--allow-source', str(docs)]
        orient = json.loads(command([binary, 'orient', *common, '--json']))
        work, = [w for w in orient['workItems'] if w['id'] == 'fixture-local-prerequisite']
        acceptance = dict(type='Acceptance', id=str(uuid.uuid4()), title='Historical fixture acceptance',
                          projectId='checkpoint-project', workItemId=work['id'],
                          actor=dict(kind='workflow', identity='independent workflow probe'),
                          decidedAt='2026-09-19T02:00:00+02:00',
                          testedRevision=dict(origin=str(app), revision=tested_revision),
                          fingerprintVersion=work['fingerprintVersion'], ticketSHA256=work['ticketSHA256'],
                          criteriaSHA256=work['criteriaSHA256'])
        evidence_digest = digest((docs / 'local-evidence.md').read_bytes())
        (records / 'acceptance.md').write_text('---\n' + json.dumps(acceptance) + '\n---\n'
             + '## Requirements\nNone\n## Evidence\n'
             + '- [Local result](../docs/local-evidence.md) `' + evidence_digest + '`\n')
        prerequisite.write_text(prerequisite.read_text().replace('execution: in-progress', 'execution: completed')
                               + '[Acceptance](acceptance.md)\n')
        captured = command([binary, 'context', *common, '--ticket', 'task.md']).encode()
        (f['observation'] / 'context.json').write_bytes(captured)
        f['header']['contextSHA256'] = digest(captured)
        f['header']['checkoutRevision'] = dict(origin=str(app), revision=tested_revision)
        f['publish_header']()
        note = f['observation'] / 'note.md'
        note.write_text(note.read_text().replace('## Questions\nNone',
             '## Questions\nRuntime target remains open in the authoritative decision.md.\n')
             .replace('## Remaining\nInspect source differences.',
             '## Remaining\nDevice/runtime verification is missing. Inspect source differences.'))
        git(records, 'add', '.')
        git(records, 'commit', '-qm', 'Record controlled workflow requirements and historical evidence')
        (app / 'next.txt').write_text('Later application revision; historical evidence stays attributed.\n')
        git(app, 'add', 'next.txt')
        git(app, 'commit', '-qm', 'Advance application after the historical check')
        a, b = base / 'checkout-a', base / 'checkout-b'
        git(app, 'worktree', 'add', '-q', '-b', 'fixture-a', str(a))
        git(app, 'worktree', 'add', '-q', '-b', 'fixture-b', str(b))
        shutil.copytree(app / '.context-cache', a / '.context-cache')
        f['observation'] = a / f['observation'].relative_to(app)
        args = [binary, 'resume', *common, '--checkout', str(a), '--ticket', 'task.md', '--json']
        git_roots = [records, app, a, b]

        def read(label, argv, expected=0):
            before = tree_bytes(base)
            status_before = [git(p, 'status', '--porcelain=v1', '--untracked-files=all') for p in git_roots]
            result = subprocess.run(argv, capture_output=True, text=True)
            assert result.returncode == expected, (label, result.returncode, result.stderr, result.stdout[:500])
            assert not result.stderr, (label, result.stderr)
            assert tree_bytes(base) == before, label + ' changed file bytes'
            assert status_before == [git(p, 'status', '--porcelain=v1', '--untracked-files=all') for p in git_roots]
            report = json.loads(result.stdout) if '--json' in argv or argv[1] == 'context' else None
            if report and report.get('kind') == 'task-resumption':
                check(report)
            rows.append(dict(case=label, command=argv, exit=result.returncode,
                             stdoutSHA256=digest(result.stdout.encode()), readOnly=True,
                             gitStateUnchanged=True, complete=report.get('complete') if report else None,
                             selected=selected_fields(report)))
            return report, result.stdout

        current, text = read('separate-record-store-and-historical-evidence', args)
        assert current['scope']['workingDirectory'] == str(a) and current['scope']['recordsDirectory'] == str(records)
        works = {w['id']: w for w in current['orientation']['workItems']}
        assert works['fixture-local-prerequisite']['acceptance']['status'] == 'valid'
        assert works['fixture-local-prerequisite']['acceptance']['testedRevision']['revision'] == tested_revision
        assert works['checkpoint-task']['readiness'] == 'blocked'
        assert works['checkpoint-task']['acceptance'] is None
        assert works['fixture-broader-outcome']['execution'] == 'unstarted'
        assert works['fixture-broader-outcome']['readiness'] == 'blocked'
        assert 'without retained evidence or tested revision' in text and 'Device/runtime verification is missing' in text
        decision_report, = [decision for decision in current['orientation']['decisions'] if decision['id'] == 'fixture-runtime-decision']
        decision_record = records / 'decision.md'
        assert decision_report['source'] == str(decision_record) and decision_report['state'] == 'open'
        assert decision_report['checkStatus'] == 'fail'
        assert any(reason['code'] == 'open_decision' and reason['path'] == str(records / 'decision.md')
                   for reason in decision_report['reasons'])
        task_checks = check_by_name(work_by_id(current, 'checkpoint-task'), 'blocking_decisions')
        assert task_checks['status'] == 'fail'
        assert any(reason['code'] == 'open_decision' and reason['path'] == str(records / 'decision.md')
                   and reason['from'] == str(task) and reason['link'] == 'decision.md'
                   for reason in task_checks['reasons'])
        decision_record.write_text(decision_record.read_text().replace('decisionState: open', 'decisionState: resolved')
                                   + '\n## Resolution\nFixture runtime target is authorized for this controlled check.\n')
        resolved, _ = read('resolved-decision-unblocks-task', args)
        assert work_by_id(resolved, 'checkpoint-task')['readiness'] == 'ready'
        resolved_checks = check_by_name(work_by_id(resolved, 'checkpoint-task'), 'blocking_decisions')
        assert resolved_checks['status'] == 'pass'
        assert any(reason['code'] == 'decision_resolved' and reason['path'] == str(records / 'decision.md')
                   for reason in resolved_checks['reasons'])
        decision_record.write_text(decision_record.read_text().replace('decisionState: resolved', 'decisionState: open')
                                   .replace('\n## Resolution\nFixture runtime target is authorized for this controlled check.\n', '\n'))
        restored, _ = read('restored-open-decision-blocks-task', args)
        assert work_by_id(restored, 'checkpoint-task')['readiness'] == 'blocked'
        for label, tail in [('compact-operator', []), ('detailed-operator', ['--detail']), ('orientation-agent', ['--json'])]:
            read(label, [binary, 'orient', *common, *tail])
        read('task-context', [binary, 'context', *common, '--ticket', 'task.md'])
        _, rendered = read('resume-operator', args[:-1])
        assert 'completion not established' in rendered.lower()
        missing = args.copy()
        missing[missing.index('--checkout') + 1] = str(b)
        absent, _ = read('other-checkout-has-no-notes', missing)
        assert absent['recovery']['status'] == 'absent' and not absent['comparison']['baselineAvailable']
        (docs / 'requirements.md').write_text('Changed uncommitted requirement for the current task.\n')
        changed, _ = read('uncommitted-requirement-change', args)
        assert any(s['status'] == 'changed' and 'Changed uncommitted' in s['current']['text']
                   for s in changed['comparison']['candidates'][0]['sources'])
        left, _ = add_note(f, [f['observation_id']])
        right, _ = add_note(f, [f['observation_id']], '2040-01-01T00:00:00Z')
        conflict, _ = read('competing-successors', args)
        assert conflict['recovery']['candidates'] == sorted([left, right]) and conflict['recovery']['status'] == 'conflicting'
        partial = f['observation'].parent / str(uuid.uuid4())
        partial.mkdir()
        (partial / 'context.json').write_bytes(captured)
        interrupted, _ = read('unfinished-publication', args, 1)
        assert interrupted['recovery']['graphStatus'] == 'incomplete' and not interrupted['recovery']['candidates']
        assert not (partial / 'note.md').exists()
        archive = partial.parent.parent / 'quarantine' / str(uuid.uuid4())
        archive.parent.mkdir()
        partial.rename(archive)
        (archive / 'recovery.md').write_text('Sequential fixture writer completed; no inbound references or final note.\n')
        recovered, _ = read('confirmed-stop-quarantine', args)
        assert recovered['recovery']['candidates'] == sorted([left, right])
        merged, _ = add_note(f, [left, right])
        reconciled, _ = read('deliberate-reconciliation', args)
        assert reconciled['recovery']['candidates'] == [merged]
        restricted = args.copy()
        index = restricted.index('--allow-source')
        del restricted[index:index + 2]
        withheld, restricted_text = read('authorization-loss', restricted, 1)
        assert 'Original requirement before uncommitted editing.' not in restricted_text
        assert any(d['code'] == 'recovery_snapshot_outside_scope' for d in withheld['diagnostics'])
        source_limited, _ = read('current-source-limit', args + ['--max-files', '1'], 1)
        assert not source_limited['complete'] and not source_limited['context']['complete']
        assert has_diagnostic(source_limited, 'source_limit_exceeded', task)
        limited, _ = read('cache-entry-limit', args + ['--max-cache-files', '1'], 1)
        assert not limited['recovery']['candidates']
        assert not limited['complete'] and not limited['recovery']['inventoryComplete']
        assert limited['recovery']['graphStatus'] == 'incomplete'
        assert has_diagnostic(limited, 'recovery_limit_exceeded', f['observation'].parent)
        byte_limited, _ = read('cache-byte-limit', args + ['--max-cache-bytes', '1'], 1)
        assert not byte_limited['complete'] and not byte_limited['recovery']['inventoryComplete']
        assert byte_limited['recovery']['graphStatus'] == 'incomplete'
        assert any(d['code'] == 'recovery_limit_exceeded' and d['path'] is not None
                   and Path(d['path']).name == 'note.md'
                   and Path(d['path']).parent.parent == f['observation'].parent
                   for d in byte_limited['diagnostics'])
        overridden, _ = read('explicit-limit-overrides', args + ['--max-files', '1000', '--max-cache-files', '1000', '--max-cache-bytes', '16777216'])
        assert overridden['complete']
        independent = base / 'independent-project'
        independent.mkdir()
        second = fixture(independent, binary)
        other_manifest = second['records'] / 'project.md'
        other_manifest.write_text(other_manifest.read_text().replace('checkpoint-project', 'independent-project'))
        second['header']['projectId'] = 'independent-project'
        second['publish_header']()
        namespace_root = second['working'] / '.context-cache/resume-v1'
        (namespace_root / digest(b'checkpoint-project')).rename(namespace_root / digest(b'independent-project'))
        report, _ = read('independent-nongit-project', second['args'])
        assert report['recovery']['status'] == 'available'
        assert report['recovery']['observations'][0]['checkoutRevision'] is None
        assert not (second['records'] / '.git').exists() and not (second['working'] / '.git').exists()
        other_scope = second['args'].copy()
        other_scope[other_scope.index('--checkout') + 1] = str(a)
        report, _ = read('same-task-id-other-project-keeps-separate-namespace', other_scope)
        assert report['scope']['projectId'] == 'independent-project' and report['recovery']['status'] == 'absent'
        authoritative = {**{str(p): p.read_bytes() for p in records.rglob('*') if p.is_file()},
                         **{str(p): p.read_bytes() for p in docs.rglob('*') if p.is_file()}}
        cache = a / '.context-cache'
        assert cache.parent == a and a.is_relative_to(base)
        shutil.rmtree(cache)
        assert all(Path(p).read_bytes() == data for p, data in authoritative.items())
        deleted, _ = read('cache-deletion-preserves-authority', args)
        assert deleted['recovery']['status'] == 'absent'
        works = {w['id']: w for w in deleted['orientation']['workItems']}
        assert works['fixture-local-prerequisite']['acceptance']['status'] == 'valid'
        assert works['checkpoint-task']['readiness'] == 'blocked'
        rows[-1]['authoritativeBytesUnchangedAfterDeletion'] = True
    Path(output).write_text(json.dumps(dict(kind='controlled-fixtures-v2', cliIdentity=identity,
                                             historicalTestedRevision=tested_revision, cases=rows), indent=2) + '\n')
    for row in rows:
        print(row['case'], row['exit'], 'read-only')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('binary')
    parser.add_argument('output')
    parser.add_argument('--build-from')
    options = parser.parse_args()
    identity = build_binary(options.binary, options.build_from) if options.build_from else binary_identity(options.binary)
    run(options.binary, options.output, identity)
