"""Independent assertions for the approved version-1 resumption JSON contract."""
import json
import sys


def fields(value, shape, where):
    assert isinstance(value, dict), where
    for name, expected in shape.items():
        assert name in value, f'{where}.{name} is missing'
        item = value[name]
        types = expected if isinstance(expected, tuple) else (expected,)
        assert type(item) in types, f'{where}.{name}: {type(item).__name__}'


def verify(report):
    null = type(None)
    fields(report, dict(schemaVersion=int, kind=str, complete=bool, scope=dict,
                        orientation=dict, context=dict, recovery=dict,
                        comparison=dict, diagnostics=list), 'report')
    assert report['schemaVersion'] == 1 and report['kind'] == 'task-resumption'
    fields(report['scope'], dict(recordsDirectory=str, workingDirectory=str,
                                cacheRoot=str, projectId=(str, null),
                                taskId=(str, null)), 'scope')
    recovery = report['recovery']
    fields(recovery, dict(status=str, inventoryComplete=bool, graphStatus=str,
                         observations=list, candidates=list), 'recovery')
    assert recovery['status'] in ('absent', 'available', 'conflicting', 'unknown')
    assert recovery['graphStatus'] in ('valid', 'invalid', 'incomplete', 'not_evaluated')
    ids = []
    by_id = {}
    for note in recovery['observations']:
        fields(note, dict(id=str, projectId=str, taskId=str, observedAt=str, actor=str,
                          predecessors=list, ticketPath=str, checkoutRevision=(dict, null),
                          source=dict, body=str, metadata=dict, snapshot=dict), 'observation')
        ids.append(note['id'])
        by_id[note['id']] = note
        assert all(type(p) is str for p in note['predecessors'])
        if note['checkoutRevision'] is not None:
            fields(note['checkoutRevision'], dict(origin=str, revision=str), 'revision')
        fields(note['source'], dict(path=str, sha256=(str, null)), 'note.source')
        snap = note['snapshot']
        fields(snap, dict(path=str, recordedSHA256=str, observedSHA256=(str, null),
                          status=str, sourceCount=(int, null)), 'snapshot')
        assert snap['status'] in ('not_loaded', 'valid', 'incomplete', 'invalid', 'unavailable', 'withheld')
        if snap['status'] == 'not_loaded':
            assert snap['observedSHA256'] is None and snap['sourceCount'] is None
    assert ids == sorted(set(ids)), 'observations must have unique lexical IDs'
    candidates = recovery['candidates']
    assert candidates == sorted(set(candidates))
    assert set(candidates) <= set(ids)
    if recovery['graphStatus'] == 'valid':
        assert recovery['inventoryComplete']
        assert all(set(n['predecessors']) <= set(ids) for n in by_id.values())
        predecessors = {p for n in by_id.values() for p in n['predecessors']}
        assert candidates == sorted(set(ids) - predecessors)
        expected = 'absent' if not candidates else 'available' if len(candidates) == 1 else 'conflicting'
        assert recovery['status'] == expected
    else:
        assert candidates == [] and recovery['status'] == 'unknown'
    for ident, note in by_id.items():
        if ident not in candidates:
            assert note['snapshot']['status'] == 'not_loaded'
    comparison = report['comparison']
    fields(comparison, dict(baselineAvailable=bool, complete=bool, candidates=list), 'comparison')
    comparisons = comparison['candidates']
    assert [c['observationId'] for c in comparisons] == candidates
    for candidate in comparisons:
        fields(candidate, dict(observationId=str, baselineAvailable=bool, complete=bool, sources=list), 'candidate comparison')
        for difference in candidate['sources']:
            fields(difference, dict(status=str, previous=(dict, null), current=(dict, null)), 'source difference')
            assert difference['status'] in ('unchanged', 'changed', 'added', 'removed', 'unknown')
            for side in ('previous', 'current'):
                source = difference[side]
                if source is None:
                    continue
                fields(source, dict(path=str, sha256=(str, null), text=(str, null), availability=str), 'compared source')
                assert source['availability'] in ('available', 'unavailable', 'withheld')
                if source['availability'] == 'available':
                    assert type(source['text']) is str and type(source['sha256']) is str
                else:
                    assert source['text'] is None
            if by_id[candidate['observationId']]['snapshot']['status'] == 'invalid':
                assert difference['previous'] is None or difference['previous']['text'] is None
    assert comparison['baselineAvailable'] == any(c['baselineAvailable'] for c in comparisons)
    assert comparison['complete'] == (report['context']['complete'] and recovery['graphStatus'] == 'valid' and all(c['complete'] for c in comparisons))
    assert report['complete'] == (report['orientation']['complete'] and report['context']['complete'] and comparison['complete'] and recovery['inventoryComplete'] and recovery['graphStatus'] == 'valid')
    seen = set()
    for d in report['diagnostics']:
        fields(d, {'code': str, 'severity': str, 'message': str, 'path': (str, null),
                   'from': (str, null), 'link': (str, null), 'observationId': (str, null)}, 'diagnostic')
        assert d['severity'] in ('error', 'warning')
        key = tuple(d[k] for k in ('code', 'severity', 'path', 'from', 'link', 'observationId'))
        assert key not in seen, 'duplicate diagnostic attribution'
        seen.add(key)


if __name__ == '__main__':
    for path in sys.argv[1:]:
        with open(path) as f:
            verify(json.load(f))
        print(path + ': valid resumption shape and consistency')
