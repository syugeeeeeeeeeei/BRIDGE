from __future__ import annotations

import argparse
import csv
import json
import math
import os
import statistics
import time
import tracemalloc
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, Iterable

from bridge_py.graph import path_distance
from bridge_py.solvers.astar import astar
from bridge_py.solvers.dijkstra import bidirectional_dijkstra, dijkstra
from bridge_py.solvers.mrpc_cg import mrpc_dg5_switchback
from bridge_py.solvers.mrpc_dg6 import mrpc_dg6
from broad_eval import clustered, grid_graph, random_geometric, scale_free


GRID_TOPOLOGIES = (
    "open",
    "weighted_noise",
    "wall",
    "double_wall",
    "u_shape",
    "culdesac",
    "spiral",
    "random_obstacles",
    "disconnected",
)
OTHER_TOPOLOGIES = (
    "random_geometric",
    "clustered",
    "scale_free_pos",
    "scale_free_no_pos",
)
DEFAULT_SOLVERS = ("dijkstra", "bidir", "astar", "dg5_clean", "dg6")


@dataclass(frozen=True)
class Preset:
    grid_sides: tuple[int, ...]
    graph_sizes: tuple[int, ...]
    seeds: int


PRESETS = {
    # CIや動作確認用。
    "quick": Preset(grid_sides=(20, 40), graph_sizes=(400, 900), seeds=2),
    # 旧dg6_eval.pyと同程度だが、seedを増やす。
    "standard": Preset(grid_sides=(20, 40, 70), graph_sizes=(400, 900, 1600), seeds=3),
    # 既定。20～50,000ノード級を広く評価する。
    "high": Preset(
        grid_sides=(20, 40, 70, 100, 141, 224),
        graph_sizes=(400, 900, 2000, 5000, 10000, 20000, 50000),
        seeds=5,
    ),
    # 長時間耐久評価。全solverでは非常に重い。
    "stress": Preset(
        grid_sides=(20, 40, 70, 100, 141, 224, 317),
        graph_sizes=(400, 900, 2000, 5000, 10000, 20000, 50000, 100000),
        seeds=10,
    ),
}


def percentile(values: Iterable[float], q: float) -> float:
    xs = sorted(values)
    if not xs:
        return math.inf
    if len(xs) == 1:
        return xs[0]
    pos = (len(xs) - 1) * q
    lo = math.floor(pos)
    hi = math.ceil(pos)
    if lo == hi:
        return xs[lo]
    return xs[lo] + (xs[hi] - xs[lo]) * (pos - lo)


def parse_int_list(text: str | None) -> tuple[int, ...] | None:
    if not text:
        return None
    values = tuple(int(x.strip()) for x in text.split(",") if x.strip())
    if not values or any(v <= 0 for v in values):
        raise argparse.ArgumentTypeError("正の整数をカンマ区切りで指定してください。")
    return values


def run_solver(name, graph, source, target, measure_memory: bool):
    if name == "dg6":
        fn = lambda: mrpc_dg6(graph, source, target, fallback_exact=False)
    elif name == "dg5_clean":
        fn = lambda: mrpc_dg5_switchback(
            graph,
            source,
            target,
            workers=1,
            trace_level=0,
            measure_memory=False,
            enable_topology_gate=True,
            dg5_mode="balanced",
            quality_guard_max_ratio=None,
            allow_oracle_quality_guard=False,
        )
    elif name == "astar":
        fn = lambda: astar(graph, source, target)
    elif name == "dijkstra":
        fn = lambda: dijkstra(graph, source, target)
    elif name == "bidir":
        fn = lambda: bidirectional_dijkstra(graph, source, target)
    else:
        raise ValueError(f"unknown solver: {name}")

    if measure_memory:
        tracemalloc.start()
    start = time.perf_counter()
    result = fn()
    elapsed_ms = (time.perf_counter() - start) * 1000
    if measure_memory:
        _, peak = tracemalloc.get_traced_memory()
        tracemalloc.stop()
    else:
        peak = 0

    telemetry = result.telemetry or {}
    actual_distance = path_distance(graph, result.path) if result.found else math.inf
    valid = (not result.found) or (
        bool(result.path)
        and result.path[0] == source
        and result.path[-1] == target
        and math.isfinite(actual_distance)
        and abs(actual_distance - result.distance) <= 1e-7 * max(1.0, actual_distance)
    )
    work = int(
        telemetry.get(
            "total_work_including_preprocessing",
            int(telemetry.get("precheck_work", 0))
            + int(telemetry.get("query_work_units", result.total_work)),
        )
    )
    return result, elapsed_ms, peak / 1024, valid, actual_distance, work


def make_cases(preset: Preset, grid_sides: tuple[int, ...], graph_sizes: tuple[int, ...], seeds: int):
    cases: list[tuple[str, int, int, Callable]] = []
    for topology in GRID_TOPOLOGIES:
        for side in grid_sides:
            for seed in range(seeds):
                cases.append(
                    (
                        topology,
                        side * side,
                        seed,
                        lambda topology=topology, side=side, seed=seed: grid_graph(
                            side,
                            "normal" if topology == "open" else topology,
                            seed,
                            0.8 if topology == "weighted_noise" else 0.0,
                        ),
                    )
                )
    for n in graph_sizes:
        for seed in range(seeds):
            cases.extend(
                [
                    ("random_geometric", n, seed, lambda n=n, seed=seed: random_geometric(n, seed)),
                    ("clustered", n, seed, lambda n=n, seed=seed: clustered(n, seed)),
                    ("scale_free_pos", n, seed, lambda n=n, seed=seed: scale_free(n, seed, True)),
                    ("scale_free_no_pos", n, seed, lambda n=n, seed=seed: scale_free(n, seed, False)),
                ]
            )
    return cases


def aggregate(rs: list[dict]) -> dict:
    connected = [r for r in rs if r["reference_found"]]
    finite = [r["distance_ratio"] for r in connected if math.isfinite(r["distance_ratio"])]
    work_values = [r["total_work"] for r in rs]
    time_values = [r["total_time_ms"] for r in rs]
    return {
        "cases": len(rs),
        "connected_cases": len(connected),
        "disconnected_cases": len(rs) - len(connected),
        "found_rate_all": sum(bool(r["found"]) for r in rs) / len(rs),
        "found_rate_connected": sum(bool(r["found"]) for r in connected) / len(connected) if connected else math.nan,
        "valid_rate": sum(bool(r["valid"]) for r in rs) / len(rs),
        "unreachable_agreement_rate": (
            sum(not r["found"] for r in rs if not r["reference_found"])
            / max(1, sum(not r["reference_found"] for r in rs))
        ),
        "exact_rate_connected": sum(bool(r["exact_match_connected"]) for r in connected) / len(connected) if connected else math.nan,
        "within_1pct_rate": sum(bool(r["within_1pct"]) for r in connected) / len(connected) if connected else math.nan,
        "within_3pct_rate": sum(bool(r["within_3pct"]) for r in connected) / len(connected) if connected else math.nan,
        "within_5pct_rate": sum(bool(r["within_5pct"]) for r in connected) / len(connected) if connected else math.nan,
        "within_10pct_rate": sum(bool(r["within_10pct"]) for r in connected) / len(connected) if connected else math.nan,
        "mean_distance_ratio": statistics.mean(finite) if finite else math.inf,
        "p50_distance_ratio": percentile(finite, 0.50),
        "p95_distance_ratio": percentile(finite, 0.95),
        "p99_distance_ratio": percentile(finite, 0.99),
        "worst_distance_ratio": max(finite) if finite else math.inf,
        "mean_total_work": statistics.mean(work_values),
        "p50_total_work": percentile(work_values, 0.50),
        "p95_total_work": percentile(work_values, 0.95),
        "p99_total_work": percentile(work_values, 0.99),
        "mean_total_time_ms": statistics.mean(time_values),
        "p50_total_time_ms": percentile(time_values, 0.50),
        "p95_total_time_ms": percentile(time_values, 0.95),
        "p99_total_time_ms": percentile(time_values, 0.99),
        "mean_steps": statistics.mean(r["steps"] for r in rs),
        "fallback_rate": sum(bool(r["fallback_used"]) for r in rs) / len(rs),
        "oracle_rate": sum(bool(r["oracle_used"]) for r in rs) / len(rs),
    }


def write_csv(path: Path, rows: list[dict]):
    if not rows:
        return
    fields = sorted({key for row in rows for key in row})
    with path.open("w", newline="", encoding="utf-8-sig") as f:
        writer = csv.DictWriter(f, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="BRIDGE MRPC-DG6 high-load benchmark")
    parser.add_argument("--preset", choices=PRESETS, default="high", help="負荷プリセット。既定はhigh。")
    parser.add_argument("--grid-sides", help="グリッド辺長の上書き。例: 20,40,70,100,224")
    parser.add_argument("--graph-sizes", help="非グリッドのノード数上書き。例: 400,2000,10000,50000")
    parser.add_argument("--seeds", type=int, help="各条件のseed数を上書き")
    parser.add_argument("--solvers", default=",".join(DEFAULT_SOLVERS), help="実行solverをカンマ区切りで指定")
    parser.add_argument("--output", default="evaluation_results/dg6_high", help="出力ディレクトリ")
    parser.add_argument("--measure-memory", action="store_true", help="tracemallocで各runのpeak memoryを測定")
    parser.add_argument("--max-cases", type=int, default=None, help="デバッグ用にcase数を制限")
    parser.add_argument("--progress-every", type=int, default=10, help="進捗表示間隔")
    return parser


def main():
    args = build_parser().parse_args()
    preset = PRESETS[args.preset]
    grid_sides = parse_int_list(args.grid_sides) or preset.grid_sides
    graph_sizes = parse_int_list(args.graph_sizes) or preset.graph_sizes
    seeds = args.seeds if args.seeds is not None else preset.seeds
    if seeds <= 0:
        raise SystemExit("--seedsは1以上で指定してください。")

    solvers = tuple(x.strip() for x in args.solvers.split(",") if x.strip())
    unknown = set(solvers) - set(DEFAULT_SOLVERS)
    if unknown:
        raise SystemExit(f"unknown solver(s): {sorted(unknown)}")
    if "dijkstra" not in solvers:
        raise SystemExit("distance ratioの基準としてdijkstraを--solversに含めてください。")

    out = Path(args.output)
    out.mkdir(parents=True, exist_ok=True)
    cases = make_cases(preset, grid_sides, graph_sizes, seeds)
    if args.max_cases is not None:
        cases = cases[: max(0, args.max_cases)]

    config = {
        "preset": args.preset,
        "grid_sides": list(grid_sides),
        "graph_sizes": list(graph_sizes),
        "seeds": seeds,
        "solvers": list(solvers),
        "measure_memory": args.measure_memory,
        "cases": len(cases),
        "runs": len(cases) * len(solvers),
    }
    (out / "config.json").write_text(json.dumps(config, indent=2, ensure_ascii=False), encoding="utf-8")
    print(json.dumps(config, ensure_ascii=False), flush=True)

    rows: list[dict] = []
    total_start = time.perf_counter()
    for idx, (topology, requested_nodes, seed, maker) in enumerate(cases):
        graph, source, target = maker()
        case_rows: list[dict] = []
        results = {}
        for solver in solvers:
            result, elapsed_ms, peak_kib, valid, actual_distance, work = run_solver(
                solver, graph, source, target, args.measure_memory
            )
            results[solver] = result
            telemetry = result.telemetry or {}
            case_rows.append(
                {
                    "case_id": idx,
                    "topology": topology,
                    "requested_nodes": requested_nodes,
                    "nodes": len(graph.adj),
                    "edges": graph.edge_count(),
                    "seed": seed,
                    "source": source,
                    "target": target,
                    "solver": solver,
                    "found": result.found,
                    "valid": valid,
                    "distance": result.distance,
                    "path_distance": actual_distance,
                    "path_nodes": len(result.path),
                    "total_work": work,
                    "work_per_node": work / max(1, len(graph.adj)),
                    "total_time_ms": elapsed_ms,
                    "peak_kib_uniform": peak_kib,
                    "steps": result.parallel_steps,
                    "strategy": telemetry.get("strategy", ""),
                    "fallback_used": telemetry.get("fallback_used", False),
                    "oracle_used": telemetry.get("oracle_used", False)
                    or telemetry.get("oracle_quality_guard_used", False),
                    "candidate_count": telemetry.get("candidate_count", 0),
                    "repair_triggered": telemetry.get("repair_triggered", False),
                    "repair_success": telemetry.get("repair_success", False),
                    "target_work": telemetry.get("target_work", ""),
                    "work_goal_ratio": telemetry.get("work_goal_ratio", ""),
                    "first_path_work": telemetry.get("first_path_work", ""),
                    "quality_budget_used": telemetry.get("quality_budget_used", ""),
                    "budget_exhausted": telemetry.get("budget_exhausted", ""),
                    "emergency_path_used": telemetry.get("emergency_path_used", False),
                    "emergency_work": telemetry.get("emergency_work", 0),
                    "error_code": telemetry.get("error_code"),
                    "topology_gate_action": telemetry.get("topology_gate_action", ""),
                }
            )

        exact = results["dijkstra"]
        for row in case_rows:
            row["reference_found"] = exact.found
            if exact.found and row["found"] and exact.distance > 0:
                ratio = row["distance"] / exact.distance
            elif exact.found and row["found"] and exact.distance == 0:
                ratio = 1.0 if row["distance"] == 0 else math.inf
            else:
                ratio = math.inf
            row["distance_ratio"] = ratio
            row["exact_match_connected"] = bool(exact.found and math.isfinite(ratio) and abs(ratio - 1.0) <= 1e-9)
            row["within_1pct"] = bool(exact.found and math.isfinite(ratio) and ratio <= 1.01 + 1e-9)
            row["within_3pct"] = bool(exact.found and math.isfinite(ratio) and ratio <= 1.03 + 1e-9)
            row["within_5pct"] = bool(exact.found and math.isfinite(ratio) and ratio <= 1.05 + 1e-9)
            row["within_10pct"] = bool(exact.found and math.isfinite(ratio) and ratio <= 1.10 + 1e-9)
            row["unreachable_agreement"] = bool((not exact.found) and (not row["found"]))
        rows.extend(case_rows)

        if args.progress_every > 0 and ((idx + 1) % args.progress_every == 0 or idx + 1 == len(cases)):
            elapsed = time.perf_counter() - total_start
            rate = (idx + 1) / elapsed if elapsed > 0 else 0.0
            remaining = (len(cases) - idx - 1) / rate if rate > 0 else math.inf
            print(
                f"cases {idx + 1}/{len(cases)} | runs {(idx + 1) * len(solvers)}/{len(cases) * len(solvers)} "
                f"| elapsed {elapsed:.1f}s | estimated_remaining {remaining:.1f}s",
                flush=True,
            )

    write_csv(out / "raw.csv", rows)

    topology_groups = defaultdict(list)
    for row in rows:
        topology_groups[(row["topology"], row["solver"])].append(row)
    summary_rows = []
    for (topology, solver), group in sorted(topology_groups.items()):
        summary_rows.append({"topology": topology, "solver": solver, **aggregate(group)})
    write_csv(out / "summary.csv", summary_rows)

    global_rows = []
    for solver in solvers:
        group = [row for row in rows if row["solver"] == solver]
        global_rows.append({"solver": solver, **aggregate(group)})
    write_csv(out / "global_summary.csv", global_rows)

    size_groups = defaultdict(list)
    for row in rows:
        size_groups[(row["requested_nodes"], row["solver"])].append(row)
    size_rows = []
    for (requested_nodes, solver), group in sorted(size_groups.items()):
        size_rows.append({"requested_nodes": requested_nodes, "solver": solver, **aggregate(group)})
    write_csv(out / "size_summary.csv", size_rows)

    strategy_groups = defaultdict(list)
    for row in rows:
        if row["solver"] == "dg6":
            strategy_groups[(row["topology"], row["strategy"])].append(row)
    strategy_rows = []
    for (topology, strategy), group in sorted(strategy_groups.items()):
        strategy_rows.append({"topology": topology, "strategy": strategy, **aggregate(group)})
    write_csv(out / "strategy_summary.csv", strategy_rows)

    elapsed = time.perf_counter() - total_start
    result = {
        **config,
        "elapsed_seconds": elapsed,
        "output": str(out),
    }
    (out / "run_summary.json").write_text(json.dumps(result, indent=2, ensure_ascii=False), encoding="utf-8")
    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
