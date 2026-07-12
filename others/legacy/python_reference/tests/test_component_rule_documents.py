from __future__ import annotations

import ast
from pathlib import Path

COMPONENTS = ("core", "gate", "truss", "anchor", "bolts", "bearing", "ultrasound", "traffic")
REQUIRED_PHRASES = (
    "# ",
    "## 1. 定義",
    "所有する責務",
    "所有してはならない責務",
    "依存規則",
    "必須テスト",
    "文書の効力",
)


def package_root() -> Path:
    return Path(__file__).resolve().parents[1] / "bridge_py"


def imported_components(component: str) -> set[str]:
    found: set[str] = set()
    for path in (package_root() / component).glob("*.py"):
        tree = ast.parse(path.read_text(encoding="utf-8"), filename=str(path))
        for node in ast.walk(tree):
            if isinstance(node, ast.ImportFrom) and node.level >= 2 and node.module:
                found.add(node.module.split(".")[0])
            elif isinstance(node, ast.Import):
                for alias in node.names:
                    parts = alias.name.split(".")
                    if len(parts) >= 2 and parts[0] == "bridge_py":
                        found.add(parts[1])
    return found


def test_every_component_has_complete_rule_document() -> None:
    for component in COMPONENTS:
        path = package_root() / component / "COMPONENT_RULE.md"
        assert path.is_file(), component
        text = path.read_text(encoding="utf-8")
        for phrase in REQUIRED_PHRASES:
            assert phrase in text, f"{component}: missing {phrase}"


def test_core_and_bearing_remain_inward_only() -> None:
    assert imported_components("core") == set()
    assert imported_components("bearing") == set()


def test_anchor_has_no_forbidden_component_dependency() -> None:
    assert imported_components("anchor").isdisjoint({"truss", "bolts", "gate", "ultrasound", "traffic"})


def test_bolts_have_no_forbidden_component_dependency() -> None:
    assert imported_components("bolts").isdisjoint({"truss", "anchor", "gate", "ultrasound", "traffic"})


def test_gate_only_crosses_public_boundary_to_core_truss_bearing() -> None:
    assert imported_components("gate") <= {"core", "truss", "bearing"}
