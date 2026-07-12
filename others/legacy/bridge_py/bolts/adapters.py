from __future__ import annotations
import heapq, math, time
from collections import deque
from dataclasses import dataclass
from ..core import SolverProgress, WorkBudget
from ..core.graph import euclidean
from ..core.result import PathResult
from .ports import SolverCapabilities
from ..bearing.events import CandidateFoundEvent

def _reconstruct(previous, source, target):
    if source == target: return [source]
    if target not in previous: return []
    path=[target]; cur=target
    while cur != source:
        cur=previous[cur]; path.append(cur)
    path.reverse(); return path

def _search(graph, source, target, max_work=None, heuristic=None, exact=True, name="dijkstra"):
    started=time.perf_counter()
    if source not in graph.adj or target not in graph.adj:
        return PathResult([],math.inf,False,exact,name)
    h=heuristic or (lambda _:0.0)
    dist={source:0.0}; prev={}; queue=[(h(source),0,source)]; settled=set(); counter=1
    relax=pushes=pops=0; exhausted=False
    while queue:
        if max_work is not None and len(settled) >= max_work:
            exhausted=True; break
        _,_,u=heapq.heappop(queue); pops+=1
        if u in settled: continue
        settled.add(u)
        if u==target: break
        for v,w in graph.adj.get(u,[]):
            relax+=1; cand=dist[u]+w
            if cand < dist.get(v,math.inf):
                dist[v]=cand; prev[v]=u
                heapq.heappush(queue,(cand+h(v),counter,v)); counter+=1; pushes+=1
    path=_reconstruct(prev,source,target)
    return PathResult(path,dist.get(target,math.inf),bool(path),exact and not exhausted,name,
        relax,len(settled),pushes,pops,len(settled),(time.perf_counter()-started)*1000,
        telemetry={"budget_exhausted":exhausted})

def _dijkstra(graph, source, target, max_work=None):
    return _search(graph,source,target,max_work=max_work,name="dijkstra")

def _astar(graph, source, target, max_work=None):
    def h(n):
        if graph.pos and n in graph.pos and target in graph.pos:
            return euclidean(graph.pos[n],graph.pos[target])
        return 0.0
    # Exactness is conditional; generic weighted graphs are not assumed admissible.
    return _search(graph,source,target,max_work=max_work,heuristic=h,exact=False,name="astar")

def _reachability(graph, source, target, max_work=None):
    started=time.perf_counter()
    if source not in graph.adj or target not in graph.adj:
        return PathResult([],math.inf,False,True,"reachability")
    q=deque([source]); seen={source}; prev={}; exhausted=False
    while q:
        if max_work is not None and len(seen) > max_work:
            exhausted=True; break
        u=q.popleft()
        if u==target: break
        for v,_ in graph.adj.get(u,[]):
            if v not in seen: seen.add(v); prev[v]=u; q.append(v)
    path=_reconstruct(prev,source,target)
    return PathResult(path,0.0 if path else math.inf,bool(path),not exhausted,"reachability",
        work_expanded_nodes=min(len(seen),max_work) if max_work is not None else len(seen),
        time_ms=(time.perf_counter()-started)*1000,
        telemetry={"reachable":bool(path),"budget_exhausted":exhausted})

@dataclass
class OneShotBoltSession:
    graph: object; request: object; task: object; observer: object; function: object; name: str
    _result: PathResult|None=None; _cancelled: bool=False
    def run_slice(self,budget:WorkBudget)->SolverProgress:
        if self._cancelled: return SolverProgress(self.task.task_id,finished=True,failure_reason="cancelled")
        self.observer.phase_started("bolt",{"task_id":self.task.task_id,"solver":self.name})
        self._result=self.function(self.graph,self.request.source,self.request.target,budget.max_work)
        work=self._result.work_expanded_nodes
        self.observer.candidate_found(CandidateFoundEvent(
            task_id=self.task.task_id, logical_step=0, lane="bolt", phase="bolt_search",
            found=self._result.found,
            distance=self._result.distance if self._result.found else None,
            path_length=len(self._result.path), solver=self.name, strategy=None,
        ))
        self.observer.phase_finished("bolt",{"task_id":self.task.task_id,"work":work})
        return SolverProgress(task_id=self.task.task_id,work_used=work,elapsed_ms=self._result.time_ms,
            found=self._result.found,best_distance=self._result.distance if self._result.found else None,
            lower_bound=self._result.distance if self._result.exact and self._result.found else None,
            candidate_count=1 if self._result.found else 0,finished=True,
            failure_reason="budget_exhausted" if self._result.telemetry.get("budget_exhausted") else None)
    def pause(self): return {"finished":self._result is not None}
    def resume(self,state): return None
    def cancel(self): self._cancelled=True
    def result(self): return self._result

class DijkstraBolt:
    name="dijkstra"; capabilities=SolverCapabilities(exact=True,provides_lower_bound=True)
    def create_session(self,g,r,t,o): return OneShotBoltSession(g,r,t,o,_dijkstra,self.name)
class BidirectionalDijkstraBolt(DijkstraBolt):
    # Compatibility registration until a true bidirectional implementation is introduced.
    name="bidirectional_dijkstra"
class AStarBolt:
    name="astar"; capabilities=SolverCapabilities(exact=False,supports_coordinates=True)
    def create_session(self,g,r,t,o): return OneShotBoltSession(g,r,t,o,_astar,self.name)
class ReachabilityBolt:
    name="reachability"; capabilities=SolverCapabilities(exact=True)
    def create_session(self,g,r,t,o): return OneShotBoltSession(g,r,t,o,_reachability,self.name)
class EmergencyApproxBolt:
    name="emergency_approx"; capabilities=SolverCapabilities(exact=False,supports_coordinates=True)
    def create_session(self,g,r,t,o): return OneShotBoltSession(g,r,t,o,_astar,self.name)
