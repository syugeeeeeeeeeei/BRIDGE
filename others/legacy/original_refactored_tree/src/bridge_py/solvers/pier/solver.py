from __future__ import annotations

import math
from dataclasses import replace
from typing import Any

from ...core.bounds import make_bounds
from ...graph import Graph, path_distance
from ...types import Node, PathResult
from ..mrpc_dg6 import mrpc_dg6


def pier(G: Graph, source: Node, target: Node, **kwargs: Any) -> PathResult:
    """PIER v0.1 compatibility implementation built from the DG6 baseline.

    DG6 remains available as a frozen legacy entry point.  This adapter changes
    ownership and telemetry semantics without silently claiming capabilities the
    underlying implementation does not yet possess.
    """
    target_ratio = float(kwargs.get("target_ratio", kwargs.get("max_suboptimality", 1.10)))
    result = mrpc_dg6(G, source, target, **kwargs)
    verified_distance = path_distance(G, result.path) if result.found else math.inf
    distance_corrected = result.found and math.isfinite(verified_distance) and not math.isclose(verified_distance, result.distance, rel_tol=1e-12, abs_tol=1e-12)
    if distance_corrected:
        result = replace(result, distance=verified_distance)
    bounds = make_bounds(G, source, target, result.distance, target_ratio)
    tel = dict(result.telemetry)
    first_work = tel.get("first_path_work")
    tel.update({
        "algorithm_family": "PIER",
        "legacy_name": "MRPC-DG6",
        "pier_version": "0.1.0",
        "phase_model": "first_path_then_refinement",
        "first_path_found": bool(tel.get("first_path_found", result.found)),
        "first_path_work": first_work,
        "first_path_time_ms": tel.get("first_path_time_ms"),
        "refinement_work": int(tel.get("quality_budget_used", 0) or 0),
        "lower_bound": bounds.lower_bound,
        "upper_bound": bounds.upper_bound,
        "certified_ratio": bounds.certified_ratio,
        "quality_certified": bounds.quality_certified,
        "bound_method": bounds.method,
        "quality_prediction_is_certificate": False,
        "distance_recomputed_from_path": True,
        "distance_corrected": distance_corrected,
    })
    return replace(
        result,
        solver_name="pier",
        lower_bound=bounds.lower_bound,
        certified_ratio=bounds.certified_ratio,
        quality_certified=bounds.quality_certified,
        first_path_work=int(first_work) if first_work is not None else None,
        budget_exhausted=bool(tel.get("budget_exhausted", False)),
        telemetry=tel,
    )
