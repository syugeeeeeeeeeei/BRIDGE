export type ObservationMode = "off" | "summary" | "trace" | "profile";
export interface GraphNode { id: number; x?: number; y?: number }
export interface GraphEdge { from: number; to: number; weight: number }
export interface RouteRequest {
  schema_version: "bridge.route.v1";
  request_id?: string;
  graph: { type: "inline"; directed?: boolean; nodes: GraphNode[]; edges: GraphEdge[] };
  route: { source: number; target: number; mode?: string; max_suboptimality?: number; workers?: number; seed?: number };
  budget?: { total_work?: number; timeout_ms?: number };
  observation?: { mode?: ObservationMode };
}
export interface RouteResult {
  schema_version: "bridge.route.result.v1";
  request_id?: string;
  status: string;
  found: boolean;
  distance?: number;
  path: number[];
  exact: boolean;
  solver_name?: string;
  work: Record<string, number>;
  time_ms: number;
  error_code?: string;
  observation?: { mode: ObservationMode; event_count: number; dropped_events?: number; truncated: boolean };
}
export interface RouteResponse { result: RouteResult; warnings: string[] }
