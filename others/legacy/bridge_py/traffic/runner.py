from __future__ import annotations
import json, math, platform, statistics, sys, time
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Callable, Iterable
from ..gate import Gate
from ..truss import Truss
from ..ultrasound import InMemoryObserver
from .acceptance import AcceptanceCriteria
from .scenario import TrafficRunRecord, TrafficScenario

@dataclass(frozen=True)
class TrafficVerdict:
    passed: bool
    failures: tuple[str,...]
    metrics: dict

class TrafficRunner:
    def __init__(self, output_dir:str|Path): self.output_dir=Path(output_dir)
    def run(self,scenario:TrafficScenario,graph_factory:Callable[[int],object],baseline_factory:Callable[[object,object],object]|None=None)->tuple[list[TrafficRunRecord],TrafficVerdict]:
        self.output_dir.mkdir(parents=True,exist_ok=True); records=[]; failures=[]
        criteria=scenario.acceptance or AcceptanceCriteria()
        seeds=scenario.seeds or tuple(range(scenario.repetitions))
        for i,seed in enumerate(seeds[:scenario.repetitions]):
            graph=graph_factory(seed); observer=InMemoryObserver() if scenario.trace_level!='OFF' else None
            gate=Gate(Truss(observer=observer))
            started=time.perf_counter(); result=gate.route_request(graph,scenario.route_request); wall_ms=(time.perf_counter()-started)*1000
            baseline=baseline_factory(graph,scenario.route_request) if baseline_factory else None
            artifact=None
            if observer is not None:
                artifact=observer.write_jsonl(self.output_dir/f'{scenario.scenario_id}-{i}-{seed}.jsonl')
            record=TrafficRunRecord(scenario.scenario_id,f'{scenario.scenario_id}-{i}',seed,scenario.route_request,result,baseline,artifact,self.environment())
            records.append(record)
            failures.extend(self._judge(record,criteria,observer,wall_ms))
        metrics=self._aggregate(records)
        verdict=TrafficVerdict(not failures,tuple(failures),metrics)
        self._write_manifest(scenario,records,verdict)
        return records,verdict

    def compare_observation(self,graph,request)->TrafficVerdict:
        off=Gate(Truss()).route_request(graph,request)
        obs=InMemoryObserver(); on=Gate(Truss(observer=obs)).route_request(graph,request)
        fields=('path','distance','found','work_expanded_nodes','solver_trace','exact')
        failures=[f'non_interference:{f}' for f in fields if getattr(off,f)!=getattr(on,f)]
        if not obs.events: failures.append('trace_empty')
        return TrafficVerdict(not failures,tuple(failures),{'off_work':off.work_expanded_nodes,'on_work':on.work_expanded_nodes,**obs.metrics()})

    @staticmethod
    def _judge(record,criteria,observer,wall_ms):
        r=record.result; failures=[]
        if criteria.require_valid_path and r.found and (not r.path or r.path[0]!=record.request.source or r.path[-1]!=record.request.target): failures.append(f'{record.case_id}:invalid_path')
        if criteria.max_work is not None and r.work_expanded_nodes>criteria.max_work: failures.append(f'{record.case_id}:work>{criteria.max_work}')
        if criteria.require_budget_compliance and r.telemetry.get('budget_violation'): failures.append(f'{record.case_id}:budget_violation')
        if criteria.max_distance_ratio is not None and record.baseline and record.baseline.found and r.found and r.distance/record.baseline.distance>criteria.max_distance_ratio: failures.append(f'{record.case_id}:distance_ratio')
        if criteria.require_trace and (observer is None or not observer.events): failures.append(f'{record.case_id}:trace_missing')
        return failures

    @staticmethod
    def _aggregate(records):
        distances=[r.result.distance for r in records if r.result.found and math.isfinite(r.result.distance)]
        works=[r.result.work_expanded_nodes for r in records]
        return {'runs':len(records),'found_rate':sum(r.result.found for r in records)/max(1,len(records)),'mean_distance':statistics.fmean(distances) if distances else None,'mean_work':statistics.fmean(works) if works else 0,'max_work':max(works,default=0)}
    @staticmethod
    def environment(): return {'python':sys.version.split()[0],'platform':platform.platform()}
    def _write_manifest(self,scenario,records,verdict):
        def result_dict(r):
            return {'case_id':r.case_id,'seed':r.seed,'found':r.result.found,'distance':r.result.distance,'work':r.result.work_expanded_nodes,'trace':r.ultrasound_artifact}
        (self.output_dir/'manifest.json').write_text(json.dumps({'scenario':scenario.scenario_id,'records':[result_dict(r) for r in records],'verdict':asdict(verdict)},indent=2,ensure_ascii=False,default=repr),encoding='utf-8')
