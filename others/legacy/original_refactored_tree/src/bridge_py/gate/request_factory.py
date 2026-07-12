from __future__ import annotations

from typing import Any, Mapping

from ..core import RouteRequest


def build_route_request(
    source: Any,
    target: Any,
    *,
    mode: str = "balanced",
    constraints: Mapping[str, Any] | None = None,
    **overrides: Any,
) -> RouteRequest:
    options = dict(constraints or {})
    normalized_mode = "balanced" if mode == "auto" else mode

    def take(name: str, default: Any = None) -> Any:
        value = overrides.get(name)
        return options.pop(name, default) if value is None else value

    max_suboptimality = take("max_suboptimality")
    if max_suboptimality is None:
        max_suboptimality = options.pop("max_distance_ratio", None)

    return RouteRequest(
        source=source,
        target=target,
        mode=normalized_mode,
        max_suboptimality=max_suboptimality,
        deadline_ms=take("deadline_ms"),
        work_budget=take("work_budget"),
        memory_budget_kib=take("memory_budget_kib"),
        workers=int(take("workers", 1)),
        seed=int(take("seed", 0)),
        constraints=options,
    )
