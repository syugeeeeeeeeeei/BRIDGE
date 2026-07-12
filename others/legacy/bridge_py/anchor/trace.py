from __future__ import annotations
from contextvars import ContextVar
from dataclasses import dataclass
from typing import Any
from ..bearing.events import EdgeRelaxedEvent, NeighborScoredEvent, NodeExpandedEvent

@dataclass
class _TraceState:
    observer: Any
    task_id: str
    phase: str
    lane: str = "anchor"
    logical_step: int = 0

_STATE: ContextVar[_TraceState | None] = ContextVar("bridge_anchor_trace", default=None)

def begin(observer: Any, task_id: str, phase: str = "anchor_search"):
    return _STATE.set(_TraceState(observer, task_id, phase))

def end(token) -> None: _STATE.reset(token)

def expanded(node: Any, distance: float | None, frontier_size: int | None, work_used: int) -> None:
    s=_STATE.get()
    if s is None: return
    s.logical_step += 1
    s.observer.step_started(s.logical_step, s.lane)
    # The trace contract defines work_used as task-cumulative node expansions.
    s.observer.node_expanded(NodeExpandedEvent(s.task_id,s.logical_step,s.lane,s.phase,node,distance,frontier_size,s.logical_step))

def relaxed(source: Any,target: Any,weight: float,old_distance: float|None,new_distance: float,improved: bool) -> None:
    s=_STATE.get()
    if s is None: return
    s.observer.edge_relaxed(EdgeRelaxedEvent(s.task_id,s.logical_step,s.lane,s.phase,source,target,float(weight),old_distance,new_distance,improved))

def current_step() -> int:
    s=_STATE.get(); return 0 if s is None else s.logical_step


def neighbor_scored(source: Any, target: Any, edge_weight: float, heuristic_to_target: float, progress: float, score: float) -> None:
    s = _STATE.get()
    if s is None: return
    s.observer.neighbor_scored(NeighborScoredEvent(
        s.task_id, s.logical_step, s.lane, s.phase, source, target,
        float(edge_weight), float(heuristic_to_target), float(progress), float(score)
    ))
