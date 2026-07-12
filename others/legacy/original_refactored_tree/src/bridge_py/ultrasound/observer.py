from __future__ import annotations
from dataclasses import dataclass, field
from typing import Any


@dataclass
class InMemoryObserver:
    """Development-only BEARING adapter. Never imported by production layers."""
    events: list[dict[str, Any]] = field(default_factory=list)
    def _add(self, kind, **payload): self.events.append({'kind':kind, **payload})
    def phase_started(self, phase, attributes): self._add('phase_started', phase=phase, attributes=dict(attributes))
    def phase_finished(self, phase, attributes): self._add('phase_finished', phase=phase, attributes=dict(attributes))
    def step_started(self, logical_step, lane=None): self._add('step_started', logical_step=logical_step, lane=lane)
    def node_expanded(self, event): self._add('node_expanded', event=event)
    def edge_relaxed(self, event): self._add('edge_relaxed', event=event)
    def candidate_found(self, event): self._add('candidate_found', event=event)
    def bound_updated(self, event): self._add('bound_updated', event=event)
    def budget_updated(self, event): self._add('budget_updated', event=event)
