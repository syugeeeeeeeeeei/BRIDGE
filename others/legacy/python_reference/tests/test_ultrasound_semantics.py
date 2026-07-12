from __future__ import annotations

import copy
import pytest

from bridge_py import RouteRequest
from bridge_py.gate import Gate
from bridge_py.truss import Truss
from bridge_py.traffic import grid_graph
from bridge_py.ultrasound import (
    COMMON_FIELDS,
    EVENT_SEMANTICS,
    InMemoryObserver,
    TRACE_SCHEMA_VERSION,
    validate_trace,
)


def _trace():
    observer = InMemoryObserver()
    result = Gate(Truss(observer=observer)).route_request(
        grid_graph(7, 7, seed=4, noise=.05),
        RouteRequest(0, 48, mode="balanced", work_budget=128),
    )
    assert result.found
    return observer, result


def test_semantic_registry_covers_every_persisted_event_and_field():
    observer, _ = _trace()
    assert "sequence" in COMMON_FIELDS and "relative_ns" in COMMON_FIELDS
    for record in observer.events:
        assert record["schema_version"] == TRACE_SCHEMA_VERSION
        assert record["kind"] in EVENT_SEMANTICS
        payload = record.get("event", record)
        for field in EVENT_SEMANTICS[record["kind"]]:
            assert field in payload or field in record


def test_real_trace_is_semantically_valid_and_work_has_defined_meaning(tmp_path):
    observer, result = _trace()
    report = observer.validate()
    assert report.valid, report.errors
    expanded = [e["event"] for e in observer.events if e["kind"] == "node_expanded"]
    assert expanded
    # node_expanded.work_used is task-local cumulative expansion work.
    by_task = {}
    for event in expanded:
        by_task.setdefault(event["task_id"], []).append(event["work_used"])
    assert all(values == sorted(values) for values in by_task.values())
    assert len(expanded) <= result.work_expanded_nodes
    artifact = observer.write_jsonl(tmp_path / "trace.jsonl")
    restored = InMemoryObserver.read_jsonl(artifact)
    assert restored.validate().valid


def test_validator_rejects_false_relaxation_meaning():
    observer, _ = _trace()
    damaged = copy.deepcopy(observer.events)
    event = next(e for e in damaged if e["kind"] == "edge_relaxed")
    event["event"]["improved"] = not event["event"]["improved"]
    report = validate_trace(damaged)
    assert not report.valid
    assert any("improved disagrees" in error for error in report.errors)


def test_validator_rejects_sequence_and_schema_corruption():
    observer, _ = _trace()
    damaged = copy.deepcopy(observer.events)
    damaged[0]["sequence"] = 9
    damaged[0]["schema_version"] = "0.0.0"
    report = validate_trace(damaged)
    assert not report.valid
    assert any("sequence" in error for error in report.errors)
    assert any("schema_version" in error for error in report.errors)


def test_jsonl_refuses_semantically_invalid_artifact(tmp_path):
    observer, _ = _trace()
    observer.events[0]["sequence"] = 2
    with pytest.raises(ValueError, match="Invalid ULTRASOUND trace"):
        observer.write_jsonl(tmp_path / "invalid.jsonl")
