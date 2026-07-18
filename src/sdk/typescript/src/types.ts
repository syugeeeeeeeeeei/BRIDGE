export type ObservationMode = "minimum" | "debug" | "trace";
export interface GraphNode { id: number; x?: number; y?: number }
export interface GraphEdge { from: number; to: number; weight: number }
export interface RouteRequest {
  schema_version: "bridge.route.request.v1";
  request_id?: string;
  graph: { type: "inline"; directed?: boolean; nodes: GraphNode[]; edges: GraphEdge[] };
  route: { source: number; target: number; route_mode?: string; max_suboptimality?: number; logical_worker_count?: number; seed?: number };
  budget?: { total_work?: number; timeout_ms?: number };
  observation_config?: { level?: ObservationMode; sample_rate?: number };
}
export interface RouteResult {
  schema_version: "bridge.route.result.v1";
  request_id?: string;
  status: string;
  path_found: boolean;
  search_completed: boolean;
  reachability_proven: boolean;
  path_cost?: number;
  path: number[];
  optimality_proven: boolean;
  solver_name?: string;
  work: Record<string, number>;
  end_to_end_time_ms: number;
  error_code?: string;
  observation_data?: { level: ObservationMode; event_count: number; dropped_events?: number; truncated: boolean };
}
export interface RouteResponse { result: RouteResult; warnings: string[] }
