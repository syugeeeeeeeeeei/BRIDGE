from __future__ import annotations
from dataclasses import asdict, dataclass
from typing import Any, Mapping

@dataclass(frozen=True)
class SearchEvent:
    task_id: str
    logical_step: int
    lane: str
    phase: str
    def as_dict(self) -> dict[str, Any]: return asdict(self)

@dataclass(frozen=True)
class NodeExpandedEvent(SearchEvent):
    node: Any
    distance: float | None = None
    frontier_size: int | None = None
    work_used: int = 0

@dataclass(frozen=True)
class EdgeRelaxedEvent(SearchEvent):
    source: Any
    target: Any
    weight: float
    old_distance: float | None
    new_distance: float
    improved: bool


@dataclass(frozen=True)
class NeighborScoredEvent(SearchEvent):
    source: Any
    target: Any
    edge_weight: float
    heuristic_to_target: float
    progress: float
    score: float

@dataclass(frozen=True)
class CandidateFoundEvent(SearchEvent):
    found: bool
    distance: float | None
    path_length: int
    solver: str
    strategy: str | None = None

@dataclass(frozen=True)
class BoundUpdatedEvent(SearchEvent):
    lower_bound: float | None = None
    upper_bound: float | None = None
    certified_ratio: float | None = None

@dataclass(frozen=True)
class BudgetUpdatedEvent(SearchEvent):
    max_work: int | None = None
    work_used: int = 0
    portfolio_remaining: int | None = None
