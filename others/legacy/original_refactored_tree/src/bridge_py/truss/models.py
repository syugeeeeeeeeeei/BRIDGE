from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class TaskTrace:
    task_id: str
    solver: str
    purpose: str
    allocation: float
    budget: int | None
    found: bool
    distance: float
    work: int
    continued_reason: str

    def as_dict(self) -> dict[str, object]:
        return {
            "task_id": self.task_id,
            "solver": self.solver,
            "purpose": self.purpose,
            "allocation": self.allocation,
            "budget": self.budget,
            "found": self.found,
            "distance": self.distance,
            "work": self.work,
            "continued_reason": self.continued_reason,
        }
