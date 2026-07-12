from __future__ import annotations
from dataclasses import dataclass


@dataclass(frozen=True)
class SolverProgress:
    task_id: str
    work_used: int = 0
    elapsed_ms: float = 0.0
    found: bool = False
    best_distance: float | None = None
    lower_bound: float | None = None
    upper_bound_improvement: float = 0.0
    lower_bound_improvement: float = 0.0
    candidate_count: int = 0
    stagnation_score: float = 0.0
    memory_used: int | None = None
    resumable: bool = False
    finished: bool = True
    failure_reason: str | None = None
