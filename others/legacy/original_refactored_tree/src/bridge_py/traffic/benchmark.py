from __future__ import annotations

import argparse
import csv
import json
import math
import statistics
from concurrent.futures import ProcessPoolExecutor, as_completed
from dataclasses import asdict
from pathlib import Path
from typing import Iterable, List

from ..graphs.generators import diagonal_extreme_pair, grid_graph, random_geometric_graph
from ..solvers.dijkstra import bidirectional_dijkstra
from ..solvers.mprc import mprc
from ..types import BenchmarkRow


def _ratio(a: float, b: float) -> float:
    if not math.isfinite(a) or not math.isfinite(b) or b == 0:
        return math.inf
    return a / b


def _run_case(args: tuple[int, int, str, int, str, int]) -> list[BenchmarkRow]:
    n, trial, graph_type, seed, exp, mprc_workers = args
    sseed = seed + n * 1000 + trial
    if graph_type == "grid":
        side = max(2, int(round(math.sqrt(n))))
        G = grid_graph(side, side, diagonal=False)
    else:
        G = random_geometric_graph(n, seed=sseed, k_neighbors=12)
    source, target = diagonal_extreme_pair(G)
    exact = bidirectional_dijkstra(G, source, target)
    solvers = [exact, mprc(G, source, target, workers=mprc_workers)]
    rows: list[BenchmarkRow] = []
    for res in solvers:
        ratio = _ratio(res.distance, exact.distance)
        exact_match = bool(res.found and exact.found and abs(res.distance - exact.distance) <= 1e-9 * max(1.0, exact.distance))
        tel = res.telemetry
        rows.append(BenchmarkRow(
            run_id=f"{exp}-{n}-{trial}-{res.solver_name}",
            experiment_id=exp,
            trial=trial,
            seed=sseed,
            graph_type=graph_type,
            nodes=len(G.adj),
            edges=G.edge_count(),
            source=source,
            target=target,
            query_class="far",
            solver_name=res.solver_name,
            found=res.found,
            distance=res.distance,
            exact_distance=exact.distance,
            distance_ratio=ratio,
            exact_match=exact_match,
            work_relaxations=res.work_relaxations,
            work_expanded_nodes=res.work_expanded_nodes,
            total_work=res.total_work,
            parallel_steps=res.parallel_steps,
            time_ms=res.time_ms,
            peak_memory_kib=res.peak_memory_kib,
            k_corridors=int(tel.get("k_corridors", 0)),
            candidate_count=int(tel.get("candidate_count", 0)),
            best_corridor_id=str(tel.get("best_corridor_id", "")),
            rescue_triggered=bool(tel.get("rescue_triggered", False)),
            repair_triggered=bool(tel.get("repair_triggered", False)),
            error_code=tel.get("error_code"),
        ))
    return rows


def run_benchmark(
    node_sizes: Iterable[int],
    *,
    trials: int = 5,
    graph_type: str = "rgg",
    seed: int = 20260709,
    mprc_workers: int = 1,
    benchmark_workers: int = 1,
) -> List[BenchmarkRow]:
    rows: List[BenchmarkRow] = []
    exp = f"mprc_vs_bidir_{graph_type}_py"
    cases = [(n, trial, graph_type, seed, exp, mprc_workers) for n in node_sizes for trial in range(1, trials + 1)]
    if benchmark_workers > 1 and len(cases) > 1:
        max_workers = max(1, min(int(benchmark_workers), len(cases)))
        with ProcessPoolExecutor(max_workers=max_workers) as ex:
            futures = [ex.submit(_run_case, c) for c in cases]
            for fut in as_completed(futures):
                rows.extend(fut.result())
    else:
        for c in cases:
            rows.extend(_run_case(c))
    rows.sort(key=lambda r: (r.nodes, r.trial, r.solver_name))
    return rows


def write_rows(rows: List[BenchmarkRow], out: Path, fmt: str) -> None:
    out.parent.mkdir(parents=True, exist_ok=True)
    data = [asdict(r) for r in rows]
    if not data:
        raise ValueError("No benchmark rows were produced.")
    if fmt == "json":
        out.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
    else:
        with out.open("w", newline="", encoding="utf-8") as f:
            w = csv.DictWriter(f, fieldnames=list(data[0].keys()))
            w.writeheader(); w.writerows(data)


def summarize(rows: List[BenchmarkRow]) -> List[dict]:
    groups = {}
    for r in rows:
        groups.setdefault((r.graph_type, r.query_class, r.nodes, r.solver_name), []).append(r)
    out = []
    for (graph_type, query_class, nodes, solver), rs in sorted(groups.items()):
        ratios = [r.distance_ratio for r in rs if math.isfinite(r.distance_ratio)]
        times = [r.time_ms for r in rs]
        out.append({
            "experiment_id": rs[0].experiment_id,
            "graph_type": graph_type,
            "query_class": query_class,
            "nodes": nodes,
            "solver_name": solver,
            "trials": len(rs),
            "found_rate": sum(r.found for r in rs) / len(rs),
            "exact_rate": sum(r.exact_match for r in rs) / len(rs),
            "mean_distance_ratio": statistics.fmean(ratios) if ratios else math.inf,
            "worst_distance_ratio": max(ratios) if ratios else math.inf,
            "mean_work": statistics.fmean(r.total_work for r in rs),
            "mean_parallel_steps": statistics.fmean(r.parallel_steps for r in rs),
            "mean_time_ms": statistics.fmean(times),
            "p95_time_ms": sorted(times)[max(0, math.ceil(len(times) * 0.95) - 1)],
            "mean_peak_memory_kib": statistics.fmean(r.peak_memory_kib for r in rs),
            "rescue_rate": sum(r.rescue_triggered for r in rs) / len(rs),
            "repair_success_rate": sum((r.repair_triggered and r.found) for r in rs) / max(1, sum(r.repair_triggered for r in rs)),
        })
    return out


def main(argv=None) -> int:
    p = argparse.ArgumentParser(prog="bridge-py benchmark")
    p.add_argument("nodes", nargs="*", type=int, default=[100, 250, 500, 750, 1000, 1500, 2000])
    p.add_argument("-o", "--output", default="benchmark_raw.csv")
    p.add_argument("-s", "--summary", default="benchmark_summary.csv")
    p.add_argument("-f", "--format", choices=["csv", "json"], default="csv")
    p.add_argument("--trials", type=int, default=5)
    p.add_argument("--graph-type", choices=["rgg", "grid"], default="rgg")
    p.add_argument("--seed", type=int, default=20260709)
    p.add_argument("--mprc-workers", type=int, default=1, help="MPRC corridor search process count per query.")
    p.add_argument("--benchmark-workers", type=int, default=1, help="Process count for independent benchmark cases.")
    args = p.parse_args(argv)
    rows = run_benchmark(
        args.nodes,
        trials=args.trials,
        graph_type=args.graph_type,
        seed=args.seed,
        mprc_workers=args.mprc_workers,
        benchmark_workers=args.benchmark_workers,
    )
    write_rows(rows, Path(args.output), args.format)
    summary = summarize(rows)
    with Path(args.summary).open("w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=list(summary[0].keys()))
        w.writeheader(); w.writerows(summary)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
