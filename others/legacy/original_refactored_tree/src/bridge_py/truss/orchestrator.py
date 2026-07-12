from __future__ import annotations

import time
from dataclasses import asdict, dataclass, replace
from typing import Iterable

from ..anchor import PierAnchor
from ..bearing import NullObserver, SafeObserver
from ..bolts import AStarBolt, BidirectionalDijkstraBolt
from ..core import Deadline, SharedSearchState, SolverTask, WorkBudget, make_bounds
from ..core.result import PathResult
from .models import TaskTrace
from .profile import QueryProfile, profile_query

_DEFAULT_TARGETS = {
    "fast": 1.25,
    "balanced": 1.08,
    "quality": 1.03,
    "exact": 1.0,
}


@dataclass
class _ExecutionState:
    started_at: float
    total_limit: int | None
    shared: SharedSearchState
    traces: list[TaskTrace]
    used_work: int = 0
    deadline_hit: bool = False

    def remaining_work(self) -> int | None:
        if self.total_limit is None:
            return None
        return max(0, self.total_limit - self.used_work)


class Truss:
    """Owns portfolio planning, budget accounting, and solver orchestration."""

    def __init__(self, anchor=None, bolts: Iterable[object] | None = None, observer=None) -> None:
        self._anchor = anchor or PierAnchor()
        configured_bolts = bolts or (BidirectionalDijkstraBolt(), AStarBolt())
        self._bolts = {bolt.name: bolt for bolt in configured_bolts}
        self._observer = SafeObserver(observer or NullObserver())

    def route(self, graph, request) -> PathResult:
        request.validate()
        profile = profile_query(graph, request.source, request.target)
        target_ratio = request.max_suboptimality or _DEFAULT_TARGETS[request.mode]
        state = _ExecutionState(
            started_at=time.perf_counter(),
            total_limit=request.work_budget,
            shared=SharedSearchState(),
            traces=[],
        )

        if request.mode == "exact":
            result = self._run_exact(graph, request, target_ratio, state)
        else:
            result = self._run_anytime(graph, request, profile, target_ratio, state)

        return self._finalize(graph, request, profile, target_ratio, state, result)

    def _run_exact(self, graph, request, target_ratio: float, state: _ExecutionState) -> PathResult:
        result = self._run_task(
            graph,
            request,
            target_ratio,
            state,
            port=self._bolts["bidirectional_dijkstra"],
            task_id="bolt-exact-0",
            purpose="certification",
            allocation=1.0,
            reason="exact_mode",
            budget=state.remaining_work(),
        )
        return result or PathResult(
            [],
            float("inf"),
            False,
            False,
            "bridge_exact",
            error_code="deadline_exceeded",
        )

    def _run_anytime(
        self,
        graph,
        request,
        profile: QueryProfile,
        target_ratio: float,
        state: _ExecutionState,
    ) -> PathResult:
        anchor_budget = self._anchor_budget(request.mode, profile, state.remaining_work())
        anchor_result = self._run_task(
            graph,
            request,
            target_ratio,
            state,
            port=self._anchor,
            task_id="anchor-0",
            purpose="first_path",
            allocation=0.60,
            reason="initial_probe",
            budget=anchor_budget,
        )
        candidates = [anchor_result] if anchor_result is not None and anchor_result.found else []

        if self._should_run_exact(request, anchor_result, candidates, state):
            exact_result = self._run_task(
                graph,
                request,
                target_ratio,
                state,
                port=self._bolts["bidirectional_dijkstra"],
                task_id="bolt-exact-1",
                purpose="fallback" if not candidates else "certification",
                allocation=0.40,
                reason="quality_or_recovery",
                budget=state.remaining_work(),
            )
            if exact_result is not None and exact_result.found:
                candidates.append(exact_result)

        if not candidates and self._can_continue(request, state):
            astar_result = self._run_task(
                graph,
                request,
                target_ratio,
                state,
                port=self._bolts["astar"],
                task_id="bolt-astar-2",
                purpose="fallback",
                allocation=0.20,
                reason="last_recovery",
                budget=state.remaining_work(),
            )
            if astar_result is not None and astar_result.found:
                candidates.append(astar_result)

        if candidates:
            return min(candidates, key=lambda candidate: candidate.distance)
        return anchor_result or PathResult([], float("inf"), False, False, "bridge")

    def _run_task(
        self,
        graph,
        request,
        target_ratio: float,
        state: _ExecutionState,
        *,
        port,
        task_id: str,
        purpose: str,
        allocation: float,
        reason: str,
        budget: int | None,
    ) -> PathResult | None:
        if self._deadline_exceeded(request, state):
            state.deadline_hit = True
            return None

        task = SolverTask(
            task_id=task_id,
            solver_kind=getattr(port, "name", "anchor"),
            purpose=purpose,
            budget_slice=WorkBudget(budget),
            deadline_slice=Deadline(request.deadline_ms),
            worker_slice=request.workers,
            quality_target=target_ratio,
            shared_state=state.shared,
            resumable=False,
        )
        self._observer.budget_updated(
            {
                "task_id": task_id,
                "max_work": budget,
                "portfolio_remaining": state.remaining_work(),
            }
        )
        session = port.create_session(graph, request, task, self._observer)
        progress = session.run_slice(task.budget_slice)
        result = session.result()
        if result is None:
            return None

        state.used_work += progress.work_used
        state.shared.consider(result)
        state.traces.append(
            TaskTrace(
                task_id=task_id,
                solver=getattr(port, "name", result.solver_name),
                purpose=purpose,
                allocation=allocation,
                budget=budget,
                found=result.found,
                distance=result.distance,
                work=result.total_work,
                continued_reason=reason,
            )
        )
        return result

    def _finalize(
        self,
        graph,
        request,
        profile: QueryProfile,
        target_ratio: float,
        state: _ExecutionState,
        best: PathResult,
    ) -> PathResult:
        bounds = make_bounds(graph, request.source, request.target, best.distance, target_ratio)
        if best.exact and best.found:
            bounds = replace(
                bounds,
                lower_bound=best.distance,
                certified_ratio=1.0,
                quality_certified=True,
                method="exact_solver",
            )

        elapsed_ms = (time.perf_counter() - state.started_at) * 1000
        trace_dicts = [trace.as_dict() for trace in state.traces]
        deadline_exceeded = state.deadline_hit or self._deadline_exceeded(request, state)
        budget_violation = (
            state.total_limit is not None and state.used_work > state.total_limit
        )
        exact_mode = request.mode == "exact"

        telemetry = dict(best.telemetry)
        telemetry.update(
            {
                "truss_version": "0.2.0",
                "cable_version": "0.1.0",
                "deprecated_component": "CABLE compatibility metadata",
                "architecture": "TRUSS/ANCHOR/BOLTS/BEARING/GATE",
                "portfolio_execution": "serial_reference",
                "solver_trace": trace_dicts,
                "query_profile": asdict(profile),
                "shared_upper_bound": state.shared.upper_bound,
                "portfolio_work_used": state.used_work,
                "portfolio_work_budget": state.total_limit,
                "budget_violation": budget_violation,
                "deadline_ms": request.deadline_ms,
                "deadline_exceeded": deadline_exceeded,
                "selected_solver": best.solver_name,
                "requested_mode": request.mode,
                "requested_max_suboptimality": target_ratio,
            }
        )
        return replace(
            best,
            solver_name="bridge_exact" if exact_mode else f"bridge_{request.mode}",
            work_expanded_nodes=state.used_work,
            time_ms=elapsed_ms,
            lower_bound=best.distance if exact_mode and best.found else bounds.lower_bound,
            certified_ratio=1.0 if exact_mode and best.found else bounds.certified_ratio,
            quality_certified=best.found if exact_mode else bounds.quality_certified,
            solver_trace=trace_dicts,
            fallback_used=len(trace_dicts) > 1,
            budget_exhausted=(
                state.total_limit is not None and state.used_work >= state.total_limit
            ),
            deadline_exceeded=deadline_exceeded,
            telemetry=telemetry,
        )

    @staticmethod
    def _anchor_budget(mode: str, profile: QueryProfile, remaining: int | None) -> int:
        if remaining is None:
            ratio = 0.30 if mode == "fast" else 0.50
            return max(1, int(profile.nodes * ratio))
        if mode == "quality":
            return max(0, int(remaining * 0.60))
        return remaining

    def _should_run_exact(self, request, anchor_result, candidates, state: _ExecutionState) -> bool:
        should_run = request.mode == "quality" or not candidates
        if anchor_result is not None and (
            anchor_result.quality_certified or request.mode == "fast"
        ):
            should_run = False
        if not self._can_continue(request, state):
            should_run = False
        return should_run

    def _can_continue(self, request, state: _ExecutionState) -> bool:
        if self._deadline_exceeded(request, state):
            state.deadline_hit = True
            return False
        remaining = state.remaining_work()
        return remaining is None or remaining > 0

    @staticmethod
    def _deadline_exceeded(request, state: _ExecutionState) -> bool:
        if request.deadline_ms is None:
            return False
        elapsed_ms = (time.perf_counter() - state.started_at) * 1000
        return elapsed_ms >= request.deadline_ms
