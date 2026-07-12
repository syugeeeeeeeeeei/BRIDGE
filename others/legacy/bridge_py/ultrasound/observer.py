from __future__ import annotations
import json, time
from dataclasses import asdict, dataclass, field, is_dataclass
from pathlib import Path
from typing import Any

from .semantics import TRACE_SCHEMA_VERSION, TraceValidationReport, validate_trace


def _plain(value: Any) -> Any:
    if is_dataclass(value): return {k:_plain(v) for k,v in asdict(value).items()}
    if isinstance(value, dict): return {str(k):_plain(v) for k,v in value.items()}
    if isinstance(value, (list,tuple)): return [_plain(v) for v in value]
    return value


@dataclass
class InMemoryObserver:
    """Development-only BEARING adapter with deterministic event order.

    Semantic rules are normative in ``ultrasound/TRACE_SEMANTICS.md`` and
    executable in ``ultrasound.semantics``. ``relative_ns`` is diagnostic
    elapsed time only; algorithmic comparisons must use logical_step/work_used.
    """
    events: list[dict[str, Any]] = field(default_factory=list)
    started_ns: int = field(default_factory=time.perf_counter_ns)

    def _add(self, kind, **payload):
        self.events.append({
            'schema_version': TRACE_SCHEMA_VERSION,
            'sequence': len(self.events)+1,
            'kind': kind,
            'relative_ns': time.perf_counter_ns()-self.started_ns,
            **_plain(payload),
        })

    def phase_started(self, phase, attributes): self._add('phase_started', phase=phase, attributes=dict(attributes))
    def phase_finished(self, phase, attributes): self._add('phase_finished', phase=phase, attributes=dict(attributes))
    def step_started(self, logical_step, lane=None): self._add('step_started', logical_step=logical_step, lane=lane)
    def node_expanded(self, event): self._add('node_expanded', event=event)
    def edge_relaxed(self, event): self._add('edge_relaxed', event=event)
    def neighbor_scored(self, event): self._add('neighbor_scored', event=event)
    def candidate_found(self, event): self._add('candidate_found', event=event)
    def bound_updated(self, event): self._add('bound_updated', event=event)
    def budget_updated(self, event): self._add('budget_updated', event=event)

    def metrics(self) -> dict[str, Any]:
        """Return descriptive counts only; this method does not certify correctness."""
        counts={}
        for e in self.events: counts[e['kind']]=counts.get(e['kind'],0)+1
        steps=[e['logical_step'] for e in self.events if e['kind']=='step_started']
        report=self.validate(strict=False)
        return {
            'schema_version': TRACE_SCHEMA_VERSION,
            'event_count':len(self.events),
            'counts':counts,
            'logical_steps':max(steps,default=0),
            'lanes':sorted({e.get('lane') for e in self.events if e.get('lane') is not None}),
            'semantically_valid': report.valid,
            'semantic_error_count': len(report.errors),
        }

    def validate(self, *, strict: bool = True) -> TraceValidationReport:
        return validate_trace(self.events, strict=strict)

    def write_jsonl(self,path:str|Path, *, validate: bool = True)->str:
        if validate:
            self.validate(strict=True).require_valid()
        p=Path(path); p.parent.mkdir(parents=True,exist_ok=True)
        with p.open('w',encoding='utf-8') as f:
            for e in self.events: f.write(json.dumps(e,ensure_ascii=False,default=repr)+'\n')
        return str(p)

    @classmethod
    def read_jsonl(cls,path:str|Path, *, validate: bool = True)->'InMemoryObserver':
        obj=cls(); obj.events=[json.loads(x) for x in Path(path).read_text(encoding='utf-8').splitlines() if x.strip()]
        if validate:
            obj.validate(strict=True).require_valid()
        return obj
