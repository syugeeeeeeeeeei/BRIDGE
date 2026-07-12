from __future__ import annotations

from dataclasses import dataclass, replace

from ..core import SolverProgress, WorkBudget
from ..core.result import PathResult
from .algorithm import run_anchor_algorithm
from .trace import begin as begin_trace, end as end_trace, current_step
from ..bearing.events import CandidateFoundEvent


@dataclass
class AnchorSession:
    """Reference ANCHOR session.

    The legacy PIER algorithm is executed behind the ANCHOR port.  The current
    Python reference keeps the legacy phase ordering intact while exposing its
    result through the new session contract.  True incremental frontier resume
    remains a later optimization; pause/resume is therefore terminal-state only.
    """

    graph: object
    request: object
    task: object
    observer: object
    _result: PathResult | None = None
    _cancelled: bool = False

    def run_slice(self, budget: WorkBudget) -> SolverProgress:
        if self._cancelled:
            return SolverProgress(self.task.task_id, finished=True, failure_reason="cancelled")

        self.observer.phase_started("anchor", {"task_id": self.task.task_id})
        node_count = max(1, len(self.graph.adj))
        kwargs = {
            "target_ratio": self.task.quality_target or 1.10,
            "strategy": self.task.parameters.get("strategy"),
        }
        if budget.max_work is not None:
            kwargs["target_work_ratio"] = max(0.0, budget.max_work / node_count)

        trace_token = begin_trace(self.observer, self.task.task_id)
        try:
            result = run_anchor_algorithm(
                self.graph,
                self.request.source,
                self.request.target,
                **kwargs,
            )
            final_step = current_step()
        finally:
            end_trace(trace_token)
        telemetry = dict(result.telemetry)
        telemetry.update(
            {
                "algorithm_family": "ANCHOR",
                "algorithm_origin": "legacy_PIER_MRPC_DG6",
                "anchor_version": "0.2.0",
                "phase_model": "truss_selected_strategy_then_candidate_then_local_refinement",
                "task_budget": budget.max_work,
            }
        )
        self._result = replace(
            result,
            solver_name="anchor",
            exact=False,
            first_path_work=telemetry.get("first_path_work"),
            budget_exhausted=bool(telemetry.get("budget_exhausted", False)),
            telemetry=telemetry,
        )

        # PIER's planning work is expansion-based.  Use that value for TRUSS
        # accounting instead of edge relaxations, which are separately retained.
        work = int(
            telemetry.get(
                "query_work_units",
                self._result.work_expanded_nodes,
            )
        )
        failure_reason = None
        if budget.max_work is not None and work > budget.max_work:
            failure_reason = "budget_exceeded"

        self.observer.candidate_found(CandidateFoundEvent(
            task_id=self.task.task_id, logical_step=final_step, lane="anchor",
            phase="anchor_search", found=self._result.found,
            distance=self._result.distance if self._result.found else None,
            path_length=len(self._result.path), solver="anchor",
            strategy=telemetry.get("strategy"),
        ))
        self.observer.phase_finished(
            "anchor",
            {"task_id": self.task.task_id, "work": work, "strategy": telemetry.get("strategy")},
        )
        return SolverProgress(
            task_id=self.task.task_id,
            work_used=work,
            elapsed_ms=self._result.time_ms,
            found=self._result.found,
            best_distance=self._result.distance if self._result.found else None,
            candidate_count=int(telemetry.get("candidate_count", 1 if self._result.found else 0)),
            stagnation_score=0.0 if self._result.found else 1.0,
            resumable=False,
            finished=True,
            failure_reason=failure_reason,
        )

    def pause(self):
        return {"finished": self._result is not None, "result": self._result}

    def resume(self, state):
        if isinstance(state, dict) and state.get("finished"):
            self._result = state.get("result")

    def cancel(self):
        self._cancelled = True

    def result(self):
        return self._result


class PierAnchor:
    """ANCHOR port backed by the reconstructed legacy PIER algorithm."""

    name = "anchor"

    def create_session(self, graph, request, task, observer):
        return AnchorSession(graph, request, task, observer)
