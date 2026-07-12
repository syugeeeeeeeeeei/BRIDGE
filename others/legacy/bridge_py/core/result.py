from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, Hashable, List, Mapping, Optional, Sequence, Tuple

Node = Hashable
Adjacency = Mapping[Node, Sequence[Tuple[Node, float]]]
Point = Tuple[float, float]


@dataclass(frozen=True)
class PathResult:
    path: List[Node]
    distance: float
    found: bool
    exact: bool
    solver_name: str
    work_relaxations: int = 0
    work_expanded_nodes: int = 0
    queue_pushes: int = 0
    queue_pops: int = 0
    parallel_steps: int = 0
    time_ms: float = 0.0
    peak_memory_kib: float = 0.0
    telemetry: Dict[str, Any] = field(default_factory=dict)
    lower_bound: float = 0.0
    certified_ratio: Optional[float] = None
    quality_certified: bool = False
    first_path_work: Optional[int] = None
    first_path_time_ms: Optional[float] = None
    solver_trace: List[Dict[str, Any]] = field(default_factory=list)
    fallback_used: bool = False
    budget_exhausted: bool = False
    deadline_exceeded: bool = False
    error_code: Optional[str] = None

    @property
    def total_work(self) -> int:
        return int(self.work_relaxations + self.telemetry.get("work_candidate_expansions", 0) + self.telemetry.get("work_repair", 0))


@dataclass(frozen=True)
class Corridor:
    corridor_id: str
    width: float
    offset: float
    nodes: List[Node]


@dataclass(frozen=True)
class BenchmarkRow:
    run_id: str
    experiment_id: str
    trial: int
    seed: int
    graph_type: str
    nodes: int
    edges: int
    source: Node
    target: Node
    query_class: str
    solver_name: str
    found: bool
    distance: float
    exact_distance: float
    distance_ratio: float
    exact_match: bool
    work_relaxations: int
    work_expanded_nodes: int
    total_work: int
    parallel_steps: int
    time_ms: float
    peak_memory_kib: float
    k_corridors: int = 0
    candidate_count: int = 0
    best_corridor_id: str = ""
    rescue_triggered: bool = False
    repair_triggered: bool = False
    error_code: Optional[str] = None
