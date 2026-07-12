from bridge_py import Graph, route
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback, build_component_index


def test_mrpc_dg5_basic_path():
    G = Graph.from_edges([
        ("A", "B", 1.0),
        ("B", "C", 1.0),
        ("A", "C", 3.0),
    ])
    r = mrpc_dg5_switchback(G, "A", "C", workers=2, min_budget=2)
    assert r.found
    assert r.distance <= 3.0


def test_mrpc_dg5_component_precheck():
    G = Graph.from_edges([
        ("A", "B", 1.0),
        ("C", "D", 1.0),
    ])
    idx = build_component_index(G)
    r = mrpc_dg5_switchback(G, "A", "D", component_index=idx)
    assert not r.found
    assert r.telemetry["error_code"] == "DISCONNECTED_PRECHECK"


def test_route_mode_mrpc_dg5():
    G = Graph.from_edges([
        (1, 2, 1.0),
        (2, 3, 1.0),
        (1, 3, 4.0),
    ])
    r = route(G, 1, 3, mode="mrpc_dg5", constraints={"workers": 2})
    assert r.found


def test_mrpc_dg5_trace_and_cold_preprocessing_metrics():
    G = Graph.from_edges([
        ("A", "B", 1.0),
        ("B", "C", 1.0),
        ("A", "C", 3.0),
    ])
    r = mrpc_dg5_switchback(G, "A", "C", workers=1, min_budget=2, trace_level=2, trace_sample_every=1)
    tel = r.telemetry
    assert tel["preprocessing_work"] > 0
    assert tel["total_work_including_preprocessing"] >= r.total_work + tel["preprocessing_work"]
    assert tel["total_time_ms"] >= tel["preprocessing_time_ms"]
    assert tel["trace_event_count"] > 0
    assert any(e["event"] == "query_start" for e in tel["trace_events"])
    assert any(e["event"] == "mrpc_segment_start" for e in tel["trace_events"])


def test_dg5_adaptive_reentry_reports_selected_candidate():
    from bridge_py.bench.dg5_ablation import wall_gap
    from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback
    G, s, t = wall_gap(400, "top")
    r = mrpc_dg5_switchback(
        G, s, t, workers=1, local_exact_ratio=1.0,
        adaptive_reentry_minimum=True, reentry_candidate_window=32,
        trace_level=1, trace_sample_every=1,
    )
    assert r.found
    assert r.path[0] == s and r.path[-1] == t
    events = (r.telemetry or {}).get("trace_events", [])
    assert any(e.get("event") == "reentry_selected" for e in events)
