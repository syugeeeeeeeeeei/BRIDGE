from __future__ import annotations

import os
import socket
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Mapping

from .client import BridgeClient
from .discovery import resolve_binary
from .errors import BridgeIOError, BridgeTimeoutError


class BridgeServer:
    """SDK-managed local BRIDGE HTTP server process."""

    def __init__(self, process: subprocess.Popen[str], base_url: str):
        self.process = process
        self.base_url = base_url

    @classmethod
    def start(
        cls,
        binary_path: str | Path | None = None,
        *,
        host: str = "127.0.0.1",
        port: int | None = None,
        config_path: str | Path | None = None,
        startup_timeout: float = 10.0,
        env: Mapping[str, str] | None = None,
    ) -> "BridgeServer":
        binary = resolve_binary(binary_path)
        selected_port = port if port is not None else _reserve_port(host)
        listen = f"{host}:{selected_port}"
        child_env = os.environ.copy()
        child_env.update(env or {})
        child_env["BRIDGE_SERVER_LISTEN"] = listen
        args = [str(binary), "serve"]
        if config_path is not None:
            args.extend(["--config", str(config_path)])
        try:
            process = subprocess.Popen(
                args,
                stdin=subprocess.DEVNULL,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                env=child_env,
                shell=False,
            )
        except OSError as exc:
            raise BridgeIOError(f"failed to start BRIDGE server: {exc}") from exc
        server = cls(process, f"http://{listen}")
        try:
            server.wait_until_ready(timeout=startup_timeout)
        except Exception:
            server.stop()
            raise
        return server

    def wait_until_ready(self, *, timeout: float = 10.0) -> None:
        deadline = time.monotonic() + timeout
        last_error: Exception | None = None
        while time.monotonic() < deadline:
            if self.process.poll() is not None:
                stderr = self.process.stderr.read() if self.process.stderr else ""
                raise BridgeIOError(f"BRIDGE server exited during startup: {stderr.strip()}")
            try:
                with urllib.request.urlopen(f"{self.base_url}/readyz", timeout=0.5) as response:
                    if response.status == 200:
                        return
            except (urllib.error.URLError, TimeoutError) as exc:
                last_error = exc
                time.sleep(0.05)
        raise BridgeTimeoutError(f"BRIDGE server did not become ready: {last_error}")

    def client(self, *, default_timeout: float | None = None, verify_compatibility: bool = True, headers: Mapping[str, str] | None = None) -> BridgeClient:
        return BridgeClient.server(self.base_url, default_timeout=default_timeout, verify_compatibility=verify_compatibility, headers=headers)

    def stop(self, *, timeout: float = 10.0) -> None:
        if self.process.poll() is not None:
            return
        self.process.terminate()
        try:
            self.process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait(timeout=2)
        finally:
            if self.process.stdout is not None:
                self.process.stdout.close()
            if self.process.stderr is not None:
                self.process.stderr.close()

    def __enter__(self) -> "BridgeServer":
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        self.stop()


def _reserve_port(host: str) -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind((host, 0))
        return int(sock.getsockname()[1])
