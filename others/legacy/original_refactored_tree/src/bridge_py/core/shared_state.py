from __future__ import annotations
import math
from dataclasses import dataclass, field
from typing import Any


@dataclass
class SharedSearchState:
    upper_bound: float = math.inf
    lower_bound: float = 0.0
    incumbent: Any | None = None
    metadata: dict[str, Any] = field(default_factory=dict)

    def consider(self, result: Any) -> bool:
        if getattr(result, 'found', False) and result.distance < self.upper_bound:
            self.upper_bound = float(result.distance)
            self.incumbent = result
            return True
        return False
