from __future__ import annotations
import time
from dataclasses import asdict, dataclass, replace
from typing import Iterable
from ..anchor import PierAnchor
from ..bearing import NullObserver, SafeObserver
from ..bolts import AStarBolt, BidirectionalDijkstraBolt, EmergencyApproxBolt, ReachabilityBolt
from ..core import Deadline, SharedSearchState, SolverTask, WorkBudget, make_bounds
from ..core.result import PathResult
from ..bearing.events import BoundUpdatedEvent, BudgetUpdatedEvent
from .models import TaskTrace
from .profile import QueryProfile, profile_query
from .strategy import AnchorPlan, make_anchor_plan

_DEFAULT_TARGETS={"fast":1.25,"balanced":1.08,"quality":1.03,"exact":1.0}

@dataclass
class _ExecutionState:
    started_at: float; total_limit: int|None; shared: SharedSearchState; traces:list[TaskTrace]
    used_work:int=0; deadline_hit:bool=False
    def remaining_work(self):
        return None if self.total_limit is None else max(0,self.total_limit-self.used_work)

class Truss:
    """Single owner of planning, portfolio budget, fallback, certification and selection."""
    def __init__(self,anchor=None,bolts:Iterable[object]|None=None,observer=None):
        self._anchor=anchor or PierAnchor()
        configured=bolts or (BidirectionalDijkstraBolt(),AStarBolt(),ReachabilityBolt(),EmergencyApproxBolt())
        self._bolts={b.name:b for b in configured}; self._observer=SafeObserver(observer or NullObserver())

    def route(self,graph,request)->PathResult:
        request.validate(); profile=profile_query(graph,request.source,request.target)
        target=request.max_suboptimality or _DEFAULT_TARGETS[request.mode]
        state=_ExecutionState(time.perf_counter(),request.work_budget,SharedSearchState(),[])
        plan=make_anchor_plan(graph,request.source,request.target)
        result=self._run_exact(graph,request,target,state) if request.mode=="exact" else self._run_anytime(graph,request,profile,plan,target,state)
        return self._finalize(graph,request,profile,plan,target,state,result)

    def _run_exact(self,g,r,target,state):
        return self._run_task(g,r,target,state,self._bolts["bidirectional_dijkstra"],"bolt-exact-0","certification",1.0,"exact_mode",state.remaining_work()) or PathResult([],float("inf"),False,False,"bridge_exact",error_code="budget_or_deadline_exhausted")

    def _run_anytime(self,g,r,profile,plan,target,state):
        candidates=[]
        initial=self._anchor_budget(r.mode,profile,state.remaining_work())
        primary=self._run_task(g,r,target,state,self._anchor,"anchor-primary","first_path",.55,plan.reason,initial,{"strategy":plan.strategy})
        if primary and primary.found: candidates.append(primary)

        # TRUSS alone decides whether another ANCHOR hypothesis deserves budget.
        if not candidates and self._can_continue(r,state):
            for i,strategy in enumerate(plan.alternates[:2]):
                remaining=state.remaining_work()
                if remaining == 0: break
                slice_budget=remaining if remaining is not None else max(1,int(profile.nodes*.20))
                alt=self._run_task(g,r,target,state,self._anchor,f"anchor-alt-{i}","hypothesis_probe",.15,"primary_stagnated",slice_budget,{"strategy":strategy})
                if alt and alt.found:
                    candidates.append(alt); break

        # Reachability is a BOLTS concern and runs only when ANCHOR produced no route.
        if not candidates and self._can_continue(r,state):
            reach=self._run_task(g,r,target,state,self._bolts["reachability"],"bolt-reachability","reachability",.10,"anchor_no_candidate",state.remaining_work())
            if reach and not reach.found and reach.exact:
                return PathResult([],float("inf"),False,True,"reachability",telemetry={"component_reachable":False})

        # Recovery is an explicit BOLTS task; no soft budget overrun is permitted.
        if not candidates and self._can_continue(r,state):
            recovery=self._run_task(g,r,target,state,self._bolts["emergency_approx"],"bolt-recovery","fallback",.20,"reachable_but_anchor_failed",state.remaining_work())
            if recovery and recovery.found: candidates.append(recovery)

        # Quality certification is owned by TRUSS+BOLTS, never ANCHOR.
        if r.mode=="quality" and self._can_continue(r,state):
            exact=self._run_task(g,r,target,state,self._bolts["bidirectional_dijkstra"],"bolt-certify","certification",.40,"quality_mode",state.remaining_work())
            if exact and exact.found: candidates.append(exact)
        return min(candidates,key=lambda x:x.distance) if candidates else (primary or PathResult([],float("inf"),False,False,"bridge"))

    def _run_task(self,g,r,target,state,port,task_id,purpose,allocation,reason,budget,parameters=None):
        if self._deadline_exceeded(r,state) or budget==0:
            state.deadline_hit=self._deadline_exceeded(r,state); return None
        task=SolverTask(task_id,getattr(port,"name","anchor"),purpose,WorkBudget(budget),Deadline(r.deadline_ms),r.workers,target,state.shared,False,parameters or {})
        self._observer.budget_updated(BudgetUpdatedEvent(task_id,0,"truss","task_scheduling",max_work=budget,work_used=state.used_work,portfolio_remaining=state.remaining_work()))
        session=port.create_session(g,r,task,self._observer); progress=session.run_slice(task.budget_slice); result=session.result()
        used=progress.work_used
        if budget is not None: used=min(used,budget)
        state.used_work+=used
        if result is not None:
            state.shared.consider(result)
            self._observer.bound_updated(BoundUpdatedEvent(task_id,0,"truss","portfolio",upper_bound=result.distance if result.found else None))
            state.traces.append(TaskTrace(task_id,getattr(port,"name",result.solver_name),purpose,allocation,budget,result.found,result.distance,used,reason))
        return result

    def _finalize(self,g,r,profile,plan,target,state,best):
        bounds=make_bounds(g,r.source,r.target,best.distance,target)
        if best.exact and best.found: bounds=replace(bounds,lower_bound=best.distance,certified_ratio=1.0,quality_certified=True,method="exact_solver")
        elapsed=(time.perf_counter()-state.started_at)*1000; traces=[t.as_dict() for t in state.traces]
        deadline=state.deadline_hit or self._deadline_exceeded(r,state)
        violation=state.total_limit is not None and state.used_work>state.total_limit
        tel=dict(best.telemetry); tel.update({"truss_version":"0.3.0","architecture":"TRUSS/ANCHOR/BOLTS/BEARING/GATE","solver_trace":traces,"query_profile":asdict(profile),"anchor_plan":asdict(plan),"shared_upper_bound":state.shared.upper_bound,"portfolio_work_used":state.used_work,"portfolio_work_budget":state.total_limit,"budget_violation":violation,"deadline_exceeded":deadline,"selected_solver":best.solver_name,"requested_mode":r.mode,"algorithm_origin":"legacy_PIER_MRPC_DG6_refactored","responsibility_split":{"planning":"TRUSS","primary_search":"ANCHOR","fallback_and_certification":"BOLTS"}})
        exact_mode=r.mode=="exact"
        return replace(best,solver_name="bridge_exact" if exact_mode else f"bridge_{r.mode}",work_expanded_nodes=state.used_work,time_ms=elapsed,lower_bound=best.distance if best.exact and best.found else bounds.lower_bound,certified_ratio=1.0 if best.exact and best.found else bounds.certified_ratio,quality_certified=best.exact and best.found,solver_trace=traces,fallback_used=any(t.purpose in {"fallback","certification","reachability"} for t in state.traces[1:]),budget_exhausted=state.total_limit is not None and state.used_work>=state.total_limit,deadline_exceeded=deadline,telemetry=tel)

    @staticmethod
    def _anchor_budget(mode,profile,remaining):
        if remaining is None: return max(1,int(profile.nodes*(.30 if mode=="fast" else .50)))
        return max(0,int(remaining*(.60 if mode=="quality" else 1.0)))
    def _can_continue(self,r,state):
        if self._deadline_exceeded(r,state): state.deadline_hit=True; return False
        rem=state.remaining_work(); return rem is None or rem>0
    @staticmethod
    def _deadline_exceeded(r,state):
        return False if r.deadline_ms is None else (time.perf_counter()-state.started_at)*1000>=r.deadline_ms
