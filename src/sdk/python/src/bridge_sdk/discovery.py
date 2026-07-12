from __future__ import annotations
import os, platform, shutil
from importlib.resources import files
from pathlib import Path
from .errors import BridgeBinaryNotFoundError, BridgeBinaryPermissionError

def _platform_key() -> str:
    system = platform.system().lower()
    machine = platform.machine().lower()
    os_key = {"linux":"linux", "windows":"windows", "darwin":"darwin"}.get(system)
    arch_key = {"x86_64":"amd64", "amd64":"amd64", "aarch64":"arm64", "arm64":"arm64"}.get(machine)
    if not os_key or not arch_key:
        raise BridgeBinaryNotFoundError(f"unsupported platform: {system}/{machine}")
    return f"{os_key}-{arch_key}"

def resolve_binary(binary_path: str | os.PathLike[str] | None = None) -> Path:
    candidates = []
    if binary_path:
        candidates.append(Path(binary_path))
    if os.environ.get("BRIDGE_BINARY"):
        candidates.append(Path(os.environ["BRIDGE_BINARY"]))
    exe = "bridge.exe" if os.name == "nt" else "bridge"
    candidates.append(Path(str(files("bridge_sdk").joinpath("bin", _platform_key(), exe))))
    path_hit = shutil.which("bridge")
    if path_hit:
        candidates.append(Path(path_hit))
    for candidate in candidates:
        if candidate.is_file():
            if os.name != "nt" and not os.access(candidate, os.X_OK):
                raise BridgeBinaryPermissionError(f"BRIDGE binary is not executable: {candidate}")
            return candidate
    raise BridgeBinaryNotFoundError("BRIDGE binary was not found")
