from __future__ import annotations
from dataclasses import dataclass
from ..core import SolverProgress, WorkBudget
from ..solvers.pier import pier


@dataclass
class PierAnchorSession:
    graph: object
    request: object
    task: object
    observer: object
    _result: object | None = None
    _cancelled: bool = False

    def run_slice(self, budget: WorkBudget) -> SolverProgress:
        if self._cancelled:
            return SolverProgress(self.task.task_id, finished=True, failure_reason='cancelled')
        self.observer.phase_started('anchor', {'task_id': self.task.task_id, 'purpose': self.task.purpose})
        ratio = 0.5
        if budget.max_work is not None:
            ratio = max(0.0, budget.max_work / max(1, len(self.graph.adj)))
        self._result = pier(
            self.graph, self.request.source, self.request.target,
            target_ratio=self.task.quality_target,
            target_work_ratio=ratio,
            workers=self.task.worker_slice,
            fallback_exact=False,
            **self.request.constraints,
        )
        work = self._result.total_work
        self.observer.candidate_found({'task_id': self.task.task_id, 'found': self._result.found, 'distance': self._result.distance})
        self.observer.phase_finished('anchor', {'task_id': self.task.task_id, 'work': work})
        return SolverProgress(
            task_id=self.task.task_id, work_used=work, elapsed_ms=self._result.time_ms,
            found=self._result.found, best_distance=self._result.distance if self._result.found else None,
            lower_bound=self._result.lower_bound, candidate_count=1 if self._result.found else 0,
            resumable=False, finished=True,
            failure_reason='budget_exceeded_by_legacy_solver' if budget.max_work is not None and work > budget.max_work else None,
        )

    def pause(self): return {'finished': self._result is not None}
    def resume(self, state): return None
    def cancel(self): self._cancelled = True
    def result(self): return self._result


class PierAnchor:
    def create_session(self, graph, request, task, observer):
        return PierAnchorSession(graph, request, task, observer)
