from __future__ import annotations

import argparse
import os
import shutil
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import BUILD_DIR, PYTHON_SDK, ROOT, TYPESCRIPT_SDK, run


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


def main() -> int:
    parser = argparse.ArgumentParser(description="BRIDGE setup and cleanup tasks")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("setup", help="Install project development dependencies")
    subparsers.add_parser("clean", help="Remove generated build and package outputs")
    subparsers.add_parser(
        "clean-typescript-dependencies",
        help="Remove TypeScript node_modules",
    )
    subparsers.add_parser(
        "install-typescript",
        help="Install TypeScript SDK dependencies",
    )

    args = parser.parse_args()

    if args.command == "setup":
        setup()
    elif args.command == "clean":
        clean()
    elif args.command == "clean-typescript-dependencies":
        clean_typescript_dependencies()
    elif args.command == "install-typescript":
        install_typescript_dependencies()

    return 0


if __name__ == "__main__":
    raise SystemExit(main())


