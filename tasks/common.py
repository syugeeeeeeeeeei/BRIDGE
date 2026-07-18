from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tarfile
from typing import Iterable


def find_repository_root(start: Path) -> Path:
    """Locate the repository root from the script location."""

    current = start.resolve()

    if current.is_file():
        current = current.parent

    for candidate in (current, *current.parents):
        if (
            (candidate / "mise.toml").is_file()
            and (candidate / "go.mod").is_file()
            and (candidate / "src").is_dir()
        ):
            return candidate

    raise RuntimeError(
        "BRIDGE repository root could not be located. "
        "Expected mise.toml, go.mod, and src/ in a parent directory of "
        f"{start.resolve()}."
    )


ROOT = find_repository_root(Path(__file__))
CLI_PACKAGE = "./src/products/cli/cmd/bridge"

BUILD_DIR = ROOT / "build"
SDK_BUILD_DIR = BUILD_DIR / "sdk-binaries"

PYTHON_SDK = ROOT / "src" / "sdk" / "python"
PYTHON_PACKAGE = PYTHON_SDK / "src" / "bridge_sdk"

TYPESCRIPT_SDK = ROOT / "src" / "sdk" / "typescript"

TARGETS = (
    ("linux-amd64", "linux", "amd64", "bridge"),
    ("linux-arm64", "linux", "arm64", "bridge"),
    ("darwin-amd64", "darwin", "amd64", "bridge"),
    ("darwin-arm64", "darwin", "arm64", "bridge"),
    ("windows-amd64", "windows", "amd64", "bridge.exe"),
)

GITHUB_OWNER = "syugeeeeeeeeeei"
NPM_PACKAGE_NAME = f"@{GITHUB_OWNER}/bridge"
GITHUB_PACKAGES_REGISTRY = "https://npm.pkg.github.com"
MAX_NPM_PACKAGE_BYTES = 250 * 1024 * 1024
SEMVER_PATTERN = r"\d+\.\d+\.\d+"


def resolve_command(args: Iterable[str]) -> list[str]:
    """Resolve an executable through PATH without invoking a shell."""

    command = [str(value) for value in args]

    if not command:
        raise ValueError("command must not be empty")

    executable = command[0]

    if (
        not Path(executable).is_absolute()
        and not any(separator in executable for separator in ("/", "\\"))
    ):
        resolved = shutil.which(executable)

        if resolved is None:
            raise FileNotFoundError(
                f"required command was not found on PATH: {executable}. "
                "Run 'mise install' and retry."
            )

        command[0] = resolved

    return command


def run(
    args: Iterable[str],
    *,
    cwd: Path = ROOT,
    env: dict[str, str] | None = None,
) -> None:
    """Run a command and raise an exception when it fails."""

    command = resolve_command(args)

    print("+", subprocess.list2cmdline(command), flush=True)

    subprocess.run(
        command,
        cwd=cwd,
        env=env,
        check=True,
        shell=False,
    )


def output(
    args: Iterable[str],
    *,
    cwd: Path = ROOT,
    env: dict[str, str] | None = None,
) -> str:
    """Run a command and return its standard output."""

    command = resolve_command(args)

    print("+", subprocess.list2cmdline(command), flush=True)

    return subprocess.check_output(
        command,
        cwd=cwd,
        env=env,
        text=True,
        shell=False,
    ).strip()


def git_output(args: Iterable[str]) -> str:
    """Run a git command and return stdout."""

    return output(["git", *args])


def current_commit() -> str:
    """Return the current commit SHA, or unknown outside git."""

    try:
        return git_output(["rev-parse", "HEAD"])
    except (subprocess.CalledProcessError, FileNotFoundError):
        return "unknown"


def current_build_time() -> str:
    """Return the UTC build timestamp used in manifests and ldflags."""

    from datetime import datetime, timezone

    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace(
        "+00:00",
        "Z",
    )


def source_dirty_state() -> str:
    """Return whether the git worktree has local modifications."""

    try:
        return "true" if git_output(["status", "--porcelain"]) else "false"
    except (subprocess.CalledProcessError, FileNotFoundError):
        return "unknown"


def bridge_ldflags(
    *,
    version: str,
    commit: str,
    build_time: str,
    dirty: str,
    stripped: bool,
) -> str:
    """Build ldflags for BRIDGE release metadata."""

    values = [
        f"-X github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo.Version={version}",
        f"-X github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo.Commit={commit}",
        f"-X github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo.BuildTime={build_time}",
        f"-X github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo.Dirty={dirty}",
    ]

    if stripped:
        values[:0] = ["-s", "-w"]

    return " ".join(values)


def executable_name() -> str:
    """Return the native BRIDGE executable filename."""

    return "bridge.exe" if os.name == "nt" else "bridge"


def normalize_bridge_version(raw: str) -> str:
    """Extract a SemVer version from BRIDGE CLI version output."""

    text = raw.strip()

    try:
        data = json.loads(text)
    except json.JSONDecodeError:
        data = None

    if isinstance(data, dict) and isinstance(data.get("version"), str):
        return data["version"]

    match = re.search(r"\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?", text)

    if not match:
        raise RuntimeError(f"could not parse BRIDGE version output: {raw!r}")

    return match.group(0)


def sha256(path: Path) -> str:
    """Calculate the SHA-256 digest of a file."""

    digest = hashlib.sha256()

    with path.open("rb") as stream:
        for chunk in iter(
            lambda: stream.read(1024 * 1024),
            b"",
        ):
            digest.update(chunk)

    return digest.hexdigest()


def make_executable(path: Path) -> None:
    """Add executable permission to Unix binaries."""

    if path.suffix.lower() == ".exe":
        return

    path.chmod(
        path.stat().st_mode
        | stat.S_IXUSR
        | stat.S_IXGRP
        | stat.S_IXOTH
    )


def sync_tree(
    source: Path,
    destination: Path,
) -> None:
    """Replace a destination directory with a copied source tree."""

    if destination.exists():
        shutil.rmtree(destination)

    shutil.copytree(source, destination)

    for path in destination.rglob("bridge"):
        make_executable(path)

