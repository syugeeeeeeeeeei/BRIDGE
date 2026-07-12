from __future__ import annotations
from dataclasses import dataclass

@dataclass(frozen=True)
class AcceptanceCriteria:
    require_valid_path: bool = True
    max_distance_ratio: float | None = None
    max_work: int | None = None
    require_budget_compliance: bool = True
    require_trace: bool = False
    require_non_interference: bool = False
