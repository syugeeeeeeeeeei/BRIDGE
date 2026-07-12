from __future__ import annotations
from dataclasses import dataclass
from .budget import Deadline, WorkBudget
from .shared_state import SharedSearchState


@dataclass(frozen=True)
class SolverTask:
    task_id: str
    solver_kind: str
    purpose: str
    budget_slice: WorkBudget
    deadline_slice: Deadline
    worker_slice: int
    quality_target: float | None
    shared_state: SharedSearchState
    resumable: bool = False
