import ast
from pathlib import Path

from bridge_py import Graph, RouteRequest
from bridge_py.bearing import NullObserver
from bridge_py.gate import Gate
from bridge_py.truss import Truss
from bridge_py.ultrasound import InMemoryObserver

ROOT = Path(__file__).parents[2] / 'src' / 'bridge_py'


def imports_under(package):
    found = set()
    for file in (ROOT / package).rglob('*.py'):
        tree = ast.parse(file.read_text(encoding='utf-8'))
        for node in ast.walk(tree):
            if isinstance(node, ast.ImportFrom) and node.module:
                found.add(node.module)
            elif isinstance(node, ast.Import):
                found.update(x.name for x in node.names)
    return found


def test_forbidden_production_dependencies():
    for package in ('truss', 'anchor', 'bolts', 'gate', 'bearing'):
        imports = imports_under(package)
        assert not any('ultrasound' in x or 'traffic' in x for x in imports)
    gate_imports = imports_under('gate')
    assert not any('.anchor' in x or '.bolts' in x for x in gate_imports)


def test_ultrasound_non_interference():
    graph = Graph.from_edges([('A','B',1),('B','C',1),('A','C',4)], pos={'A':(0,0),'B':(1,0),'C':(2,0)})
    req = RouteRequest('A','C',mode='quality')
    off = Gate(Truss(observer=NullObserver())).route_request(graph, req)
    obs = InMemoryObserver()
    on = Gate(Truss(observer=obs)).route_request(graph, req)
    assert off.path == on.path
    assert off.distance == on.distance
    assert off.total_work == on.total_work
    assert off.solver_trace == on.solver_trace
    assert obs.events


def test_gate_calls_truss_contract():
    graph = Graph.from_edges([(0,1,1),(1,2,1)])
    result = Gate().route(graph, 0, 2, mode='balanced')
    assert result.found
    assert result.telemetry['architecture'] == 'TRUSS/ANCHOR/BOLTS/BEARING/GATE'


def test_explicit_budget_is_accounted():
    graph = Graph.from_edges([(0,1,1),(1,2,1),(0,2,5)])
    result = Gate().route(graph, 0, 2, mode='fast', work_budget=10)
    assert result.telemetry['portfolio_work_budget'] == 10
    assert 'budget_violation' in result.telemetry
