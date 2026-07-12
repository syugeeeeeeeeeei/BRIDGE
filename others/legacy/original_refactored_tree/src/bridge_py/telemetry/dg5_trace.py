from __future__ import annotations

import math
import time
from collections import Counter
from dataclasses import dataclass, field
from typing import Any

from ..graph import Graph
from ..types import Node


@dataclass
class DG5TraceCollector:
    """Bounded structured event collector for DG5 diagnosis.

    level=0 disables event storage, level=1 stores phase/decision events,
    level=2 additionally stores sampled frontier/expansion events.
    """

    level: int = 0
    sample_every: int = 8
    max_events: int = 20_000
    events: list[dict[str, Any]] = field(default_factory=list)
    dropped_events: int = 0
    _sequence: int = 0
    _start_ns: int = field(default_factory=time.perf_counter_ns)

    @property
    def enabled(self) -> bool:
        return self.level > 0

    def emit(self, event: str, *, detail_level: int = 1, force: bool = False, **fields: Any) -> None:
        if not self.enabled or detail_level > self.level:
            return
        self._sequence += 1
        if not force and detail_level >= 2 and self.sample_every > 1 and self._sequence % self.sample_every:
            return
        if len(self.events) >= self.max_events:
            self.dropped_events += 1
            return
        row = {
            "seq": self._sequence,
            "elapsed_ms": (time.perf_counter_ns() - self._start_ns) / 1_000_000.0,
            "event": event,
        }
        row.update({k: _json_safe(v) for k, v in fields.items()})
        self.events.append(row)

    def summary(self) -> dict[str, Any]:
        counts = Counter(str(e.get("event")) for e in self.events)
        return {
            "trace_level": self.level,
            "trace_sample_every": self.sample_every,
            "trace_max_events": self.max_events,
            "trace_event_count": len(self.events),
            "trace_dropped_events": self.dropped_events,
            "trace_event_counts": dict(sorted(counts.items())),
        }


def _json_safe(value: Any) -> Any:
    if value is None or isinstance(value, (bool, int, float, str)):
        if isinstance(value, float) and not math.isfinite(value):
            return str(value)
        return value
    if isinstance(value, tuple):
        return [_json_safe(v) for v in value]
    if isinstance(value, list):
        return [_json_safe(v) for v in value]
    if isinstance(value, set):
        return [_json_safe(v) for v in sorted(value, key=repr)]
    if isinstance(value, dict):
        return {str(k): _json_safe(v) for k, v in value.items()}
    return repr(value)


def graph_topology_profile(G: Graph, source: Node, target: Node) -> dict[str, Any]:
    """Cheap, deterministic topology features available before the query."""
    degrees = [len(nbrs) for nbrs in G.adj.values()]
    edge_entries = sum(degrees)
    directed = bool(getattr(G, "directed", False))
    edge_count = edge_entries if directed else edge_entries // 2
    n = len(degrees)
    mean_degree = edge_entries / max(1, n)
    degree_var = sum((d - mean_degree) ** 2 for d in degrees) / max(1, n)
    out: dict[str, Any] = {
        "nodes": n,
        "edges": edge_count,
        "directed": directed,
        "mean_degree": mean_degree,
        "degree_std": math.sqrt(degree_var),
        "min_degree": min(degrees, default=0),
        "max_degree": max(degrees, default=0),
        "source_degree": len(G.adj.get(source, [])),
        "target_degree": len(G.adj.get(target, [])),
        "has_positions": bool(G.pos),
    }
    if G.pos and source in G.pos and target in G.pos:
        sx, sy = G.pos[source]
        tx, ty = G.pos[target]
        out["source_target_euclidean"] = math.hypot(tx - sx, ty - sy)
    return out
