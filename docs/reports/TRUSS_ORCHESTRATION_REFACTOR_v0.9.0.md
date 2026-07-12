# TRUSS orchestration refactor v0.9.0

## Implemented

- Separated TRUSS into Orchestrator, Budget, Supervisor, and Arbiter responsibilities.
- Added CORE coordination contracts for progress, emergency, and directives.
- Removed production wiring of ANCHOR's direct BOLTS connector.
- TRUSS now receives ANCHOR results, classifies emergencies, issues directives, starts BOLTS, and arbitrates complete candidates.
- Added TRUSS-level ANCHOR/BOLTS Work accounting and aggregate investigated-node/edge telemetry.
- Added BEARING events for progress, emergency, directives, and component lifecycle.
- Documented second-stage Scheduler and third-stage Session Registry extraction triggers.

## Evaluation summary

The 5,000-node, 12 connected topology benchmark achieved 100% discovery. BRIDGE mean
path ratio was 1.0058, median 1.0000, p95 1.0356, and worst 1.0635. Mean Total Work was
22,966, compared with 47,415 for A* and 45,581 for bidirectional Dijkstra. BRIDGE inspected
34.5% of nodes and 37.3% of directed edge slots on average.

Hard cases remain expensive: alternating walls inspect about 83% of nodes; deceptive
weights about 97%; the snake path necessarily covers all nodes. Portal quality is corrected
by TRUSS arbitration, but this requires broad BOLTS recovery and loses much of the Work
advantage. The next algorithmic priority is checkpoint/state transfer so BOLTS can avoid
restarting from the source while preserving complete-path arbitration.
