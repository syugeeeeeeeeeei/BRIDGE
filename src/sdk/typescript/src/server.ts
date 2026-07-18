import { spawn } from "node:child_process";
import { createServer } from "node:net";
import { resolveBinary } from "./discovery.js";
import { BridgeClient } from "./client.js";
import { BridgeIOError, BridgeTimeoutError } from "./errors.js";

export interface BridgeServerStartOptions {
  binaryPath?: string;
  host?: string;
  port?: number;
  configPath?: string;
  startupTimeoutMs?: number;
  env?: Record<string, string>;
}

export class BridgeServer {
  private constructor(readonly process: any, readonly baseUrl: string) {}

  static async start(options: BridgeServerStartOptions = {}): Promise<BridgeServer> {
    const host = options.host ?? "127.0.0.1";
    const port = options.port ?? await reservePort(host);
    const listen = `${host}:${port}`;
    const args = ["serve"];
    if (options.configPath) args.push("--config", options.configPath);
    const child = spawn(resolveBinary(options.binaryPath), args, {
      shell: false,
      windowsHide: true,
      stdio: ["pipe", "pipe", "pipe"],
      env: { ...process.env, ...options.env, BRIDGE_SERVER_LISTEN: listen },
    });
    child.stdin.end();
    const server = new BridgeServer(child, `http://${listen}`);
    try { await server.waitUntilReady(options.startupTimeoutMs ?? 10000); }
    catch (error) { await server.stop(); throw error; }
    return server;
  }

  async client(options: { defaultTimeoutMs?: number; verifyCompatibility?: boolean; headers?: Record<string, string> } = {}): Promise<BridgeClient> {
    return await BridgeClient.server(this.baseUrl, options);
  }

  async waitUntilReady(timeoutMs = 10000): Promise<void> {
    const deadline = Date.now() + timeoutMs;
    let stderr = "";
    this.process.stderr.setEncoding("utf8");
    this.process.stderr.on("data", (chunk: string) => stderr += chunk);
    while (Date.now() < deadline) {
      if (this.process.exitCode !== null) throw new BridgeIOError(`BRIDGE server exited during startup: ${stderr.trim()}`);
      try {
        const response = await fetch(`${this.baseUrl}/readyz`);
        if (response.ok) return;
      } catch { /* retry */ }
      await new Promise(resolve => setTimeout(resolve, 50));
    }
    throw new BridgeTimeoutError("BRIDGE server did not become ready");
  }

  async stop(timeoutMs = 10000): Promise<void> {
    if (this.process.exitCode !== null) return;
    await new Promise<void>(resolve => {
      let settled = false;
      const finish = () => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        resolve();
      };
      const timer = setTimeout(() => {
        if (this.process.exitCode === null) this.process.kill("SIGKILL");
        finish();
      }, timeoutMs);
      this.process.once("exit", finish);
      this.process.kill("SIGTERM");
      if (this.process.exitCode !== null) finish();
    });
  }
}

async function reservePort(host: string): Promise<number> {
  return await new Promise((resolve, reject) => {
    const server = createServer();
    server.once("error", reject);
    server.listen(0, host, () => {
      const address = server.address();
      if (typeof address === "string" || address === null) { server.close(); reject(new Error("failed to reserve port")); return; }
      const port = address.port;
      server.close((error?: Error) => error ? reject(error) : resolve(port));
    });
  });
}
