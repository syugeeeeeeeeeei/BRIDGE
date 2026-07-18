#!/usr/bin/env python3
"""Generate BRIDGE trace simulations directly from a benchmark ZIP artifact."""
from __future__ import annotations

import argparse
import json
import tempfile
import zipfile
import math
import io

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from PIL import Image
from pathlib import Path
from typing import Any
from matplotlib.collections import LineCollection


def load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def load_runs(path: Path) -> list[dict[str, Any]]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def resolve_member(root: Path, recorded: str, fallback: Path) -> Path:
    if recorded:
        name = Path(recorded).name
        candidates = list(root.rglob(name))
        for candidate in candidates:
            if candidate.parent.name == fallback.parent.name:
                return candidate
    return fallback


def discover(zip_path: Path, work: Path) -> list[dict[str, Any]]:
    with zipfile.ZipFile(zip_path) as archive:
        archive.extractall(work)
    manifests = list(work.rglob("manifest.json"))
    execution_manifests = [p for p in manifests if load_json(p).get("schema_version") == "bridge.benchmark.execution.v1"]
    if len(execution_manifests) != 1:
        raise ValueError(f"expected one benchmark execution manifest, found {len(execution_manifests)}")
    root = execution_manifests[0].parent
    runs_path = root / "runs.jsonl"
    if not runs_path.is_file():
        raise ValueError("runs.jsonl is missing from benchmark archive")
    selected = []
    for run in load_runs(runs_path):
        refs = run.get("references", {})
        ordinal = int(run.get("run_metadata", {}).get("run_ordinal", 0))
        run_dir = root / "traces" / f"run-{ordinal:06d}"
        trace = resolve_member(root, refs.get("trace_path", ""), run_dir / "trace.jsonl")
        graph = resolve_member(root, refs.get("graph_snapshot_path", ""), run_dir / "graph.json")
        if trace.is_file() and graph.is_file():
            selected.append({"run": run, "trace": trace, "graph": graph})
    if not selected:
        raise ValueError("archive contains no simulatable trace runs; use observation.mode=trace")
    return selected


def invoke_renderer(item: dict[str, Any], output: Path, duration_ms: int, max_frames: int) -> None:
    snapshot = load_json(item["graph"])
    graph = snapshot["graph"]
    nodes = [int(n["id"]) for n in graph.get("nodes", [])]
    edges = [(int(e["from"]), int(e["to"])) for e in graph.get("edges", [])]
    source, target = int(snapshot["source"]), int(snapshot["target"])
    events = [json.loads(line) for line in item["trace"].read_text(encoding="utf-8").splitlines() if line.strip()]
    expanded: set[int] = set()
    frontier: set[int] = set()
    path: list[int] = []
    states: list[tuple[set[int], set[int], list[int], str]] = []
    for event in events:
        kind = str(event.get("kind", ""))
        attrs = event.get("attributes") or {}
        if kind == "frontier_enqueued" and "node" in attrs:
            frontier.add(int(attrs["node"]))
        elif kind == "frontier_selected" and "node" in attrs:
            frontier.discard(int(attrs["node"]))
        elif kind == "node_expanded" and "node" in attrs:
            node = int(attrs["node"]); expanded.add(node); frontier.discard(node)
        elif kind in {"candidate_submitted", "search_finished"} and isinstance(attrs.get("path"), list):
            path = [int(v) for v in attrs["path"]]
        if kind in {"node_expanded", "candidate_submitted", "search_finished"}:
            states.append((set(expanded), set(frontier), list(path), kind))
    if not states:
        states = [(set(), set(), [], "initial")]
    max_frames = max(1, max_frames)
    if len(states) > max_frames:
        indexes = sorted({round(i * (len(states) - 1) / (max_frames - 1)) for i in range(max_frames)})
        states = [states[i] for i in indexes]
    count = max(1, len(nodes))
    positions = {node: (math.cos(2 * math.pi * i / count), math.sin(2 * math.pi * i / count)) for i, node in enumerate(nodes)}
    path_edges = lambda p: {tuple(sorted((a, b))) for a, b in zip(p, p[1:])}
    images = []
    for expanded_state, frontier_state, path_state, kind in states:
        fig, ax = plt.subplots(figsize=(10, 8), dpi=100)
        pe = path_edges(path_state)
        base_segments = [[positions[u], positions[v]] for u, v in edges if tuple(sorted((u, v))) not in pe]
        path_segments = [[positions[u], positions[v]] for u, v in edges if tuple(sorted((u, v))) in pe]
        if base_segments:
            ax.add_collection(LineCollection(base_segments, linewidths=0.4, alpha=0.25))
        if path_segments:
            ax.add_collection(LineCollection(path_segments, linewidths=2.2, alpha=0.9))
        for node in nodes:
            x, y = positions[node]
            marker = "*" if node in {source, target} else "o"
            size = 100 if node in {source, target} else 24
            alpha = 1.0 if node in expanded_state or node in frontier_state or node in path_state else 0.25
            ax.scatter([x], [y], s=size, marker=marker, alpha=alpha)
        ax.set_title(f"{item['run']['run_metadata']['run_id']}\n{kind} · expanded={len(expanded_state)} · frontier={len(frontier_state)}")
        ax.set_aspect("equal"); ax.axis("off")
        buffer = io.BytesIO(); fig.savefig(buffer, format="png", facecolor="white"); plt.close(fig); buffer.seek(0)
        images.append(Image.open(buffer).convert("RGB"))
    images[0].save(output, save_all=True, append_images=images[1:], duration=duration_ms, loop=0, optimize=False)


def main() -> int:
    parser = argparse.ArgumentParser(description="Simulate a BRIDGE benchmark ZIP artifact")
    parser.add_argument("archive", type=Path)
    parser.add_argument("--output-dir", type=Path)
    parser.add_argument("--duration-ms", type=int, default=220)
    parser.add_argument("--frames", type=int, default=24, help="maximum frames per run")
    parser.add_argument("--max-runs", type=int, default=0, help="process at most this many runs; 0 means all")
    args = parser.parse_args()
    if not args.archive.is_file() or args.archive.suffix.lower() != ".zip":
        parser.error("archive must be an existing .zip file")
    output_dir = args.output_dir or args.archive.with_suffix("").with_name(args.archive.stem + "-simulation")
    output_dir.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="bridge-simulator-") as tmp:
        items = discover(args.archive, Path(tmp))
        if args.max_runs > 0:
            items = items[:args.max_runs]
        manifest = {"schema_version": "bridge.simulation.bundle.v1", "source_archive": args.archive.name, "runs": []}
        for item in items:
            md = item["run"]["run_metadata"]
            name = f"run-{int(md['run_ordinal']):06d}.gif"
            target = output_dir / name
            invoke_renderer(item, target, args.duration_ms, args.frames)
            manifest["runs"].append({"run_ordinal": md["run_ordinal"], "run_id": md["run_id"], "gif": name})
        (output_dir / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(output_dir)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
