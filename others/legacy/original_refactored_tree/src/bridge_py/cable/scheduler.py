from __future__ import annotations
from dataclasses import replace
from ..bearing import NullObserver
from ..truss import Truss


def cable_route(G, request):
    """Deprecated CABLE compatibility facade backed by TRUSS."""
    result = Truss(observer=NullObserver()).route(G, request)
    telemetry = dict(result.telemetry)
    telemetry['cable_version'] = 'compat-truss-0.1.0'
    telemetry['deprecated_component'] = 'CABLE'
    return replace(result, telemetry=telemetry)
