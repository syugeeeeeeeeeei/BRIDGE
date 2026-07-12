from pathlib import Path
from bridge_py import Graph, route

def test_anchor_has_no_bolts_dependency():
    root=Path(__file__).parents[1]/"bridge_py"/"anchor"
    text="\n".join(p.read_text() for p in root.glob("*.py"))
    assert "..bolts" not in text
    assert "fallback_exact" not in text[text.index("def run_anchor_algorithm"):]

def test_truss_owns_plan_and_bolts_own_recovery():
    g=Graph.from_edges([(0,1,1.0),(1,2,1.0),(2,3,1.0)],directed=False)
    r=route(g,0,3,mode="fast",work_budget=10)
    assert r.telemetry["anchor_plan"]["strategy"]
    assert r.telemetry["responsibility_split"] == {
        "planning":"TRUSS","primary_search":"ANCHOR","fallback_and_certification":"BOLTS"
    }
    assert r.telemetry["budget_violation"] is False
    assert r.telemetry["portfolio_work_used"] <= 10

def test_zero_budget_is_hard_limit():
    g=Graph.from_edges([(0,1,1.0),(1,2,1.0)],directed=False)
    r=route(g,0,2,mode="fast",work_budget=0)
    assert r.telemetry["portfolio_work_used"] == 0
    assert r.telemetry["budget_violation"] is False
