from __future__ import annotations
from dataclasses import dataclass


@dataclass
class CancellationToken:
    cancelled: bool = False
    reason: str | None = None

    def cancel(self, reason: str | None = None) -> None:
        self.cancelled = True
        self.reason = reason
