#!/usr/bin/env python3
"""Compare pre/post-refactor BRIDGE benchmark runs by deterministic semantics.

Wall-clock, allocation, timestamp, execution-id, and host-environment fields are
intentionally excluded. Any algorithmic path, proof, Work, budget, graph, query,
or stable-digest difference fails the check.
"""
from __future__ import annotations
import argparse, json
from pathlib import Path

FIELDS = (
    ("run_metadata", "run_id"),
    ("run_metadata", "stable_digest"),
    ("scenario_definition",),
    ("graph_profile",),
    ("query_profile",),
    ("algorithm_configuration",),
    ("execution_result", "returned_path"),
    ("execution_result", "path_found"),
    ("execution_result", "search_completed"),
    ("execution_result", "reachability_proven"),
    ("execution_result", "optimality_proven"),
    ("execution_result", "path_cost"),
    ("execution_result", "termination_reason"),
    ("execution_result", "improvement_count"),
    ("execution_result", "budget_ledger"),
    ("execution_result", "quality_claims"),
    ("measurement", "work"),
)

def load(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]

def value(record: dict, path: tuple[str, ...]):
    current = record
    for key in path:
        current = current[key]
    return current

def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("before", type=Path)
    p.add_argument("after", type=Path)
    args = p.parse_args()
    before, after = load(args.before), load(args.after)
    if len(before) != len(after):
        raise SystemExit(f"run count differs: {len(before)} != {len(after)}")
    for index, (left, right) in enumerate(zip(before, after), start=1):
        for field in FIELDS:
            if value(left, field) != value(right, field):
                dotted = ".".join(field)
                raise SystemExit(f"run {index}: {dotted} differs")
    print(f"equivalent deterministic benchmark runs: {len(before)}")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
