from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from build.tasks import build_cli
from common import CLI_PACKAGE, ROOT, output, run
from project.tasks import bridge_version_from_source
from common import normalize_bridge_version
from sdk.tasks import build_sdk_binaries, test_typescript


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

    if scenario in {"${usage_scenario}", "$env:usage_scenario", "%usage_scenario%"}:
        scenario = os.environ.get("usage_scenario", "")

    if not scenario:
        raise RuntimeError("scenario is required")

    command = [
        "go",
        "run",
        CLI_PACKAGE,
        "benchmark",
        scenario,
    ]

    run(command)


def main() -> int:
    parser = argparse.ArgumentParser(description="BRIDGE test and verification tasks")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("test-cli", help="Build and smoke-test the BRIDGE CLI")
    verify_parser = subparsers.add_parser("verify", help="Run the complete verification suite")
    verify_parser.add_argument("--skip-sdk-binaries", action="store_true")
    benchmark_parser = subparsers.add_parser(
        "benchmark-run",
        help="Run a benchmark scenario through the BRIDGE CLI",
    )
    benchmark_parser.add_argument("scenario")

    args = parser.parse_args()

    if args.command == "test-cli":
        test_cli()
    elif args.command == "verify":
        verify(skip_sdk_binaries=args.skip_sdk_binaries)
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


