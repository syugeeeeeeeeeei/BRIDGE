from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, Optional


@dataclass(frozen=True)
class RouteRequest:
    """Stable BRIDGE query contract.

    ``mode`` is one of fast, balanced, quality, exact.  Legacy solver names are
    accepted by ``bridge_py.route.route`` but are intentionally excluded here.
    """

    source: Any
    target: Any
    mode: str = "balanced"
    max_suboptimality: Optional[float] = None
    deadline_ms: Optional[float] = None
    work_budget: Optional[int] = None
    memory_budget_kib: Optional[float] = None
    workers: int = 1
    seed: int = 0
    constraints: Dict[str, Any] = field(default_factory=dict)

    def validate(self) -> None:
        if self.mode not in {"fast", "balanced", "quality", "exact"}:
            raise ValueError(f"unsupported BRIDGE mode: {self.mode}")
        if self.max_suboptimality is not None and self.max_suboptimality < 1.0:
            raise ValueError("max_suboptimality must be >= 1.0")
        if self.deadline_ms is not None and self.deadline_ms <= 0:
            raise ValueError("deadline_ms must be positive")
        if self.work_budget is not None and self.work_budget < 0:
            raise ValueError("work_budget must be non-negative")
        if self.workers < 1:
            raise ValueError("workers must be >= 1")
