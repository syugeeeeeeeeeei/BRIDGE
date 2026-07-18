from __future__ import annotations

import asyncio
import json
import subprocess
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Mapping, Protocol

from .discovery import resolve_binary
from .errors import (
    BridgeAcceptanceError,
    BridgeCancelledError,
    BridgeError,
    BridgeIOError,
    BridgeInternalError,
    BridgeProtocolError,
    BridgeTimeoutError,
    BridgeValidationError,
    BridgeVersionError,
)

SDK_VERSION = "0.15.0"
REQUIRED_ROUTE_SCHEMA = "bridge.route.result.v1"


@dataclass(frozen=True)
class RouteResponse:
    result: dict[str, Any]
    warnings: tuple[str, ...] = ()


class _Transport(Protocol):
    def version(self) -> str: ...
    def capabilities(self) -> dict[str, Any]: ...
    def route(self, request: Mapping[str, Any], timeout: float | None) -> RouteResponse: ...
    async def route_async(self, request: Mapping[str, Any], timeout: float | None) -> RouteResponse: ...


def _normalize_request(request: Mapping[str, Any]) -> dict[str, Any]:
    return json.loads(json.dumps(request))


def _validate_result(result: dict[str, Any], *, stdout: str = "", stderr: str = "") -> RouteResponse:
    if result.get("schema_version") != REQUIRED_ROUTE_SCHEMA:
        raise BridgeProtocolError("unsupported route result schema", stdout=stdout, stderr=stderr)
    return RouteResponse(result=result)


class _LocalProcessTransport:
    def __init__(self, binary_path: str | Path | None, default_timeout: float | None):
        self.binary_path: Path = resolve_binary(binary_path)
        self.default_timeout = default_timeout

    def _run(self, args: list[str], *, input_text: str | None = None, timeout: float | None = None) -> subprocess.CompletedProcess[str]:
        try:
            return subprocess.run(
                [str(self.binary_path), *args],
                input=input_text,
                capture_output=True,
                text=True,
                timeout=timeout,
                shell=False,
            )
        except subprocess.TimeoutExpired as exc:
            raise BridgeTimeoutError("BRIDGE process timed out", stdout=exc.stdout or "", stderr=exc.stderr or "") from exc
        except OSError as exc:
            raise BridgeIOError(f"failed to start BRIDGE: {exc}") from exc

    def version(self) -> str:
        cp = self._run(["version", "--output", "json"], timeout=10)
        if cp.returncode != 0:
            self._raise_process_error(cp.returncode, cp.stdout, cp.stderr)
        try:
            return str(json.loads(cp.stdout)["version"])
        except (json.JSONDecodeError, KeyError, TypeError) as exc:
            raise BridgeProtocolError("BRIDGE version output was not valid JSON", stdout=cp.stdout, stderr=cp.stderr) from exc

    def capabilities(self) -> dict[str, Any]:
        cp = self._run(["capabilities"], timeout=10)
        if cp.returncode != 0:
            self._raise_process_error(cp.returncode, cp.stdout, cp.stderr)
        try:
            return json.loads(cp.stdout)
        except json.JSONDecodeError as exc:
            raise BridgeProtocolError("BRIDGE capabilities output was not valid JSON", stdout=cp.stdout, stderr=cp.stderr) from exc

    def route(self, request: Mapping[str, Any], timeout: float | None) -> RouteResponse:
        payload = json.dumps(_normalize_request(request), ensure_ascii=False, separators=(",", ":"))
        cp = self._run(["route"], input_text=payload, timeout=self.default_timeout if timeout is None else timeout)
        if cp.returncode != 0:
            self._raise_process_error(cp.returncode, cp.stdout, cp.stderr)
        try:
            result = json.loads(cp.stdout)
        except json.JSONDecodeError as exc:
            raise BridgeProtocolError("BRIDGE stdout was not valid JSON", exit_code=cp.returncode, stdout=cp.stdout, stderr=cp.stderr) from exc
        response = _validate_result(result, stdout=cp.stdout, stderr=cp.stderr)
        warnings = tuple(line.removeprefix("warning:").strip() for line in cp.stderr.splitlines() if line.strip())
        return RouteResponse(result=response.result, warnings=warnings)

    async def route_async(self, request: Mapping[str, Any], timeout: float | None) -> RouteResponse:
        normalized = _normalize_request(request)
        payload = json.dumps(normalized, ensure_ascii=False, separators=(",", ":")).encode()
        proc = await asyncio.create_subprocess_exec(
            str(self.binary_path), "route",
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        try:
            stdout_b, stderr_b = await asyncio.wait_for(
                proc.communicate(payload), self.default_timeout if timeout is None else timeout
            )
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
        response = _validate_result(result, stdout=stdout, stderr=stderr)
        warnings = tuple(line.removeprefix("warning:").strip() for line in stderr.splitlines() if line.strip())
        return RouteResponse(result=response.result, warnings=warnings)

    @staticmethod
    def _raise_process_error(code: int, stdout: str, stderr: str) -> None:
        cls = {
            2: BridgeValidationError,
            3: BridgeIOError,
            4: BridgeTimeoutError,
            5: BridgeAcceptanceError,
            10: BridgeInternalError,
        }.get(code, BridgeInternalError)
        message = stderr.strip() or f"BRIDGE exited with code {code}"
        logical_code = ""
        category = ""
        first = next((line.strip() for line in stderr.splitlines() if line.strip()), "")
        if first.startswith("error:"):
            detail = first.removeprefix("error:").strip()
            head, sep, tail = detail.partition(":")
            if sep and head.replace("_", "").isalnum() and head.upper() == head:
                logical_code = head
                message = tail.strip() or detail
        if code == 2: category = "validation"
        elif code == 3: category = "io"
        elif code == 4: category = "timeout"
        elif code == 5: category = "acceptance"
        else: category = "internal"
        raise cls(message, exit_code=code, stdout=stdout, stderr=stderr, code=logical_code, category=category, retryable=code in (3, 4, 10))


class _HTTPTransport:
    def __init__(self, base_url: str, default_timeout: float | None, headers: Mapping[str, str] | None = None):
        self.base_url = base_url.rstrip("/")
        if not self.base_url.startswith(("http://", "https://")):
            raise ValueError("base_url must start with http:// or https://")
        self.default_timeout = default_timeout
        self.headers = dict(headers or {})

    def _json_request(self, method: str, path: str, *, body: Mapping[str, Any] | None = None, timeout: float | None = None) -> tuple[dict[str, Any], Mapping[str, str]]:
        data = None if body is None else json.dumps(body, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        req = urllib.request.Request(
            f"{self.base_url}{path}", data=data, method=method,
            headers={**{"Accept": "application/json"}, **({"Content-Type": "application/json"} if data is not None else {}), **self.headers},
        )
        try:
            with urllib.request.urlopen(req, timeout=self.default_timeout if timeout is None else timeout) as response:
                raw = response.read().decode("utf-8")
                headers = dict(response.headers.items())
        except urllib.error.HTTPError as exc:
            raw = exc.read().decode("utf-8", errors="replace")
            self._raise_http_error(exc.code, raw)
        except TimeoutError as exc:
            raise BridgeTimeoutError("BRIDGE HTTP request timed out") from exc
        except urllib.error.URLError as exc:
            if isinstance(exc.reason, TimeoutError):
                raise BridgeTimeoutError("BRIDGE HTTP request timed out") from exc
            raise BridgeIOError(f"BRIDGE HTTP request failed: {exc.reason}") from exc
        try:
            return json.loads(raw), headers
        except json.JSONDecodeError as exc:
            raise BridgeProtocolError("BRIDGE HTTP response was not valid JSON", stdout=raw) from exc

    def version(self) -> str:
        capabilities = self.capabilities()
        try:
            return str(capabilities["application_version"])
        except KeyError as exc:
            raise BridgeProtocolError("capabilities response does not contain application_version") from exc

    def capabilities(self) -> dict[str, Any]:
        payload, _ = self._json_request("GET", "/v1/capabilities", timeout=10)
        return payload

    def route(self, request: Mapping[str, Any], timeout: float | None) -> RouteResponse:
        result, headers = self._json_request("POST", "/v1/routes", body=_normalize_request(request), timeout=timeout)
        response = _validate_result(result)
        warnings_header = headers.get("X-Bridge-Warnings", "")
        warnings = tuple(v.strip() for v in warnings_header.split(",") if v.strip())
        return RouteResponse(result=response.result, warnings=warnings)

    async def route_async(self, request: Mapping[str, Any], timeout: float | None) -> RouteResponse:
        return await asyncio.to_thread(self.route, request, timeout)

    @staticmethod
    def _raise_http_error(status: int, raw: str) -> None:
        message = raw or f"BRIDGE HTTP request failed with status {status}"
        code = ""
        category = ""
        retryable = False
        request_id = ""
        try:
            payload = json.loads(raw)
            error = payload.get("error", {})
            message = str(error.get("message") or message)
            code = str(error.get("code") or "")
            category = str(error.get("category") or "")
            retryable = bool(error.get("retryable", False))
            request_id = str(error.get("request_id") or "")
        except json.JSONDecodeError:
            pass
        if status in (400, 404, 405, 415, 422):
            cls: type[BridgeError] = BridgeValidationError
        elif status == 408 or status == 504 or code == "DEADLINE_EXCEEDED":
            cls = BridgeTimeoutError
        elif status == 429 or status == 503:
            cls = BridgeIOError
        else:
            cls = BridgeInternalError
        raise cls(message, exit_code=status, stderr=raw, code=code, category=category, retryable=retryable, request_id=request_id)


class BridgeClient:
    def __init__(self, transport: _Transport, *, verify_compatibility: bool = True):
        self._transport = transport
        if verify_compatibility:
            capabilities = self.capabilities()
            supported = capabilities.get("schemas", {}).get("route_response", [])
            if REQUIRED_ROUTE_SCHEMA not in supported:
                raise BridgeVersionError(
                    f"incompatible BRIDGE endpoint: required route schema {REQUIRED_ROUTE_SCHEMA} is not supported"
                )

    @classmethod
    def solver(cls, binary_path: str | Path | None = None, *, default_timeout: float | None = None, verify_compatibility: bool = True) -> "BridgeClient":
        return cls(_LocalProcessTransport(binary_path, default_timeout), verify_compatibility=verify_compatibility)

    @classmethod
    def server(cls, base_url: str, *, default_timeout: float | None = None, verify_compatibility: bool = True, headers: Mapping[str, str] | None = None) -> "BridgeClient":
        return cls(_HTTPTransport(base_url, default_timeout, headers), verify_compatibility=verify_compatibility)

    @property
    def binary_path(self) -> Path:
        transport = self._transport
        if not isinstance(transport, _LocalProcessTransport):
            raise AttributeError("binary_path is available only for solver clients")
        return transport.binary_path

    @property
    def base_url(self) -> str:
        transport = self._transport
        if not isinstance(transport, _HTTPTransport):
            raise AttributeError("base_url is available only for server clients")
        return transport.base_url

    def version(self) -> str:
        return self._transport.version()

    def capabilities(self) -> dict[str, Any]:
        return self._transport.capabilities()

    def route(self, request: Mapping[str, Any], *, timeout: float | None = None) -> RouteResponse:
        return self._transport.route(request, timeout)

    async def route_async(self, request: Mapping[str, Any], *, timeout: float | None = None) -> RouteResponse:
        return await self._transport.route_async(request, timeout)
