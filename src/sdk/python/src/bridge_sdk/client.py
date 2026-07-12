from __future__ import annotations
import json, subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Mapping
from .discovery import resolve_binary
from .errors import *

SDK_VERSION = "0.14.0"
EXPECTED_BINARY_VERSION = "0.14.0"

@dataclass(frozen=True)
class RouteResponse:
    result: dict[str, Any]
    warnings: tuple[str, ...] = ()

class BridgeClient:
    def __init__(self, binary_path=None, *, default_timeout: float | None = None, verify_version: bool = True):
        self.binary_path: Path = resolve_binary(binary_path)
        self.default_timeout = default_timeout
        if verify_version:
            version = self.version()
            if version != EXPECTED_BINARY_VERSION:
                raise BridgeVersionError(f"incompatible BRIDGE binary: expected {EXPECTED_BINARY_VERSION}, got {version}")

    @classmethod
    def local(cls, binary_path=None, **kwargs):
        return cls(binary_path, **kwargs)

    def version(self) -> str:
        try:
            cp = subprocess.run([str(self.binary_path), "version"], capture_output=True, text=True, timeout=10, shell=False)
        except subprocess.TimeoutExpired as exc:
            raise BridgeTimeoutError("BRIDGE version check timed out") from exc
        if cp.returncode != 0:
            raise BridgeInternalError("BRIDGE version check failed", exit_code=cp.returncode, stdout=cp.stdout, stderr=cp.stderr)
        return cp.stdout.strip()

    def route(self, request: Mapping[str, Any], *, timeout: float | None = None) -> RouteResponse:
        payload = json.dumps(request, ensure_ascii=False, separators=(",", ":"))
        try:
            cp = subprocess.run([str(self.binary_path), "route"], input=payload, capture_output=True, text=True,
                                timeout=self.default_timeout if timeout is None else timeout, shell=False)
        except subprocess.TimeoutExpired as exc:
            raise BridgeTimeoutError("BRIDGE route timed out", stdout=exc.stdout or "", stderr=exc.stderr or "") from exc
        if cp.returncode != 0:
            self._raise_process_error(cp.returncode, cp.stdout, cp.stderr)
        try:
            result = json.loads(cp.stdout)
        except json.JSONDecodeError as exc:
            raise BridgeProtocolError("BRIDGE stdout was not valid JSON", exit_code=cp.returncode, stdout=cp.stdout, stderr=cp.stderr) from exc
        if result.get("schema_version") != "bridge.route.result.v1":
            raise BridgeProtocolError("unsupported route result schema", stdout=cp.stdout, stderr=cp.stderr)
        warnings = tuple(line.removeprefix("warning:").strip() for line in cp.stderr.splitlines() if line.strip())
        return RouteResponse(result=result, warnings=warnings)

    async def route_async(self, request: Mapping[str, Any], *, timeout: float | None = None) -> RouteResponse:
        import asyncio
        payload = json.dumps(request, ensure_ascii=False, separators=(",", ":")).encode()
        proc = await asyncio.create_subprocess_exec(str(self.binary_path), "route", stdin=asyncio.subprocess.PIPE, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE)
        try:
            stdout_b, stderr_b = await asyncio.wait_for(proc.communicate(payload), self.default_timeout if timeout is None else timeout)
        except (asyncio.TimeoutError, asyncio.CancelledError) as exc:
            proc.kill()
            await proc.wait()
            if isinstance(exc, asyncio.CancelledError):
                raise BridgeCancelledError("BRIDGE route was cancelled") from exc
            raise BridgeTimeoutError("BRIDGE route timed out") from exc
        stdout, stderr = stdout_b.decode(), stderr_b.decode()
        if proc.returncode != 0:
            self._raise_process_error(proc.returncode or 10, stdout, stderr)
        try:
            result = json.loads(stdout)
        except json.JSONDecodeError as exc:
            raise BridgeProtocolError("BRIDGE stdout was not valid JSON", stdout=stdout, stderr=stderr) from exc
        if result.get("schema_version") != "bridge.route.result.v1":
            raise BridgeProtocolError("unsupported route result schema", stdout=stdout, stderr=stderr)
        warnings = tuple(line.removeprefix("warning:").strip() for line in stderr.splitlines() if line.strip())
        return RouteResponse(result=result, warnings=warnings)

    @staticmethod
    def _raise_process_error(code: int, stdout: str, stderr: str):
        cls = {2: BridgeValidationError, 3: BridgeIOError, 4: BridgeTimeoutError,
               5: BridgeAcceptanceError, 10: BridgeInternalError}.get(code, BridgeInternalError)
        raise cls(stderr.strip() or f"BRIDGE exited with code {code}", exit_code=code, stdout=stdout, stderr=stderr)
