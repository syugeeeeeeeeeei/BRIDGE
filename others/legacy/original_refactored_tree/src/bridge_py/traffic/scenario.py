from __future__ import annotations
from dataclasses import dataclass
from typing import Any
from ..core import RouteRequest


@dataclass(frozen=True)
class TrafficScenario:
    scenario_id: str
    graph_spec: Any
    query_spec: Any
    repetitions: int
    seeds: tuple[int, ...]
    route_request: RouteRequest
    trace_level: str = 'OFF'
    acceptance: Any = None


@dataclass(frozen=True)
class TrafficRunRecord:
    scenario_id: str
    case_id: str
    seed: int
    request: RouteRequest
    result: Any
    baseline: Any = None
    ultrasound_artifact: str | None = None
    environment: Any = None
