from __future__ import annotations
from typing import Mapping, Protocol


class SearchObserver(Protocol):
    def phase_started(self, phase: str, attributes: Mapping[str, object]) -> None: ...
    def phase_finished(self, phase: str, attributes: Mapping[str, object]) -> None: ...
    def step_started(self, logical_step: int, lane: str | None = None) -> None: ...
    def node_expanded(self, event: object) -> None: ...
    def edge_relaxed(self, event: object) -> None: ...
    def neighbor_scored(self, event: object) -> None: ...
    def candidate_found(self, event: object) -> None: ...
    def bound_updated(self, event: object) -> None: ...
    def budget_updated(self, event: object) -> None: ...


class NullObserver:
    def phase_started(self, phase, attributes): return None
    def phase_finished(self, phase, attributes): return None
    def step_started(self, logical_step, lane=None): return None
    def node_expanded(self, event): return None
    def edge_relaxed(self, event): return None
    def neighbor_scored(self, event): return None
    def candidate_found(self, event): return None
    def bound_updated(self, event): return None
    def budget_updated(self, event): return None


class SafeObserver:
    """Exception-swallowing adapter preserving observation non-interference."""
    def __init__(self, delegate: SearchObserver | None = None):
        self.delegate = delegate or NullObserver()

    def __getattr__(self, name):
        target = getattr(self.delegate, name)
        def safe(*args, **kwargs):
            try:
                target(*args, **kwargs)
            except Exception:
                return None
        return safe
