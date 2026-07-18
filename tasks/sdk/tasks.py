from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import shutil
import subprocess
import sys
import tarfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import (
    CLI_PACKAGE,
    GITHUB_PACKAGES_REGISTRY,
    MAX_NPM_PACKAGE_BYTES,
    NPM_PACKAGE_NAME,
    PYTHON_PACKAGE,
    PYTHON_SDK,
    ROOT,
    SDK_BUILD_DIR,
    TARGETS,
    TYPESCRIPT_SDK,
    bridge_ldflags,
    current_build_time,
    current_commit,
    git_output,
    make_executable,
    normalize_bridge_version,
    output,
    resolve_command,
    run,
    sha256,
    source_dirty_state,
    sync_tree,
)
from project.tasks import (
    bridge_version_from_source,
    expected_release_version,
    package_version,
)
from setup.tasks import clean, ensure_typescript_dependencies, install_typescript_dependencies


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
            try:
                bridge_version = normalize_bridge_version(
                    output([str(destination), "version", "--output", "json"])
                )
            except OSError as error:
                if os.name == "nt" and getattr(error, "winerror", None) == 4551:
                    print(
                        f"skipped native binary execution check because Windows blocked {destination}"
                    )
                    bridge_version = version
                else:
                    raise

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


def main() -> int:
    parser = argparse.ArgumentParser(description="BRIDGE SDK tasks")
    subparsers = parser.add_subparsers(dest="command", required=True)

    for command, help_text in {
        "sdk-binaries": "Build and bundle all SDK binaries",
        "install-typescript": "Install TypeScript SDK dependencies",
        "build-typescript": "Build the TypeScript SDK",
        "test-typescript": "Run TypeScript SDK tests",
        "pack-typescript": "Create the TypeScript SDK npm tarball",
        "verify-typescript-package": "Verify the generated TypeScript SDK npm tarball",
        "release-typescript-package": "Build and verify the TypeScript SDK release candidate",
        "publish-typescript-package": "Publish the verified TypeScript SDK tarball",
        "package-python": "Build the Python SDK package",
        "package-typescript": "Build the TypeScript SDK package",
    }.items():
        subparsers.add_parser(command, help=help_text)

    args = parser.parse_args()

    if args.command == "sdk-binaries":
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
    elif args.command == "package-python":
        package_python()
    elif args.command == "package-typescript":
        package_typescript()

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("\nInterrupted.", file=sys.stderr, flush=True)
        raise SystemExit(130)
    except subprocess.CalledProcessError as error:
        raise SystemExit(error.returncode)

