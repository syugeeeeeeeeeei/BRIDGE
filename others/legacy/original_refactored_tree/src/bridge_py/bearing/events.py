from __future__ import annotations
from dataclasses import dataclass
from typing import Any, Mapping


@dataclass(frozen=True)
class SearchEvent:
    kind: str
    attributes: Mapping[str, Any]
