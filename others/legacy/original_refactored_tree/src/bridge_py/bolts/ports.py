from __future__ import annotations
from dataclasses import dataclass
from typing import Protocol
from ..core import SolverProgress, SolverTask, WorkBudget
from ..types import PathResult


@dataclass(frozen=True)
class SolverCapabilities:
    exact: bool
    resumable: bool = False
    supports_local_detour: bool = False
    provides_lower_bound: bool = False
    supports_coordinates: bool = False
    supports_constraints: bool = False
    parallel_capable: bool = False
    estimated_memory_class: str = 'medium'


class BoltSession(Protocol):
    def run_slice(self, budget: WorkBudget) -> SolverProgress: ...
    def pause(self) -> object: ...
    def resume(self, state: object) -> None: ...
    def cancel(self) -> None: ...
    def result(self) -> PathResult | None: ...


class BoltSolverPort(Protocol):
    name: str
    capabilities: SolverCapabilities
    def create_session(self, graph, request, task: SolverTask, observer) -> BoltSession: ...
