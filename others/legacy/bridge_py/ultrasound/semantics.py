from __future__ import annotations

from dataclasses import dataclass
from math import isfinite
from typing import Any, Iterable, Mapping

TRACE_SCHEMA_VERSION = "1.1.0"


@dataclass(frozen=True)
class FieldSemantics:
    """Normative meaning of one serialized ULTRASOUND field.

    The registry is executable documentation. A field must not be emitted unless
    its meaning, unit, nullability, producer, and invariants are defined here or
    in the event-specific payload contract.
    """

    name: str
    meaning: str
    unit: str
    nullable: bool
    producer: str
    invariants: tuple[str, ...] = ()
    forbidden_interpretations: tuple[str, ...] = ()


COMMON_FIELDS: dict[str, FieldSemantics] = {
    "schema_version": FieldSemantics(
        "schema_version",
        "Version of the serialized trace contract used to interpret every field in this record.",
        "semantic version",
        False,
        "ULTRASOUND serializer",
        ("MUST equal the writer's supported trace schema version.",),
    ),
    "sequence": FieldSemantics(
        "sequence",
        "One-based append order inside a single trace artifact.",
        "ordinal",
        False,
        "ULTRASOUND observer",
        ("Starts at 1.", "Strictly increases by 1 with no gaps inside one artifact."),
        ("Not a solver work count.", "Not a logical or parallel step."),
    ),
    "kind": FieldSemantics(
        "kind",
        "Event vocabulary member identifying the payload contract.",
        "enum string",
        False,
        "BEARING callback selected by the producer",
    ),
    "relative_ns": FieldSemantics(
        "relative_ns",
        "Monotonic elapsed time from creation of this observer to append of this event.",
        "nanoseconds",
        False,
        "ULTRASOUND observer clock",
        ("Non-negative.", "Non-decreasing within one artifact."),
        ("Not CPU time.", "Not algorithmic work.", "Not comparable across processes or artifacts."),
    ),
    "task_id": FieldSemantics(
        "task_id",
        "Stable identifier of the TRUSS-assigned solver task that produced the event.",
        "opaque identifier",
        False,
        "TRUSS task creation",
        ("Non-empty.", "Does not change during a task."),
    ),
    "logical_step": FieldSemantics(
        "logical_step",
        "One-based expansion step local to one task and lane. It advances when that task commits one node expansion.",
        "ordinal",
        False,
        "solver instrumentation",
        ("Zero is reserved for task-level events outside an expansion step.", "For node expansion events it is >= 1."),
        ("Not wall-clock time.", "Not globally unique.", "Not automatically a parallel superstep."),
    ),
    "lane": FieldSemantics(
        "lane",
        "Logical execution stream used to distinguish independently progressing search directions or solver roles.",
        "enum or implementation-defined string",
        False,
        "solver instrumentation",
        ("Stable for events belonging to the same logical stream.",),
        ("Does not prove physical thread-level parallelism."),
    ),
    "phase": FieldSemantics(
        "phase",
        "Named algorithmic phase in which the event was generated.",
        "controlled vocabulary string",
        False,
        "component emitting the event",
    ),
}


EVENT_SEMANTICS: dict[str, dict[str, FieldSemantics]] = {
    "step_started": {
        "logical_step": COMMON_FIELDS["logical_step"],
        "lane": COMMON_FIELDS["lane"],
    },
    "node_expanded": {
        "node": FieldSemantics("node", "Graph node removed or selected from the active frontier for expansion.", "NodeId", False, "solver"),
        "distance": FieldSemantics("distance", "Best known source-to-node cost at the instant of expansion.", "graph cost", True, "solver", ("Finite and non-negative when present.",)),
        "frontier_size": FieldSemantics("frontier_size", "Number of queued frontier entries immediately after removing the expanded entry and before relaxations from it.", "entries", True, "solver", ("Non-negative when present.",), ("May exceed number of unique nodes when stale queue entries are retained.")),
        "work_used": FieldSemantics("work_used", "Cumulative expansion work consumed by this task including the current expansion.", "node expansions", False, "solver", ("Positive.", "Non-decreasing within a task."), ("Not edge relaxations.", "Not portfolio total work unless the task is the only task.")),
    },
    "edge_relaxed": {
        "source": FieldSemantics("source", "Expanded endpoint from which the directed relaxation attempt originates.", "NodeId", False, "solver"),
        "target": FieldSemantics("target", "Neighbor endpoint whose tentative distance was evaluated.", "NodeId", False, "solver"),
        "weight": FieldSemantics("weight", "Graph edge cost used in this relaxation attempt.", "graph cost", False, "graph view", ("Finite and non-negative for supported shortest-path graphs.",)),
        "old_distance": FieldSemantics("old_distance", "Tentative target cost immediately before this relaxation; null means no finite tentative cost existed.", "graph cost", True, "solver"),
        "new_distance": FieldSemantics("new_distance", "Candidate target cost computed as source tentative cost plus edge weight.", "graph cost", False, "solver", ("Finite and non-negative.",)),
        "improved": FieldSemantics("improved", "True exactly when new_distance is strictly lower than old_distance, treating null old_distance as infinity.", "boolean", False, "solver", ("Must agree with old_distance and new_distance."), ("Does not mean the target was expanded or is on the final path.")),
    },
    "neighbor_scored": {
        "source": FieldSemantics("source", "Current greedy-path node whose neighbor is being ranked.", "NodeId", False, "ANCHOR greedy strategy"),
        "target": FieldSemantics("target", "Neighbor considered as the next greedy-path node.", "NodeId", False, "ANCHOR greedy strategy"),
        "edge_weight": FieldSemantics("edge_weight", "Graph edge cost from source to target.", "graph cost", False, "graph view", ("Finite and non-negative.",)),
        "heuristic_to_target": FieldSemantics("heuristic_to_target", "Geometric distance from the neighbor to the route target used only for ranking.", "coordinate distance", False, "ANCHOR greedy strategy", ("Finite and non-negative.",), ("Not a graph path cost or certified lower bound.",)),
        "progress": FieldSemantics("progress", "Decrease in geometric target distance from source to neighbor; negative values indicate movement away from target.", "coordinate distance", False, "ANCHOR greedy strategy", ("Finite.",)),
        "score": FieldSemantics("score", "Strategy-specific ranking value h + weight_bias*edge_weight - 0.25*max(0, progress); lower is preferred.", "mixed ranking score", False, "ANCHOR greedy strategy", ("Finite.",), ("Not tentative distance, route cost, bound, or quality certificate.")),
    },
    "candidate_found": {
        "found": FieldSemantics("found", "Whether the task produced a valid source-to-target candidate at this notification.", "boolean", False, "solver"),
        "distance": FieldSemantics("distance", "Total graph cost of the emitted candidate, or null when found is false.", "graph cost", True, "solver", ("Present, finite, and non-negative iff found is true.")),
        "path_length": FieldSemantics("path_length", "Number of nodes in the candidate path; zero when no candidate exists.", "nodes", False, "solver", ("Non-negative.", "Positive iff found is true.")),
        "solver": FieldSemantics("solver", "Public solver identity responsible for the candidate.", "controlled string", False, "solver adapter"),
        "strategy": FieldSemantics("strategy", "ANCHOR hypothesis/strategy used to generate the candidate, when applicable.", "controlled string", True, "TRUSS plan / ANCHOR"),
    },
    "bound_updated": {
        "lower_bound": FieldSemantics("lower_bound", "Certified lower bound on the optimal route cost known after this update.", "graph cost", True, "TRUSS/BOLTS", ("Finite and non-negative when present.",), ("Must not contain an unproven heuristic estimate.")),
        "upper_bound": FieldSemantics("upper_bound", "Cost of the best valid complete route known after this update.", "graph cost", True, "TRUSS", ("Finite and non-negative when present.",), ("Not a tentative frontier key.")),
        "certified_ratio": FieldSemantics("certified_ratio", "Upper-bound divided by positive certified lower-bound; 1 denotes proven optimality.", "dimensionless ratio", True, "TRUSS", ("At least 1 when present.", "Only present when both required certified bounds exist.")),
    },
    "budget_updated": {
        "max_work": FieldSemantics("max_work", "Maximum node-expansion work assigned to this task slice; null means no explicit work cap.", "node expansions", True, "TRUSS", ("Non-negative when present.",)),
        "work_used": FieldSemantics("work_used", "Portfolio cumulative node-expansion work already charged when the update is emitted.", "node expansions", False, "TRUSS", ("Non-negative.",)),
        "portfolio_remaining": FieldSemantics("portfolio_remaining", "Unallocated/unused portfolio expansion budget after accounting for charged work.", "node expansions", True, "TRUSS", ("Non-negative when present.",)),
    },
    "phase_started": {
        "phase": COMMON_FIELDS["phase"],
        "attributes": FieldSemantics("attributes", "Phase-specific descriptive metadata. Keys require a documented producer contract before use in acceptance criteria.", "mapping", False, "component"),
    },
    "phase_finished": {
        "phase": COMMON_FIELDS["phase"],
        "attributes": FieldSemantics("attributes", "Phase completion metadata. It describes outcome but is not authoritative over RouteResult.", "mapping", False, "component"),
    },
}


@dataclass(frozen=True)
class TraceValidationReport:
    valid: bool
    errors: tuple[str, ...]
    warnings: tuple[str, ...]
    event_count: int

    def require_valid(self) -> None:
        if not self.valid:
            raise ValueError("Invalid ULTRASOUND trace:\n- " + "\n- ".join(self.errors))


def _event_payload(record: Mapping[str, Any]) -> Mapping[str, Any]:
    value = record.get("event")
    return value if isinstance(value, Mapping) else record


def validate_trace(events: Iterable[Mapping[str, Any]], *, strict: bool = True) -> TraceValidationReport:
    records = list(events)
    errors: list[str] = []
    warnings: list[str] = []
    last_time = -1
    task_steps: dict[tuple[str, str], int] = {}
    task_work: dict[str, int] = {}
    started_steps: set[tuple[str, str, int]] = set()
    expanded_steps: set[tuple[str, str, int]] = set()
    open_phases: dict[str, int] = {}

    for index, record in enumerate(records, start=1):
        prefix = f"record {index}"
        if record.get("schema_version") != TRACE_SCHEMA_VERSION:
            errors.append(f"{prefix}: schema_version must be {TRACE_SCHEMA_VERSION!r}")
        if record.get("sequence") != index:
            errors.append(f"{prefix}: sequence must equal append position {index}")
        relative_ns = record.get("relative_ns")
        if not isinstance(relative_ns, int) or relative_ns < 0:
            errors.append(f"{prefix}: relative_ns must be a non-negative integer")
        elif relative_ns < last_time:
            errors.append(f"{prefix}: relative_ns decreased")
        else:
            last_time = relative_ns

        kind = record.get("kind")
        if kind not in EVENT_SEMANTICS:
            errors.append(f"{prefix}: undocumented event kind {kind!r}")
            continue
        payload = _event_payload(record)
        for field_name, definition in EVENT_SEMANTICS[kind].items():
            if field_name not in payload and field_name not in record:
                errors.append(f"{prefix}: missing documented field {field_name!r} for {kind}")
            value = payload.get(field_name, record.get(field_name))
            if value is None and not definition.nullable:
                errors.append(f"{prefix}: non-nullable field {field_name!r} is null")

        if kind in {"phase_started", "phase_finished"}:
            phase = record.get("phase")
            if not isinstance(phase, str) or not phase:
                errors.append(f"{prefix}: phase must be non-empty")
            elif kind == "phase_started":
                open_phases[phase] = open_phases.get(phase, 0) + 1
            elif open_phases.get(phase, 0) <= 0:
                errors.append(f"{prefix}: phase_finished has no matching phase_started for {phase!r}")
            else:
                open_phases[phase] -= 1

        if kind == "step_started":
            step, lane = record.get("logical_step"), record.get("lane")
            if not isinstance(step, int) or step < 1:
                errors.append(f"{prefix}: step_started logical_step must be >= 1")
            if not isinstance(lane, str) or not lane:
                errors.append(f"{prefix}: step_started lane must be non-empty")
            if isinstance(step, int) and isinstance(lane, str):
                started_steps.add(("", lane, step))

        if kind in {"node_expanded", "edge_relaxed", "neighbor_scored", "candidate_found", "bound_updated", "budget_updated"}:
            task_id = payload.get("task_id")
            lane = payload.get("lane")
            step = payload.get("logical_step")
            if not isinstance(task_id, str) or not task_id:
                errors.append(f"{prefix}: task_id must be non-empty")
            if not isinstance(lane, str) or not lane:
                errors.append(f"{prefix}: lane must be non-empty")
            if not isinstance(step, int) or step < 0:
                errors.append(f"{prefix}: logical_step must be a non-negative integer")

        if kind == "node_expanded":
            task_id, lane, step = payload.get("task_id"), payload.get("lane"), payload.get("logical_step")
            work = payload.get("work_used")
            frontier = payload.get("frontier_size")
            distance = payload.get("distance")
            key = (str(task_id), str(lane))
            expected = task_steps.get(key, 0) + 1
            if step != expected:
                errors.append(f"{prefix}: node expansion step {step!r} must be contiguous; expected {expected}")
            else:
                task_steps[key] = step
            if isinstance(step, int):
                expanded_steps.add((str(task_id), str(lane), step))
            if not isinstance(work, int) or work < 1:
                errors.append(f"{prefix}: work_used must be a positive integer")
            elif work < task_work.get(str(task_id), 0):
                errors.append(f"{prefix}: work_used decreased within task")
            else:
                task_work[str(task_id)] = work
            if frontier is not None and (not isinstance(frontier, int) or frontier < 0):
                errors.append(f"{prefix}: frontier_size must be non-negative or null")
            if distance is not None and (not isinstance(distance, (int, float)) or not isfinite(distance) or distance < 0):
                errors.append(f"{prefix}: distance must be finite and non-negative or null")

        if kind == "edge_relaxed":
            old, new, improved = payload.get("old_distance"), payload.get("new_distance"), payload.get("improved")
            weight = payload.get("weight")
            if not isinstance(weight, (int, float)) or not isfinite(weight) or weight < 0:
                errors.append(f"{prefix}: weight must be finite and non-negative")
            if not isinstance(new, (int, float)) or not isfinite(new) or new < 0:
                errors.append(f"{prefix}: new_distance must be finite and non-negative")
            if old is not None and (not isinstance(old, (int, float)) or not isfinite(old) or old < 0):
                errors.append(f"{prefix}: old_distance must be finite and non-negative or null")
            expected_improved = old is None or (isinstance(new, (int, float)) and isinstance(old, (int, float)) and new < old)
            if improved is not expected_improved:
                errors.append(f"{prefix}: improved disagrees with old_distance/new_distance")
            step_key = (str(payload.get("task_id")), str(payload.get("lane")), payload.get("logical_step"))
            if step_key not in expanded_steps:
                errors.append(f"{prefix}: edge relaxation has no preceding node expansion in the same task/lane/step")


        if kind == "neighbor_scored":
            edge_weight = payload.get("edge_weight")
            heuristic = payload.get("heuristic_to_target")
            progress = payload.get("progress")
            score = payload.get("score")
            for name, value in (("edge_weight", edge_weight), ("heuristic_to_target", heuristic), ("progress", progress), ("score", score)):
                if not isinstance(value, (int, float)) or not isfinite(value):
                    errors.append(f"{prefix}: {name} must be finite")
            if isinstance(edge_weight, (int, float)) and edge_weight < 0:
                errors.append(f"{prefix}: edge_weight must be non-negative")
            if isinstance(heuristic, (int, float)) and heuristic < 0:
                errors.append(f"{prefix}: heuristic_to_target must be non-negative")
            step_key = (str(payload.get("task_id")), str(payload.get("lane")), payload.get("logical_step"))
            if step_key not in expanded_steps:
                errors.append(f"{prefix}: neighbor scoring has no preceding node expansion in the same task/lane/step")

        if kind == "candidate_found":
            found, distance, path_length = payload.get("found"), payload.get("distance"), payload.get("path_length")
            if not isinstance(found, bool):
                errors.append(f"{prefix}: found must be boolean")
            if found:
                if not isinstance(distance, (int, float)) or not isfinite(distance) or distance < 0:
                    errors.append(f"{prefix}: found candidate requires finite non-negative distance")
                if not isinstance(path_length, int) or path_length < 1:
                    errors.append(f"{prefix}: found candidate requires positive path_length")
            elif distance is not None or path_length != 0:
                errors.append(f"{prefix}: absent candidate requires distance=null and path_length=0")

        if kind == "bound_updated":
            lower, upper, ratio = payload.get("lower_bound"), payload.get("upper_bound"), payload.get("certified_ratio")
            for name, value in (("lower_bound", lower), ("upper_bound", upper)):
                if value is not None and (not isinstance(value, (int, float)) or not isfinite(value) or value < 0):
                    errors.append(f"{prefix}: {name} must be finite and non-negative or null")
            if lower is not None and upper is not None and lower > upper:
                errors.append(f"{prefix}: lower_bound cannot exceed upper_bound")
            if ratio is not None:
                if lower is None or upper is None or lower <= 0:
                    errors.append(f"{prefix}: certified_ratio requires positive lower_bound and upper_bound")
                elif abs(ratio - upper / lower) > 1e-9:
                    errors.append(f"{prefix}: certified_ratio must equal upper_bound/lower_bound")
                if not isinstance(ratio, (int, float)) or ratio < 1:
                    errors.append(f"{prefix}: certified_ratio must be >= 1")

        if kind == "budget_updated":
            max_work, used, remaining = payload.get("max_work"), payload.get("work_used"), payload.get("portfolio_remaining")
            for name, value in (("max_work", max_work), ("work_used", used), ("portfolio_remaining", remaining)):
                if value is not None and (not isinstance(value, int) or value < 0):
                    errors.append(f"{prefix}: {name} must be a non-negative integer or null")

    for phase, count in open_phases.items():
        if count:
            errors.append(f"trace end: phase {phase!r} has {count} unmatched phase_started event(s)")

    if strict and not records:
        errors.append("trace is empty")
    return TraceValidationReport(not errors, tuple(errors), tuple(warnings), len(records))
