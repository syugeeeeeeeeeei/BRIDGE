import { spawn } from "node:child_process";
import { resolveBinary } from "./discovery.js";
import { BridgeAcceptanceError, BridgeCancelledError, BridgeError, BridgeIOError, BridgeInternalError, BridgeProtocolError, BridgeTimeoutError, BridgeValidationError, BridgeVersionError } from "./errors.js";
import type { RouteRequest, RouteResponse, RouteResult } from "./types.js";

export const SDK_VERSION = "0.14.0";
const EXPECTED_BINARY_VERSION = "0.14.0";
export interface LocalOptions { binaryPath?: string; defaultTimeoutMs?: number; verifyVersion?: boolean }
export interface RouteOptions { timeoutMs?: number; signal?: AbortSignal }

type ProcessResult = { code: number; stdout: string; stderr: string };
export class BridgeClient {
  readonly binaryPath: string;
  private constructor(binaryPath: string, private readonly defaultTimeoutMs?: number) { this.binaryPath = binaryPath; }
  static async local(options: LocalOptions = {}): Promise<BridgeClient> {
    const client = new BridgeClient(resolveBinary(options.binaryPath), options.defaultTimeoutMs);
    if (options.verifyVersion !== false) {
      const version = await client.version();
      if (version !== EXPECTED_BINARY_VERSION) throw new BridgeVersionError(`incompatible BRIDGE binary: expected ${EXPECTED_BINARY_VERSION}, got ${version}`);
    }
    return client;
  }
  async version(): Promise<string> { const p = await this.run(["version"], undefined, { timeoutMs: 10000 }); if (p.code !== 0) throw this.processError(p); return p.stdout.trim(); }
  async route(request: RouteRequest, options: RouteOptions = {}): Promise<RouteResponse> {
    const p = await this.run(["route"], JSON.stringify(request), { timeoutMs: options.timeoutMs ?? this.defaultTimeoutMs, signal: options.signal });
    if (p.code !== 0) throw this.processError(p);
    let result: RouteResult;
    try { result = JSON.parse(p.stdout) as RouteResult; } catch { throw new BridgeProtocolError("BRIDGE stdout was not valid JSON", p.code, p.stdout, p.stderr); }
    if (result.schema_version !== "bridge.route.result.v2") throw new BridgeProtocolError("unsupported route result schema", p.code, p.stdout, p.stderr);
    return { result, warnings: p.stderr.split(/\r?\n/).filter(Boolean).map(v => v.replace(/^warning:\s*/, "")) };
  }
  private run(args: string[], input?: string, options: RouteOptions = {}): Promise<ProcessResult> {
    return new Promise((resolve, reject) => {
      if (options.signal?.aborted) { reject(new BridgeCancelledError("BRIDGE process was cancelled")); return; }
      const child = spawn(this.binaryPath, args, { shell: false, windowsHide: true, stdio: ["pipe", "pipe", "pipe"] });
      let stdout = "", stderr = "", settled = false;
      child.stdout.setEncoding("utf8"); child.stderr.setEncoding("utf8");
      child.stdout.on("data", (d: string) => stdout += d); child.stderr.on("data", (d: string) => stderr += d);
      const finishReject = (err: Error) => { if (settled) return; settled = true; cleanup(); child.kill("SIGKILL"); reject(err); };
      const timer = options.timeoutMs ? setTimeout(() => finishReject(new BridgeTimeoutError("BRIDGE process timed out", undefined, stdout, stderr)), options.timeoutMs) : undefined;
      const onAbort = () => finishReject(new BridgeCancelledError("BRIDGE process was cancelled", undefined, stdout, stderr));
      options.signal?.addEventListener("abort", onAbort, { once: true });
      const cleanup = () => { if (timer) clearTimeout(timer); options.signal?.removeEventListener("abort", onAbort); };
      child.on("error", (err: Error) => finishReject(new BridgeInternalError(`failed to start BRIDGE: ${err.message}`, undefined, stdout, stderr)));
      child.on("close", (code: number | null) => { if (settled) return; settled = true; cleanup(); resolve({ code: code ?? 10, stdout, stderr }); });
      if (input !== undefined) child.stdin.end(input); else child.stdin.end();
    });
  }
  private processError(p: ProcessResult): BridgeError {
    const C = p.code === 2 ? BridgeValidationError : p.code === 3 ? BridgeIOError : p.code === 4 ? BridgeTimeoutError : p.code === 5 ? BridgeAcceptanceError : BridgeInternalError;
    return new C(p.stderr.trim() || `BRIDGE exited with code ${p.code}`, p.code, p.stdout, p.stderr);
  }
}
