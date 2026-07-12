from __future__ import annotations
from dataclasses import dataclass


@dataclass(frozen=True)
class WorkBudget:
    max_work: int | None = None

    def permits(self, used: int) -> bool:
        return self.max_work is None or used <= self.max_work


@dataclass(frozen=True)
class Deadline:
    deadline_ms: float | None = None
