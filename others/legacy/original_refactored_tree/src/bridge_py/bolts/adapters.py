from __future__ import annotations
from dataclasses import dataclass
from ..core import SolverProgress, WorkBudget
from ..solvers.astar import astar
from ..solvers.dijkstra import bidirectional_dijkstra
from .ports import SolverCapabilities


@dataclass
class OneShotBoltSession:
    graph: object
    request: object
    task: object
    observer: object
    function: object
    name: str
    _result: object | None = None
    _cancelled: bool = False

    def run_slice(self, budget: WorkBudget) -> SolverProgress:
        if self._cancelled:
            return SolverProgress(self.task.task_id, failure_reason='cancelled')
        self.observer.phase_started('bolt', {'task_id': self.task.task_id, 'solver': self.name, 'purpose': self.task.purpose})
        self._result = self.function(self.graph, self.request.source, self.request.target)
        work = self._result.total_work
        self.observer.candidate_found({'task_id': self.task.task_id, 'solver': self.name, 'found': self._result.found, 'distance': self._result.distance})
        self.observer.phase_finished('bolt', {'task_id': self.task.task_id, 'solver': self.name, 'work': work})
        return SolverProgress(self.task.task_id, work, self._result.time_ms, self._result.found,
                              self._result.distance if self._result.found else None,
                              self._result.distance if self._result.exact and self._result.found else self._result.lower_bound,
                              candidate_count=1 if self._result.found else 0, finished=True,
                              failure_reason='budget_exceeded_by_legacy_solver' if budget.max_work is not None and work > budget.max_work else None)

    def pause(self): return {'finished': self._result is not None}
    def resume(self, state): return None
    def cancel(self): self._cancelled = True
    def result(self): return self._result


class BidirectionalDijkstraBolt:
    name = 'bidirectional_dijkstra'
    capabilities = SolverCapabilities(exact=True, provides_lower_bound=True)
    def create_session(self, graph, request, task, observer):
        return OneShotBoltSession(graph, request, task, observer, bidirectional_dijkstra, self.name)


class AStarBolt:
    name = 'astar'
    capabilities = SolverCapabilities(exact=True, provides_lower_bound=True, supports_coordinates=True)
    def create_session(self, graph, request, task, observer):
        return OneShotBoltSession(graph, request, task, observer, astar, self.name)
