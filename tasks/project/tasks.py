from __future__ import annotations

import argparse
import json
import os
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import ROOT, SEMVER_PATTERN, TYPESCRIPT_SDK


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

    placeholders = {
        "${usage_value}",
        "${usage_part}",
        "$env:usage_value",
        "$env:usage_part",
        "%usage_value%",
        "%usage_part%",
    }
    values = [value for value in values if value and value not in placeholders]

    if not values:
        values = [
            value
            for value in (
                os.environ.get("usage_value", ""),
                os.environ.get("usage_part", ""),
            )
            if value
        ]

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


def build_argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="BRIDGE project version tasks")
    subparsers = parser.add_subparsers(dest="command")

    project_version = subparsers.add_parser(
        "project-version",
        help="Print or update synchronized project version files",
    )
    project_version.add_argument(
        "values",
        nargs="*",
        help="<version> or <add|sub> <patch|minor|major>",
    )

    return parser


def main() -> int:
    parser = build_argument_parser()
    args = parser.parse_args()

    if args.command in {None, "project-version"}:
        project_version_command(args.values if args.command else [])

    return 0


if __name__ == "__main__":
    raise SystemExit(main())


