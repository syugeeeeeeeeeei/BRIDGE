from __future__ import annotations

from typing import Any, Mapping

from ..bearing import NullObserver
from ..core import RouteRequest
from ..core.graph import normalize_graph
from ..truss import Truss
from .request_factory import build_route_request


class Gate:
    """BRIDGE public boundary.

    GATE performs input normalization and validation, then delegates only to
    the TRUSS public API.
    """

    def __init__(self, truss: Truss | None = None) -> None:
        self._truss = truss or Truss(observer=NullObserver())

    def route_request(self, graph: object, request: RouteRequest):
        request.validate()
        return self._truss.route(normalize_graph(graph), request)

    def route(
        self,
        graph: object,
        source: object,
        target: object,
        mode: str = "balanced",
        constraints: Mapping[str, Any] | None = None,
        **overrides: Any,
    ):
        request = build_route_request(
            source,
            target,
            mode=mode,
            constraints=constraints,
            **overrides,
        )
        return self.route_request(graph, request)
