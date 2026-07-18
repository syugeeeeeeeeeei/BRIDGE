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


def build_cli(
    *,
    debug: bool = False,
    release: bool = False,
) -> Path:
    """Build the BRIDGE CLI for the current platform."""

    BUILD_DIR.mkdir(parents=True, exist_ok=True)

    destination = BUILD_DIR / executable_name()
    version = bridge_version_from_source()
    commit = current_commit()
    build_time = current_build_time()
    dirty = source_dirty_state()

    args = [
        "go",
        "build",
        "-trimpath",
    ]

    args.append(
        "-ldflags="
        + bridge_ldflags(
            version=version,
            commit=commit,
            build_time=build_time,
            dirty=dirty,
            stripped=release or not debug,
        )
    )

    args.extend(
        [
            "-o",
            str(destination),
            CLI_PACKAGE,
        ]
    )

    run(args)

    print(f"built: {destination.relative_to(ROOT)}")

    return destination


def bridge_version_from_source() -> str:
    """Read the BRIDGE version from the single buildinfo source."""

    buildinfo_go = ROOT / "src" / "buildinfo" / "buildinfo.go"
    marker = 'Version   = "'

    for line in buildinfo_go.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped.startswith(marker) and stripped.endswith('"'):
            return stripped[len(marker) : -1]

    raise RuntimeError(f"could not find BRIDGE version in {buildinfo_go}")


def package_json() -> dict[str, object]:
    """Read the TypeScript SDK package metadata."""

    return json.loads((TYPESCRIPT_SDK / "package.json").read_text(encoding="utf-8"))


def package_version() -> str:
    """Return the TypeScript SDK package version."""

    version = package_json().get("version")

    if not isinstance(version, str):
        raise RuntimeError("src/sdk/typescript/package.json has no string version")

    return version


def sdk_version_from_source() -> str:
    """Read the TypeScript SDK_VERSION constant."""

    client_ts = TYPESCRIPT_SDK / "src" / "client.ts"
    match = re.search(
        rf'export const SDK_VERSION = "({SEMVER_PATTERN})";',
        client_ts.read_text(encoding="utf-8"),
    )

    if not match:
        raise RuntimeError(f"could not find SDK_VERSION in {client_ts}")

    return match.group(1)


def package_lock_version() -> str:
    """Return the TypeScript SDK package-lock root version."""

    lock_path = TYPESCRIPT_SDK / "package-lock.json"
    lock = json.loads(lock_path.read_text(encoding="utf-8"))
    version = lock.get("version")
    root_version = lock.get("packages", {}).get("", {}).get("version")

    if not isinstance(version, str) or version != root_version:
        raise RuntimeError("package-lock.json root versions are missing or inconsistent")

    return version


def project_versions() -> dict[str, str]:
    """Return every source-controlled project version."""

    return {
        "src/buildinfo/buildinfo.go": bridge_version_from_source(),
        "src/sdk/typescript/package.json": package_version(),
        "src/sdk/typescript/package-lock.json": package_lock_version(),
        "src/sdk/typescript/src/client.ts": sdk_version_from_source(),
    }


def require_project_versions_match() -> str:
    """Return the project version after ensuring all version files match."""

    versions = project_versions()
    unique = set(versions.values())

    if len(unique) != 1:
        details = ", ".join(f"{path}={version}" for path, version in versions.items())
        raise RuntimeError(f"project versions do not match: {details}")

    return next(iter(unique))


def parse_semver(version: str) -> tuple[int, int, int]:
    """Parse a strict X.Y.Z version."""

    if not re.fullmatch(SEMVER_PATTERN, version):
        raise RuntimeError(f"version must be SemVer X.Y.Z, got {version!r}")

    major, minor, patch = version.split(".")

    return int(major), int(minor), int(patch)


def format_semver(parts: tuple[int, int, int]) -> str:
    """Format a SemVer tuple."""

    return ".".join(str(value) for value in parts)


def bump_semver(version: str, operation: str, part: str) -> str:
    """Increment or decrement one SemVer part."""

    major, minor, patch = parse_semver(version)

    if operation == "add":
        if part == "major":
            return format_semver((major + 1, 0, 0))
        if part == "minor":
            return format_semver((major, minor + 1, 0))
        if part == "patch":
            return format_semver((major, minor, patch + 1))
    elif operation == "sub":
        if part == "major" and major > 0:
            return format_semver((major - 1, 0, 0))
        if part == "minor" and minor > 0:
            return format_semver((major, minor - 1, 0))
        if part == "patch" and patch > 0:
            return format_semver((major, minor, patch - 1))

    raise RuntimeError(f"cannot {operation} {part} for version {version}")


def replace_once(path: Path, pattern: str, replacement: str) -> None:
    """Replace exactly one regex occurrence in a text file."""

    text = path.read_text(encoding="utf-8")
    updated, count = re.subn(pattern, replacement, text, count=1)

    if count != 1:
        raise RuntimeError(f"expected exactly one version replacement in {path}")

    path.write_text(updated, encoding="utf-8")


def set_project_version(version: str) -> None:
    """Set every source-controlled project version file."""

    parse_semver(version)

    buildinfo_go = ROOT / "src" / "buildinfo" / "buildinfo.go"
    client_ts = TYPESCRIPT_SDK / "src" / "client.ts"
    package_json_path = TYPESCRIPT_SDK / "package.json"
    package_lock_path = TYPESCRIPT_SDK / "package-lock.json"

    replace_once(
        buildinfo_go,
        rf'Version\s+= "{SEMVER_PATTERN}"',
        f'Version   = "{version}"',
    )
    replace_once(
        client_ts,
        rf'export const SDK_VERSION = "{SEMVER_PATTERN}";',
        f'export const SDK_VERSION = "{version}";',
    )

    package_data = json.loads(package_json_path.read_text(encoding="utf-8"))
    package_data["version"] = version
    package_json_path.write_text(
        json.dumps(package_data, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )

    lock_data = json.loads(package_lock_path.read_text(encoding="utf-8"))
    lock_data["version"] = version
    lock_data.setdefault("packages", {}).setdefault("", {})["version"] = version
    package_lock_path.write_text(
        json.dumps(lock_data, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )


def project_version_command(values: list[str]) -> None:
    """Print, set, increment, or decrement the project version."""

    current = require_project_versions_match()

    if not values:
        print(current)
        return

    if len(values) == 1:
        next_version = values[0]
    elif len(values) == 2:
        operation, part = values

        if operation not in {"add", "sub"} or part not in {"major", "minor", "patch"}:
            raise RuntimeError("usage: project-version [<version>|<add|sub> <patch|minor|major>]")

        next_version = bump_semver(current, operation, part)
    else:
        raise RuntimeError("usage: project-version [<version>|<add|sub> <patch|minor|major>]")

    set_project_version(next_version)
    require_project_versions_match()
    print(next_version)


def expected_release_version() -> str:
    """Return the release version from GITHUB_REF_NAME when present."""

    tag = os.environ.get("GITHUB_REF_NAME")

    if not tag:
        return require_project_versions_match()

    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        raise RuntimeError(f"release tag must be SemVer vX.Y.Z, got {tag!r}")

    return tag[1:]


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


def build_sdk_binaries() -> None:
    """Build and bundle BRIDGE binaries for all supported platforms."""

    version = expected_release_version()
    source_version = bridge_version_from_source()
    package_json_version = package_version()

    if source_version != version or package_json_version != version:
        raise RuntimeError(
            "version mismatch before binary build: "
            f"expected {version}, buildinfo={source_version}, "
            f"package.json={package_json_version}"
        )

    commit = current_commit()
    build_time = current_build_time()
    dirty = source_dirty_state()

    if SDK_BUILD_DIR.exists():
        shutil.rmtree(SDK_BUILD_DIR)

    SDK_BUILD_DIR.mkdir(parents=True)

    go_version = output(["go", "version"])

    for key, goos, goarch, filename in TARGETS:
        destination = SDK_BUILD_DIR / key / filename
        destination.parent.mkdir(
            parents=True,
            exist_ok=True,
        )

        env = os.environ.copy()
        env.update(
            {
                "CGO_ENABLED": "0",
                "GOOS": goos,
                "GOARCH": goarch,
            }
        )

        run(
            [
                "go",
                "build",
                "-trimpath",
                "-ldflags="
                + bridge_ldflags(
                    version=version,
                    commit=commit,
                    build_time=build_time,
                    dirty=dirty,
                    stripped=True,
                ),
                "-o",
                str(destination),
                CLI_PACKAGE,
            ],
            env=env,
        )

        make_executable(destination)

        native_os = (
            "windows"
            if os.name == "nt"
            else "darwin"
            if sys.platform == "darwin"
            else "linux"
            if sys.platform.startswith("linux")
            else ""
        )
        machine = platform.machine().lower()
        native_arch = (
            "amd64"
            if machine in {"x86_64", "amd64"}
            else "arm64"
            if machine in {"aarch64", "arm64"}
            else ""
        )

        if key == f"{native_os}-{native_arch}":
            bridge_version = normalize_bridge_version(
                output([str(destination), "version", "--output", "json"])
            )

            if bridge_version != version:
                raise RuntimeError(
                    f"{key} binary version mismatch: expected {version}, got {bridge_version}"
                )

    if PYTHON_PACKAGE.parent.exists():
        sync_tree(
            SDK_BUILD_DIR,
            PYTHON_PACKAGE / "bin",
        )

    sync_tree(
        SDK_BUILD_DIR,
        TYPESCRIPT_SDK / "bin",
    )

    manifest: dict[str, dict[str, str]] = {}

    for key, goos, goarch, filename in TARGETS:
        final_binary = TYPESCRIPT_SDK / "bin" / key / filename

        if not final_binary.exists():
            raise RuntimeError(f"SDK binary was not copied: {final_binary}")

        manifest[key] = {
            "platform": key,
            "os": goos,
            "cpu": goarch,
            "path": f"bin/{key}/{filename}",
            "sha256": sha256(final_binary),
            "version": version,
            "commit": commit,
            "built_at": build_time,
            "go_version": go_version,
        }

    manifest_text = json.dumps(
        manifest,
        indent=2,
        sort_keys=True,
    ) + "\n"

    if PYTHON_PACKAGE.parent.exists():
        (
            PYTHON_PACKAGE
            / "binary-manifest.json"
        ).write_text(
            manifest_text,
            encoding="utf-8",
        )

    (
        TYPESCRIPT_SDK
        / "binary-manifest.json"
    ).write_text(
        manifest_text,
        encoding="utf-8",
    )

    print(
        f"updated SDK binaries and manifests for BRIDGE {version}"
    )


def typescript_compiler_path() -> Path:
    """Return the expected local TypeScript compiler path."""

    executable = (
        "tsc.cmd"
        if os.name == "nt"
        else "tsc"
    )

    return (
        TYPESCRIPT_SDK
        / "node_modules"
        / ".bin"
        / executable
    )


def install_typescript_dependencies() -> None:
    """Install TypeScript SDK development dependencies."""

    package_json = TYPESCRIPT_SDK / "package.json"
    package_lock = TYPESCRIPT_SDK / "package-lock.json"

    if not package_json.exists():
        raise FileNotFoundError(
            "TypeScript SDK package file was not found: "
            f"{package_json}"
        )

    if package_lock.exists():
        run(
            [
                "npm",
                "ci",
                "--include=dev",
            ],
            cwd=TYPESCRIPT_SDK,
        )
    else:
        print(
            "package-lock.json was not found; "
            "falling back to npm install.",
            flush=True,
        )

        run(
            [
                "npm",
                "install",
                "--include=dev",
            ],
            cwd=TYPESCRIPT_SDK,
        )

    compiler = typescript_compiler_path()

    if not compiler.exists():
        raise RuntimeError(
            "TypeScript compiler was not installed. "
            "Confirm that 'typescript' is declared in "
            "src/sdk/typescript/package.json."
        )

    print(
        "TypeScript SDK dependencies installed successfully."
    )


def ensure_typescript_dependencies() -> None:
    """Require TypeScript dependencies to be installed."""

    compiler = typescript_compiler_path()

    if not compiler.exists():
        raise RuntimeError(
            "TypeScript SDK dependencies are not installed. "
            "Run 'mise run sdk:typescript:install' first."
        )


def setup() -> None:
    """Prepare Python and TypeScript SDK development dependencies."""

    run(
        [
            sys.executable,
            "-m",
            "pip",
            "install",
            "--upgrade",
            "pip",
        ]
    )

    run(
        [
            sys.executable,
            "-m",
            "pip",
            "install",
            "-e",
            str(PYTHON_SDK),
        ]
    )

    install_typescript_dependencies()


def clean() -> None:
    """Remove generated build and package outputs."""

    generated_directories = (
        BUILD_DIR,
        PYTHON_SDK / "dist",
        PYTHON_SDK / "build",
        TYPESCRIPT_SDK / "dist",
        TYPESCRIPT_SDK / "bin",
    )

    for path in generated_directories:
        if not path.exists():
            continue

        shutil.rmtree(path)

        print(
            f"removed: {path.relative_to(ROOT)}"
        )

    for path in TYPESCRIPT_SDK.glob("*.tgz"):
        path.unlink()

        print(
            f"removed: {path.relative_to(ROOT)}"
        )

    manifest = TYPESCRIPT_SDK / "binary-manifest.json"

    if manifest.exists():
        manifest.unlink()

        print(f"removed: {manifest.relative_to(ROOT)}")


def clean_typescript_dependencies() -> None:
    """Remove locally installed TypeScript dependencies."""

    node_modules = TYPESCRIPT_SDK / "node_modules"

    if not node_modules.exists():
        print(
            "TypeScript node_modules directory does not exist."
        )
        return

    shutil.rmtree(node_modules)

    print(
        f"removed: {node_modules.relative_to(ROOT)}"
    )


def test_cli() -> None:
    """Build and smoke-test the BRIDGE CLI."""

    binary = build_cli(release=True)

    version = normalize_bridge_version(
        output(
            [
                str(binary),
                "version",
                "--output",
                "json",
            ]
        )
    )

    expected = bridge_version_from_source()

    if version != expected:
        raise RuntimeError(
            "CLI version mismatch: "
            f"expected {expected}, got {version}"
        )

    run(
        [
            str(binary),
            "route",
            "tests/examples/route-request.json",
        ]
    )

    run(
        [
            str(binary),
            "benchmark",
            "validate",
            "tests/scenarios/operational.yaml",
        ]
    )


def package_python() -> None:
    """Build the Python SDK source and wheel distributions."""

    try:
        import build  # type: ignore[import-not-found]  # noqa: F401
    except ImportError:
        run(
            [
                sys.executable,
                "-m",
                "pip",
                "install",
                "build",
            ]
        )

    run(
        [
            sys.executable,
            "-m",
            "build",
        ],
        cwd=PYTHON_SDK,
    )


def remove_old_typescript_package_outputs() -> None:
    """Remove previous TypeScript build and package outputs."""

    dist = TYPESCRIPT_SDK / "dist"

    if dist.exists():
        shutil.rmtree(dist)

        print(
            f"removed: {dist.relative_to(ROOT)}"
        )

    for archive in TYPESCRIPT_SDK.glob("*.tgz"):
        archive.unlink()

        print(
            f"removed: {archive.relative_to(ROOT)}"
        )


def build_typescript() -> None:
    """Compile the TypeScript SDK after deleting old dist output."""

    ensure_typescript_dependencies()

    dist = TYPESCRIPT_SDK / "dist"

    if dist.exists():
        shutil.rmtree(dist)

        print(f"removed: {dist.relative_to(ROOT)}")

    run(["npm", "run", "build"], cwd=TYPESCRIPT_SDK)


def test_typescript() -> None:
    """Run TypeScript SDK tests."""

    ensure_typescript_dependencies()

    run(["npm", "test"], cwd=TYPESCRIPT_SDK)


def pack_typescript() -> Path:
    """Create the npm package tarball and return its path."""

    for archive in TYPESCRIPT_SDK.glob("*.tgz"):
        archive.unlink()

        print(f"removed: {archive.relative_to(ROOT)}")

    pack_result_text = output(["npm", "pack", "--json"], cwd=TYPESCRIPT_SDK)

    try:
        pack_result = json.loads(pack_result_text)
        filename = pack_result[0]["filename"]
    except (json.JSONDecodeError, IndexError, KeyError, TypeError) as error:
        raise RuntimeError(
            "could not determine the npm package filename from npm pack output"
        ) from error

    archive = TYPESCRIPT_SDK / filename

    if not archive.exists():
        raise RuntimeError(f"npm pack did not create {archive}")

    print(f"packed: {archive.relative_to(ROOT)}")

    return archive


def latest_typescript_archive() -> Path:
    """Return the single generated TypeScript SDK tarball."""

    archives = sorted(TYPESCRIPT_SDK.glob("*.tgz"))

    if len(archives) != 1:
        raise RuntimeError(
            "expected exactly one generated TypeScript SDK tarball, "
            f"found {len(archives)}"
        )

    return archives[0]


def verify_npm_package(archive: Path | None = None) -> Path:
    """Validate files included in the generated npm package tarball."""

    archive = archive or latest_typescript_archive()

    if not archive.exists():
        raise FileNotFoundError(
            "npm package archive was not generated: "
            f"{archive}"
        )

    if archive.stat().st_size > MAX_NPM_PACKAGE_BYTES:
        raise RuntimeError(
            "npm package is larger than the configured limit: "
            f"{archive.stat().st_size} bytes"
        )

    forbidden_parts = {
        ".git",
        ".github",
        "node_modules",
        "src",
        "test",
        "tests",
        "tasks",
        "__pycache__",
    }

    required_prefixes = {
        "package/dist/",
        "package/bin/",
        "package/examples/",
    }

    required_files = {
        "package/package.json",
        "package/binary-manifest.json",
        "package/README.md",
        "package/LICENSE",
    }

    allowed_roots = {
        "bin",
        "binary-manifest.json",
        "dist",
        "examples",
        "LICENSE",
        "package.json",
        "README.md",
    }

    with tarfile.open(
        archive,
        mode="r:gz",
    ) as package:
        members = package.getmembers()
        names = {member.name.replace("\\", "/") for member in members if member.isfile()}
        member_by_name = {member.name.replace("\\", "/"): member for member in members}

        package_file = package.extractfile("package/package.json")
        manifest_file = package.extractfile("package/binary-manifest.json")

        if package_file is None or manifest_file is None:
            raise RuntimeError("package metadata files could not be read from tarball")

        packed_package_json = json.loads(package_file.read().decode("utf-8"))
        manifest = json.loads(manifest_file.read().decode("utf-8"))

    for name in sorted(names):
        parts = Path(name).parts
        relative_parts = parts[1:]

        if not parts or parts[0] != "package":
            raise RuntimeError(f"unexpected tarball path root: {name}")

        if not relative_parts or relative_parts[0] not in allowed_roots:
            raise RuntimeError(f"file is outside package.json files allowlist: {name}")

        if any(
            part in forbidden_parts
            for part in parts
        ) or name.endswith((".go", ".tgz", ".pem", ".key")):
            raise RuntimeError(
                "forbidden file was included in npm package: "
                f"{name}"
            )

    missing_files = sorted(
        required_files - names
    )

    if missing_files:
        raise RuntimeError(
            "npm package is missing required files: "
            + ", ".join(missing_files)
        )

    for prefix in sorted(required_prefixes):
        if not any(
            name.startswith(prefix)
            for name in names
        ):
            raise RuntimeError(
                "npm package does not contain required path: "
                f"{prefix}"
            )

    if packed_package_json.get("name") != NPM_PACKAGE_NAME:
        raise RuntimeError(f"unexpected package name: {packed_package_json.get('name')!r}")

    if packed_package_json.get("version") != expected_release_version():
        raise RuntimeError(
            "package version does not match release version: "
            f"{packed_package_json.get('version')!r}"
        )

    if (
        packed_package_json.get("publishConfig", {}).get("registry")
        != GITHUB_PACKAGES_REGISTRY
    ):
        raise RuntimeError("package publishConfig.registry must point to GitHub Packages")

    if "registry.npmjs.org" in json.dumps(packed_package_json, sort_keys=True):
        raise RuntimeError("package metadata must not publish to npmjs.com")

    expected_targets = {key for key, _, _, _ in TARGETS}

    if set(manifest) != expected_targets:
        raise RuntimeError(
            "binary manifest targets mismatch: "
            f"expected {sorted(expected_targets)}, got {sorted(manifest)}"
        )

    with tarfile.open(archive, mode="r:gz") as package:
        for key, goos, goarch, filename in TARGETS:
            entry = manifest[key]
            path = f"package/{entry.get('path')}"
            expected_path = f"package/bin/{key}/{filename}"

            if (
                entry.get("platform") != key
                or entry.get("os") != goos
                or entry.get("cpu") != goarch
            ):
                raise RuntimeError(f"manifest metadata mismatch for {key}")

            if entry.get("version") != expected_release_version():
                raise RuntimeError(f"manifest version mismatch for {key}")

            if path != expected_path or path not in names:
                raise RuntimeError(f"missing binary for {key}: {expected_path}")

            member = member_by_name[path]
            binary_file = package.extractfile(member)

            if binary_file is None:
                raise RuntimeError(f"could not read binary from tarball: {path}")

            binary_bytes = binary_file.read()

            if not binary_bytes:
                raise RuntimeError(f"binary is empty: {path}")

            if hashlib.sha256(binary_bytes).hexdigest() != entry.get("sha256"):
                raise RuntimeError(f"binary sha256 mismatch for {key}")

            if os.name != "nt" and goos in {"linux", "darwin"} and member.mode & 0o111 == 0:
                raise RuntimeError(f"binary is not executable in tarball: {path}")

            if goos == "windows" and not path.endswith(".exe"):
                raise RuntimeError(f"Windows binary must use .exe: {path}")

    print(
        "verified npm package contents: "
        f"{archive.relative_to(ROOT)} "
        f"({len(names)} files)"
    )

    return archive


def package_typescript() -> None:
    """Build and package the TypeScript SDK."""

    remove_old_typescript_package_outputs()
    build_typescript()
    archive = pack_typescript()
    verify_npm_package(archive)


def release_typescript_package() -> None:
    """Build, test, pack, and verify the TypeScript SDK release candidate."""

    clean()
    build_sdk_binaries()
    install_typescript_dependencies()
    build_typescript()
    test_typescript()
    archive = pack_typescript()
    verify_npm_package(archive)


def publish_typescript_package() -> None:
    """Publish the verified TypeScript SDK tarball to GitHub Packages."""

    archive = verify_npm_package()

    if not os.environ.get("NODE_AUTH_TOKEN"):
        raise RuntimeError("NODE_AUTH_TOKEN is required for GitHub Packages publishing")

    view_command = resolve_command(
        [
            "npm",
            "view",
            f"{NPM_PACKAGE_NAME}@{expected_release_version()}",
            "version",
            "--registry",
            GITHUB_PACKAGES_REGISTRY,
        ]
    )
    view_result = subprocess.run(
        view_command,
        cwd=TYPESCRIPT_SDK,
        text=True,
        shell=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    if view_result.returncode == 0:
        raise RuntimeError(
            f"{NPM_PACKAGE_NAME}@{expected_release_version()} is already published"
        )

    run(["npm", "publish", str(archive)], cwd=TYPESCRIPT_SDK)


def ensure_clean_tracked_worktree() -> None:
    """Require no tracked or untracked source changes before tagging."""

    status = git_output(["status", "--porcelain", "--untracked-files=normal"])

    if status:
        raise RuntimeError(
            "working tree has uncommitted changes; commit before running publish"
        )


def ensure_current_branch_is_main() -> None:
    """Require the local publish trigger to run from main."""

    branch = git_output(["branch", "--show-current"])

    if branch != "main":
        raise RuntimeError(f"publish must be run from main, current branch is {branch!r}")


def ensure_release_tag_is_available(tag: str) -> None:
    """Fail if the release tag already exists locally or remotely."""

    local = subprocess.run(
        resolve_command(["git", "rev-parse", "--verify", "--quiet", f"refs/tags/{tag}"]),
        cwd=ROOT,
        shell=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    if local.returncode == 0:
        raise RuntimeError(f"tag already exists locally: {tag}")

    remote = subprocess.run(
        resolve_command(["git", "ls-remote", "--exit-code", "--tags", "origin", tag]),
        cwd=ROOT,
        shell=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    if remote.returncode == 0:
        raise RuntimeError(f"tag already exists on origin: {tag}")


def trigger_release_publish() -> None:
    """Verify the release candidate, create the version tag, and push it."""

    version = require_project_versions_match()
    tag = f"v{version}"

    ensure_current_branch_is_main()
    ensure_clean_tracked_worktree()
    ensure_release_tag_is_available(tag)
    release_typescript_package()
    run(["git", "tag", "-a", tag, "-m", f"BRIDGE {tag}"])
    run(["git", "push", "origin", tag])

    print(f"pushed release tag: {tag}")


def verify(
    *,
    skip_sdk_binaries: bool,
) -> None:
    """Run the full repository verification suite."""

    if not skip_sdk_binaries:
        build_sdk_binaries()

    run(
        [
            "go",
            "test",
            "-count=1",
            "./...",
        ]
    )

    run(
        [
            "go",
            "test",
            "-race",
            "./...",
        ]
    )

    run(
        [
            "go",
            "vet",
            "./...",
        ]
    )

    test_cli()

    run(
        [
            sys.executable,
            "-m",
            "unittest",
            "discover",
            "-s",
            "src/sdk/python/tests",
            "-v",
        ]
    )

    test_typescript()

    run(
        [
            sys.executable,
            "tests/contracts/verify_glossary.py",
        ]
    )


def benchmark_run(
    scenario: str,
) -> None:
    """Run a BRIDGE benchmark scenario through the CLI."""

    command = [
        "go",
        "run",
        CLI_PACKAGE,
        "benchmark",
        scenario,
    ]

    run(command)


def build_argument_parser() -> argparse.ArgumentParser:
    """Build the command-line parser."""

    parser = argparse.ArgumentParser(
        description=(
            "Cross-platform BRIDGE development tasks"
        )
    )

    subparsers = parser.add_subparsers(
        dest="command",
        required=True,
    )

    subparsers.add_parser(
        "setup",
        help="Install Python and TypeScript development dependencies",
    )

    subparsers.add_parser(
        "clean",
        help="Remove generated build and package outputs",
    )

    subparsers.add_parser(
        "clean-typescript-dependencies",
        help="Remove TypeScript node_modules",
    )

    project_version_parser = subparsers.add_parser(
        "project-version",
        help="Print or update synchronized project version files",
    )

    project_version_parser.add_argument(
        "values",
        nargs="*",
        help="<version> or <add|sub> <patch|minor|major>",
    )

    subparsers.add_parser(
        "publish",
        help="Verify, tag, and push the current project version",
    )

    build_parser = subparsers.add_parser(
        "build",
        help="Build the BRIDGE CLI for the current platform",
    )

    build_group = (
        build_parser.add_mutually_exclusive_group()
    )

    build_group.add_argument(
        "--debug",
        action="store_true",
    )

    build_group.add_argument(
        "--release",
        action="store_true",
    )

    subparsers.add_parser(
        "sdk-binaries",
        help="Build and bundle all SDK binaries",
    )

    subparsers.add_parser(
        "install-typescript",
        help="Install TypeScript SDK dependencies",
    )

    subparsers.add_parser(
        "build-typescript",
        help="Build the TypeScript SDK",
    )

    subparsers.add_parser(
        "test-typescript",
        help="Run TypeScript SDK tests",
    )

    subparsers.add_parser(
        "pack-typescript",
        help="Create the TypeScript SDK npm tarball",
    )

    subparsers.add_parser(
        "verify-typescript-package",
        help="Verify the generated TypeScript SDK npm tarball",
    )

    subparsers.add_parser(
        "release-typescript-package",
        help="Build and verify the TypeScript SDK release candidate",
    )

    subparsers.add_parser(
        "publish-typescript-package",
        help="Publish the verified TypeScript SDK tarball",
    )

    subparsers.add_parser(
        "test-cli",
        help="Build and smoke-test the BRIDGE CLI",
    )

    subparsers.add_parser(
        "package-python",
        help="Build the Python SDK package",
    )

    subparsers.add_parser(
        "package-typescript",
        help="Build the TypeScript SDK package",
    )

    verify_parser = subparsers.add_parser(
        "verify",
        help="Run the complete repository verification suite",
    )

    verify_parser.add_argument(
        "--skip-sdk-binaries",
        action="store_true",
    )

    benchmark_parser = subparsers.add_parser(
        "benchmark-run",
        help="Run a benchmark scenario through the BRIDGE CLI",
    )

    benchmark_parser.add_argument(
        "scenario",
        help="Path to the benchmark scenario YAML or JSON file",
    )

    return parser


def main() -> int:
    """Execute the selected development task."""

    parser = build_argument_parser()
    args = parser.parse_args()

    if args.command == "setup":
        setup()

    elif args.command == "clean":
        clean()

    elif args.command == "clean-typescript-dependencies":
        clean_typescript_dependencies()

    elif args.command == "project-version":
        project_version_command(args.values)

    elif args.command == "publish":
        trigger_release_publish()

    elif args.command == "build":
        build_cli(
            debug=args.debug,
            release=args.release,
        )

    elif args.command == "sdk-binaries":
        build_sdk_binaries()

    elif args.command == "install-typescript":
        install_typescript_dependencies()

    elif args.command == "build-typescript":
        build_typescript()

    elif args.command == "test-typescript":
        test_typescript()

    elif args.command == "pack-typescript":
        pack_typescript()

    elif args.command == "verify-typescript-package":
        verify_npm_package()

    elif args.command == "release-typescript-package":
        release_typescript_package()

    elif args.command == "publish-typescript-package":
        publish_typescript_package()

    elif args.command == "test-cli":
        test_cli()

    elif args.command == "package-python":
        package_python()

    elif args.command == "package-typescript":
        package_typescript()

    elif args.command == "verify":
        verify(
            skip_sdk_binaries=args.skip_sdk_binaries,
        )

    elif args.command == "benchmark-run":
        benchmark_run(args.scenario)

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("\nInterrupted.", file=sys.stderr, flush=True)
        raise SystemExit(130)
    except subprocess.CalledProcessError as error:
        raise SystemExit(error.returncode)
