import { spawn } from "node:child_process";
import { resolveBinary } from "./discovery.js";
import {
  BridgeAcceptanceError,
  BridgeCancelledError,
  BridgeError,
  BridgeIOError,
  BridgeInternalError,
  BridgeProtocolError,
  BridgeTimeoutError,
  BridgeValidationError,
  BridgeVersionError,
} from "./errors.js";
import type { RouteRequest, RouteResponse, RouteResult } from "./types.js";

export const SDK_VERSION = "0.16.0";
const REQUIRED_ROUTE_SCHEMA = "bridge.route.result.v1";

export interface SolverOptions { binaryPath?: string; defaultTimeoutMs?: number; verifyCompatibility?: boolean }
export interface ServerOptions { defaultTimeoutMs?: number; verifyCompatibility?: boolean; headers?: Record<string, string> }
export interface RouteOptions { timeoutMs?: number; signal?: AbortSignal }

type ProcessResult = { code: number; stdout: string; stderr: string };

interface Transport {
  version(): Promise<string>;
  capabilities(): Promise<Record<string, any>>;
  route(request: RouteRequest, options?: RouteOptions): Promise<RouteResponse>;
}

function normalizeRequest(request: RouteRequest): RouteRequest {
  return structuredClone(request);
}

function validateResult(result: RouteResult, stdout = "", stderr = ""): RouteResult {
  if (result.schema_version !== REQUIRED_ROUTE_SCHEMA) {
    throw new BridgeProtocolError("unsupported route result schema", undefined, stdout, stderr);
  }
  return result;
}

class LocalProcessTransport implements Transport {
  readonly binaryPath: string;
  constructor(binaryPath: string | undefined, private readonly defaultTimeoutMs?: number) {
    this.binaryPath = resolveBinary(binaryPath);
  }

  async version(): Promise<string> {
    const p = await this.run(["version", "--output", "json"], undefined, { timeoutMs: 10000 });
    if (p.code !== 0) throw this.processError(p);
    try { return String(JSON.parse(p.stdout).version); }
    catch { throw new BridgeProtocolError("BRIDGE version output was not valid JSON", p.code, p.stdout, p.stderr); }
  }

  async capabilities(): Promise<Record<string, any>> {
    const p = await this.run(["capabilities"], undefined, { timeoutMs: 10000 });
    if (p.code !== 0) throw this.processError(p);
    try { return JSON.parse(p.stdout) as Record<string, any>; }
    catch { throw new BridgeProtocolError("BRIDGE capabilities output was not valid JSON", p.code, p.stdout, p.stderr); }
  }

  async route(request: RouteRequest, options: RouteOptions = {}): Promise<RouteResponse> {
    const p = await this.run(["route"], JSON.stringify(normalizeRequest(request)), {
      timeoutMs: options.timeoutMs ?? this.defaultTimeoutMs,
      signal: options.signal,
    });
    if (p.code !== 0) throw this.processError(p);
    let result: RouteResult;
    try { result = JSON.parse(p.stdout) as RouteResult; }
    catch { throw new BridgeProtocolError("BRIDGE stdout was not valid JSON", p.code, p.stdout, p.stderr); }
    validateResult(result, p.stdout, p.stderr);
    return { result, warnings: p.stderr.split(/\r?\n/).filter(Boolean).map(v => v.replace(/^warning:\s*/, "")) };
  }

  private run(args: string[], input?: string, options: RouteOptions = {}): Promise<ProcessResult> {
    return new Promise((resolve, reject) => {
      if (options.signal?.aborted) { reject(new BridgeCancelledError("BRIDGE process was cancelled")); return; }
      const child = spawn(this.binaryPath, args, { shell: false, windowsHide: true, stdio: ["pipe", "pipe", "pipe"] });
      let stdout = "", stderr = "", settled = false;
      child.stdout.setEncoding("utf8"); child.stderr.setEncoding("utf8");
      child.stdout.on("data", (d: string) => stdout += d); child.stderr.on("data", (d: string) => stderr += d);
      const cleanup = () => { if (timer) clearTimeout(timer); options.signal?.removeEventListener("abort", onAbort); };
      const finishReject = (err: Error) => { if (settled) return; settled = true; cleanup(); child.kill("SIGKILL"); reject(err); };
      const timer = options.timeoutMs ? setTimeout(() => finishReject(new BridgeTimeoutError("BRIDGE process timed out", undefined, stdout, stderr)), options.timeoutMs) : undefined;
      const onAbort = () => finishReject(new BridgeCancelledError("BRIDGE process was cancelled", undefined, stdout, stderr));
      options.signal?.addEventListener("abort", onAbort, { once: true });
      child.on("error", (err: Error) => finishReject(new BridgeInternalError(`failed to start BRIDGE: ${err.message}`, undefined, stdout, stderr)));
      child.on("close", (code: number | null) => { if (settled) return; settled = true; cleanup(); resolve({ code: code ?? 10, stdout, stderr }); });
      if (input !== undefined) child.stdin.end(input); else child.stdin.end();
    });
  }

  private processError(p: ProcessResult): BridgeError {
    const C = p.code === 2 ? BridgeValidationError : p.code === 3 ? BridgeIOError : p.code === 4 ? BridgeTimeoutError : p.code === 5 ? BridgeAcceptanceError : BridgeInternalError;
    const first = p.stderr.split(/\r?\n/).map(v => v.trim()).find(Boolean) ?? "";
    let message = p.stderr.trim() || `BRIDGE exited with code ${p.code}`;
    let logicalCode = "";
    if (first.startsWith("error:")) {
      const detail = first.slice("error:".length).trim();
      const match = /^([A-Z][A-Z0-9_]+):\s*(.*)$/.exec(detail);
      if (match) { logicalCode = match[1]; message = match[2] || detail; }
    }
    const category = p.code === 2 ? "validation" : p.code === 3 ? "io" : p.code === 4 ? "timeout" : p.code === 5 ? "acceptance" : "internal";
    return new C(message, p.code, p.stdout, p.stderr, logicalCode, category, [3, 4, 10].includes(p.code));
  }
}

class HttpTransport implements Transport {
  readonly baseUrl: string;
  constructor(baseUrl: string, private readonly defaultTimeoutMs?: number, private readonly headers: Record<string, string> = {}) {
    this.baseUrl = baseUrl.replace(/\/+$/, "");
    if (!/^https?:\/\//.test(this.baseUrl)) throw new TypeError("baseUrl must start with http:// or https://");
  }

  async version(): Promise<string> {
    const capabilities = await this.capabilities();
    if (capabilities.application_version === undefined) throw new BridgeProtocolError("capabilities response does not contain application_version");
    return String(capabilities.application_version);
  }

  async capabilities(): Promise<Record<string, any>> {
    return await this.jsonRequest("GET", "/v1/capabilities", undefined, { timeoutMs: 10000 });
  }

  async route(request: RouteRequest, options: RouteOptions = {}): Promise<RouteResponse> {
    const result = await this.jsonRequest("POST", "/v1/routes", normalizeRequest(request), {
      timeoutMs: options.timeoutMs ?? this.defaultTimeoutMs,
      signal: options.signal,
    }) as RouteResult;
    validateResult(result);
    return { result, warnings: [] };
  }

  private async jsonRequest(method: string, path: string, body?: unknown, options: RouteOptions = {}): Promise<any> {
    const controller = new AbortController();
    const onAbort = () => controller.abort();
    if (options.signal?.aborted) throw new BridgeCancelledError("BRIDGE HTTP request was cancelled");
    options.signal?.addEventListener("abort", onAbort, { once: true });
    const timer = options.timeoutMs ? setTimeout(() => controller.abort(), options.timeoutMs) : undefined;
    try {
      const response = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers: { Accept: "application/json", ...(body === undefined ? {} : { "Content-Type": "application/json" }), ...this.headers },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal,
      });
      const raw = await response.text();
      let payload: any;
      try { payload = raw ? JSON.parse(raw) : {}; }
      catch { throw new BridgeProtocolError("BRIDGE HTTP response was not valid JSON", response.status, raw, ""); }
      if (!response.ok) this.throwHttpError(response.status, payload, raw);
      return payload;
    } catch (error) {
      if (error instanceof BridgeError) throw error;
      if ((error as Error).name === "AbortError") {
        if (options.signal?.aborted) throw new BridgeCancelledError("BRIDGE HTTP request was cancelled");
        throw new BridgeTimeoutError("BRIDGE HTTP request timed out");
      }
      throw new BridgeIOError(`BRIDGE HTTP request failed: ${(error as Error).message}`);
    } finally {
      if (timer) clearTimeout(timer);
      options.signal?.removeEventListener("abort", onAbort);
    }
  }

  private throwHttpError(status: number, payload: any, raw: string): never {
    const message = String(payload?.error?.message ?? `BRIDGE HTTP request failed with status ${status}`);
    const code = String(payload?.error?.code ?? "");
    const category = String(payload?.error?.category ?? "");
    const retryable = Boolean(payload?.error?.retryable ?? false);
    const requestId = String(payload?.error?.request_id ?? "");
    const args: [string, number, string, string, string, string, boolean, string] = [message, status, "", raw, code, category, retryable, requestId];
    if ([400, 404, 405, 415, 422].includes(status)) throw new BridgeValidationError(...args);
    if (status === 408 || status === 504 || code === "DEADLINE_EXCEEDED") throw new BridgeTimeoutError(...args);
    if (status === 429 || status === 503) throw new BridgeIOError(...args);
    throw new BridgeInternalError(...args);
  }
}

export class BridgeClient {
  private constructor(private readonly transport: Transport) {}

  static async solver(options: SolverOptions = {}): Promise<BridgeClient> {
    const client = new BridgeClient(new LocalProcessTransport(options.binaryPath, options.defaultTimeoutMs));
    await client.verify(options.verifyCompatibility);
    return client;
  }

  static async server(baseUrl: string, options: ServerOptions = {}): Promise<BridgeClient> {
    const client = new BridgeClient(new HttpTransport(baseUrl, options.defaultTimeoutMs, options.headers));
    await client.verify(options.verifyCompatibility);
    return client;
  }

  get binaryPath(): string {
    if (!(this.transport instanceof LocalProcessTransport)) throw new Error("binaryPath is available only for solver clients");
    return this.transport.binaryPath;
  }

  get baseUrl(): string {
    if (!(this.transport instanceof HttpTransport)) throw new Error("baseUrl is available only for server clients");
    return this.transport.baseUrl;
  }

  async version(): Promise<string> { return await this.transport.version(); }
  async capabilities(): Promise<Record<string, any>> { return await this.transport.capabilities(); }
  async route(request: RouteRequest, options: RouteOptions = {}): Promise<RouteResponse> { return await this.transport.route(request, options); }

  private async verify(verifyCompatibility = true): Promise<void> {
    if (!verifyCompatibility) return;
    const capabilities = await this.capabilities();
    const supported = capabilities.schemas?.route_response ?? [];
    if (!supported.includes(REQUIRED_ROUTE_SCHEMA)) {
      throw new BridgeVersionError(`incompatible BRIDGE endpoint: required route schema ${REQUIRED_ROUTE_SCHEMA} is not supported`);
    }
  }
}
