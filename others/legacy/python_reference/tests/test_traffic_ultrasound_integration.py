from __future__ import annotations
import json
from bridge_py import RouteRequest
from bridge_py.gate import Gate
from bridge_py.truss import Truss
from bridge_py.ultrasound import InMemoryObserver
from bridge_py.traffic import AcceptanceCriteria, TrafficRunner, TrafficScenario, grid_graph, disconnected_graph


def test_anchor_bearing_ultrasound_detailed_trace(tmp_path):
    graph=grid_graph(8,8,seed=3,noise=.05)
    observer=InMemoryObserver(); result=Gate(Truss(observer=observer)).route_request(graph,RouteRequest(0,63,mode='balanced',work_budget=256))
    assert result.found
    kinds=[e['kind'] for e in observer.events]
    assert 'phase_started' in kinds and 'phase_finished' in kinds
    assert 'budget_updated' in kinds and 'bound_updated' in kinds
    assert 'step_started' in kinds and 'node_expanded' in kinds and 'edge_relaxed' in kinds
    assert 'candidate_found' in kinds
    nodes=[e['event'] for e in observer.events if e['kind']=='node_expanded']
    assert all(n['task_id'].startswith('anchor-') and n['logical_step']>=1 for n in nodes)
    by_task={}
    for n in nodes: by_task.setdefault(n['task_id'],[]).append(n['logical_step'])
    assert all(v==list(range(1,len(v)+1)) for v in by_task.values())
    assert len(nodes) <= result.work_expanded_nodes
    path=observer.write_jsonl(tmp_path/'trace.jsonl')
    reread=InMemoryObserver.read_jsonl(path)
    assert len(reread.events)==len(observer.events)
    assert [e['sequence'] for e in reread.events]==list(range(1,len(reread.events)+1))


def test_ultrasound_non_interference_contract(tmp_path):
    runner=TrafficRunner(tmp_path)
    verdict=runner.compare_observation(grid_graph(10,10,seed=1),RouteRequest(0,99,mode='balanced',work_budget=100))
    assert verdict.passed, verdict.failures
    assert verdict.metrics['event_count']>0
    assert verdict.metrics['off_work']==verdict.metrics['on_work']


def test_traffic_regression_scenario_and_artifacts(tmp_path):
    scenario=TrafficScenario('grid-regression',None,None,3,(1,2,3),RouteRequest(0,99,mode='balanced',work_budget=100),'STEPS',AcceptanceCriteria(require_trace=True,max_work=100))
    records,verdict=TrafficRunner(tmp_path).run(scenario,lambda seed:grid_graph(10,10,seed=seed,noise=.1))
    assert verdict.passed, verdict.failures
    assert len(records)==3 and verdict.metrics['found_rate']==1.0
    manifest=json.loads((tmp_path/'manifest.json').read_text())
    assert len(manifest['records'])==3
    assert all(r.ultrasound_artifact for r in records)


def test_traffic_disconnected_and_budget_contract(tmp_path):
    scenario=TrafficScenario('disconnected',None,None,1,(0,),RouteRequest(0,11,mode='balanced',work_budget=10),'METRICS',AcceptanceCriteria(require_trace=True,max_work=10))
    records,verdict=TrafficRunner(tmp_path).run(scenario,lambda seed:disconnected_graph())
    assert verdict.passed, verdict.failures
    assert not records[0].result.found
    assert records[0].result.work_expanded_nodes<=10
