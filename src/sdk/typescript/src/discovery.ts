import { accessSync, constants, existsSync } from "node:fs";
import { delimiter, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { BridgeBinaryNotFoundError, BridgeBinaryPermissionError } from "./errors.js";

function platformKey(): string {
  const os = process.platform === "win32" ? "windows" : process.platform === "darwin" ? "darwin" : process.platform === "linux" ? "linux" : undefined;
  const arch = process.arch === "x64" ? "amd64" : process.arch === "arm64" ? "arm64" : undefined;
  if (!os || !arch) throw new BridgeBinaryNotFoundError(`unsupported platform: ${process.platform}/${process.arch}`);
  return `${os}-${arch}`;
}
function pathCandidate(): string | undefined {
  const exe = process.platform === "win32" ? "bridge.exe" : "bridge";
  for (const dir of (process.env.PATH ?? "").split(delimiter)) {
    const p = join(dir, exe); if (existsSync(p)) return p;
  }
  return undefined;
}
export function resolveBinary(binaryPath?: string): string {
  const exe = process.platform === "win32" ? "bridge.exe" : "bridge";
  const moduleDir = dirname(fileURLToPath(import.meta.url));
  const bundled = join(moduleDir, "..", "bin", platformKey(), exe);
  const candidates = [binaryPath, process.env.BRIDGE_BINARY, bundled, pathCandidate()].filter((v): v is string => Boolean(v));
  for (const p of candidates) {
    if (!existsSync(p)) continue;
    if (process.platform !== "win32") {
      try { accessSync(p, constants.X_OK); } catch { throw new BridgeBinaryPermissionError(`BRIDGE binary is not executable: ${p}`); }
    }
    return p;
  }
  throw new BridgeBinaryNotFoundError("BRIDGE binary was not found");
}
