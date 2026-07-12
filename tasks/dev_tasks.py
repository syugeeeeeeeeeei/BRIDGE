from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
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
    ("windows-amd64", "windows", "amd64", "bridge.exe"),
    ("darwin-amd64", "darwin", "amd64", "bridge"),
    ("darwin-arm64", "darwin", "arm64", "bridge"),
)


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

    args = [
        "go",
        "build",
        "-trimpath",
    ]

    if release or not debug:
        args.append("-ldflags=-s -w")

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
    """Read the BRIDGE version constant from the CLI source."""

    main_go = (
        ROOT
        / "src"
        / "products"
        / "cli"
        / "cmd"
        / "bridge"
        / "main.go"
    )

    marker = 'const version = "'

    for line in main_go.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()

        if stripped.startswith(marker) and stripped.endswith('"'):
            return stripped[len(marker) : -1]

    raise RuntimeError(
        f"could not find CLI version in {main_go}"
    )


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

    version = bridge_version_from_source()

    if SDK_BUILD_DIR.exists():
        shutil.rmtree(SDK_BUILD_DIR)

    SDK_BUILD_DIR.mkdir(parents=True)

    manifest: dict[str, dict[str, str]] = {}

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
                "-ldflags=-s -w",
                "-o",
                str(destination),
                CLI_PACKAGE,
            ],
            env=env,
        )

        make_executable(destination)

        manifest[key] = {
            "file": f"bin/{key}/{filename}",
            "sha256": sha256(destination),
            "bridge_version": version,
        }

    sync_tree(
        SDK_BUILD_DIR,
        PYTHON_PACKAGE / "bin",
    )

    sync_tree(
        SDK_BUILD_DIR,
        TYPESCRIPT_SDK / "bin",
    )

    manifest_text = json.dumps(
        manifest,
        indent=2,
        sort_keys=True,
    ) + "\n"

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

    version = output(
        [
            str(binary),
            "version",
        ]
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
            "--request",
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


def inspect_npm_package(archive: Path) -> None:
    """Validate files included in the generated npm package."""

    if not archive.exists():
        raise FileNotFoundError(
            "npm package archive was not generated: "
            f"{archive}"
        )

    forbidden_parts = {
        "node_modules",
        "src",
        "test",
        "tests",
        "__pycache__",
    }

    required_prefixes = {
        "package/dist/",
        "package/bin/",
    }

    required_files = {
        "package/package.json",
        "package/binary-manifest.json",
        "package/README.md",
    }

    with tarfile.open(
        archive,
        mode="r:gz",
    ) as package:
        names = {
            member.name.replace("\\", "/")
            for member in package.getmembers()
            if member.isfile()
        }

    for name in sorted(names):
        parts = Path(name).parts

        if any(
            part in forbidden_parts
            for part in parts
        ):
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

    print(
        "verified npm package contents: "
        f"{archive.relative_to(ROOT)} "
        f"({len(names)} files)"
    )


def package_typescript() -> None:
    """Build and package the TypeScript SDK."""

    ensure_typescript_dependencies()
    remove_old_typescript_package_outputs()

    run(
        [
            "npm",
            "run",
            "build",
        ],
        cwd=TYPESCRIPT_SDK,
    )

    pack_result_text = output(
        [
            "npm",
            "pack",
            "--json",
        ],
        cwd=TYPESCRIPT_SDK,
    )

    try:
        pack_result = json.loads(
            pack_result_text
        )

        filename = pack_result[0]["filename"]
    except (
        json.JSONDecodeError,
        IndexError,
        KeyError,
        TypeError,
    ) as error:
        raise RuntimeError(
            "could not determine the npm package filename "
            "from npm pack output"
        ) from error

    archive = TYPESCRIPT_SDK / filename

    inspect_npm_package(archive)


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

    ensure_typescript_dependencies()

    run(
        [
            "npm",
            "test",
        ],
        cwd=TYPESCRIPT_SDK,
    )

    run(
        [
            sys.executable,
            "tests/compatibility/verify.py",
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

    elif args.command == "build":
        build_cli(
            debug=args.debug,
            release=args.release,
        )

    elif args.command == "sdk-binaries":
        build_sdk_binaries()

    elif args.command == "install-typescript":
        install_typescript_dependencies()

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
