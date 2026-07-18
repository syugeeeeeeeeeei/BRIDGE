#!/usr/bin/env python3
"""Create and verify BRIDGE performance regression baselines."""
from __future__ import annotations

import argparse
import json
import platform
import sys
from pathlib import Path
from typing import Any

SCHEMA = "bridge.performance-regression-baseline.v1"
TIER_POLICIES = {
    "small": {
        "performance": {
            "solver_time_p50_max_increase_ratio": 0.35,
            "solver_time_p95_max_increase_ratio": 0.50,
            "end_to_end_p50_max_increase_ratio": 0.35,
            "end_to_end_p95_max_increase_ratio": 0.50,
            "alloc_bytes_p50_max_increase_ratio": 0.15,
            "alloc_bytes_p95_max_increase_ratio": 0.20,
        },
        "minimum_sample_count": 60,
    },
    "medium": {
        "performance": {
            "solver_time_p50_max_increase_ratio": 0.20,
            "solver_time_p95_max_increase_ratio": 0.30,
            "end_to_end_p50_max_increase_ratio": 0.20,
            "end_to_end_p95_max_increase_ratio": 0.30,
            "alloc_bytes_p50_max_increase_ratio": 0.10,
            "alloc_bytes_p95_max_increase_ratio": 0.15,
        },
        "minimum_sample_count": 10,
    },
    "large": {
        "performance": {
            "solver_time_p50_max_increase_ratio": 0.15,
            "solver_time_p95_max_increase_ratio": 0.20,
            "end_to_end_p50_max_increase_ratio": 0.15,
            "end_to_end_p95_max_increase_ratio": 0.20,
            "alloc_bytes_p50_max_increase_ratio": 0.10,
            "alloc_bytes_p95_max_increase_ratio": 0.15,
        },
        "minimum_sample_count": 3,
    },
}

SEMANTIC_POLICY = {
    "path_found_rate_min": 1.0,
    "exact_algorithms_optimality_rate_min": 1.0,
    "bridge_false_optimality_max": 0.0,
    "work_p50_max_increase_ratio": 0.0,
}


def policy_for(tier: str) -> dict[str, Any]:
    if tier not in TIER_POLICIES:
        raise ValueError(f"unsupported tier: {tier}")
    tier_policy = TIER_POLICIES[tier]
    return {
        "tier": tier,
        "semantic": dict(SEMANTIC_POLICY),
        "performance": dict(tier_policy["performance"]),
        "minimum_sample_count": tier_policy["minimum_sample_count"],
        "environment_match_required": True,
    }



def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def artifact_dir(path: Path) -> Path:
    if path.is_file() and path.name == "result.json":
        return path.parent
    if path.is_dir() and (path / "result.json").exists():
        return path
    raise ValueError(f"artifact directory or result.json required: {path}")


def metric(summary: dict[str, Any], group: str, key: str) -> float:
    value = summary.get(group, {}).get(key)
    if value is None:
        raise ValueError(f"missing metric {group}.{key}")
    return float(value)


def key_for(summary: dict[str, Any]) -> str:
    return "/".join((summary["scenario_id"], summary["algorithm"], summary["query_id"]))


def extract(artifact: Path) -> tuple[dict[str, Any], dict[str, Any]]:
    result = load_json(artifact / "result.json")
    env = load_json(artifact / "environment.json")
    entries: dict[str, Any] = {}
    for s in result["scenario_summaries"]:
        alloc = s.get("metric_statistics", {}).get("alloc_bytes", {})
        entries[key_for(s)] = {
            "scenario_id": s["scenario_id"],
            "algorithm": s["algorithm"],
            "query_id": s["query_id"],
            "runs": int(s["runs"]),
            "path_found_rate": float(s["path_found_rate"]),
            "optimality_proven_rate": float(s["optimality_proven_rate"]),
            "work_p50": metric(s, "work_statistics", "p50"),
            "work_p95": metric(s, "work_statistics", "p95"),
            "solver_time_p50_ms": metric(s, "solver_time_statistics", "p50"),
            "solver_time_p95_ms": metric(s, "solver_time_statistics", "p95"),
            "end_to_end_p50_ms": metric(s, "end_to_end_time_statistics", "p50"),
            "end_to_end_p95_ms": metric(s, "end_to_end_time_statistics", "p95"),
            "alloc_bytes_p50": float(alloc.get("p50", 0)),
            "alloc_bytes_p95": float(alloc.get("p95", 0)),
        }
    environment = {k: env.get(k) for k in ("go_version", "goos", "goarch", "cpus")}
    return environment, entries


def create_baseline(args: argparse.Namespace) -> int:
    artifact = artifact_dir(Path(args.artifact))
    environment, entries = extract(artifact)
    baseline = {
        "schema_version": SCHEMA,
        "suite_id": load_json(artifact / "result.json")["suite_id"],
        "source_artifact": artifact.name,
        "environment": environment,
        "policy": policy_for(args.tier),
        "entries": entries,
    }
    out = Path(args.output)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(baseline, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"baseline written: {out}")
    return 0


def allowed(current: float, base: float, ratio: float) -> bool:
    return current <= base * (1.0 + ratio)


def check(args: argparse.Namespace) -> int:
    baseline = load_json(Path(args.baseline))
    if baseline.get("schema_version") != SCHEMA:
        raise ValueError("unsupported baseline schema")
    artifact = artifact_dir(Path(args.artifact))
    environment, current = extract(artifact)
    policy = baseline["policy"]
    failures: list[str] = []
    warnings: list[str] = []

    if policy.get("environment_match_required", True) and environment != baseline["environment"]:
        failures.append(f"environment mismatch: baseline={baseline['environment']} current={environment}")

    expected = baseline["entries"]
    if set(expected) != set(current):
        failures.append(f"entry set mismatch: missing={sorted(set(expected)-set(current))} extra={sorted(set(current)-set(expected))}")

    exact_algorithms = {"dijkstra", "astar"}
    sem = policy["semantic"]
    perf = policy["performance"]
    for key in sorted(set(expected) & set(current)):
        b, c = expected[key], current[key]
        if c["runs"] < int(policy["minimum_sample_count"]):
            failures.append(f"{key}: sample count {c['runs']} below {policy['minimum_sample_count']}")
        if c["path_found_rate"] < sem["path_found_rate_min"]:
            failures.append(f"{key}: path_found_rate {c['path_found_rate']}")
        if c["algorithm"] in exact_algorithms and c["optimality_proven_rate"] < sem["exact_algorithms_optimality_rate_min"]:
            failures.append(f"{key}: exact optimality rate {c['optimality_proven_rate']}")
        if c["algorithm"] == "bridge" and c["optimality_proven_rate"] > sem["bridge_false_optimality_max"]:
            failures.append(f"{key}: unexpected BRIDGE optimality claim {c['optimality_proven_rate']}")

        comparisons = [
            ("work_p50", sem["work_p50_max_increase_ratio"]),
            ("solver_time_p50_ms", perf["solver_time_p50_max_increase_ratio"]),
            ("solver_time_p95_ms", perf["solver_time_p95_max_increase_ratio"]),
            ("end_to_end_p50_ms", perf["end_to_end_p50_max_increase_ratio"]),
            ("end_to_end_p95_ms", perf["end_to_end_p95_max_increase_ratio"]),
            ("alloc_bytes_p50", perf["alloc_bytes_p50_max_increase_ratio"]),
            ("alloc_bytes_p95", perf["alloc_bytes_p95_max_increase_ratio"]),
        ]
        for name, ratio in comparisons:
            if not allowed(c[name], b[name], ratio):
                failures.append(f"{key}: {name} baseline={b[name]:.6g} current={c[name]:.6g} limit=+{ratio:.0%}")

    report = {
        "status": "pass" if not failures else "fail",
        "baseline": str(args.baseline),
        "artifact": str(artifact),
        "environment": environment,
        "failures": failures,
        "warnings": warnings,
    }
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 0 if not failures else 1


def main() -> int:
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest="command", required=True)
    create = sub.add_parser("create-baseline")
    create.add_argument("artifact")
    create.add_argument("output")
    create.add_argument("--tier", choices=sorted(TIER_POLICIES), required=True)
    create.set_defaults(func=create_baseline)
    verify = sub.add_parser("check")
    verify.add_argument("baseline")
    verify.add_argument("artifact")
    verify.set_defaults(func=check)
    args = parser.parse_args()
    try:
        return args.func(args)
    except (OSError, ValueError, KeyError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2

if __name__ == "__main__":
    raise SystemExit(main())
