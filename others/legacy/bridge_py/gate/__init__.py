from __future__ import annotations

from typing import Any, Mapping

from .api import Gate
from .request_factory import build_route_request


def route(
    graph: object,
    source: object,
    target: object,
    mode: str = "balanced",
    constraints: Mapping[str, Any] | None = None,
    **overrides: Any,
):
    if mode == "auto":
        mode = "balanced"
    return Gate().route(
        graph,
        source,
        target,
        mode=mode,
        constraints=constraints,
        **overrides,
    )


def path(graph: object, source: object, target: object, mode: str = "balanced"):
    return route(graph, source, target, mode=mode).path


__all__ = ["Gate", "build_route_request", "path", "route"]
