from __future__ import annotations

import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import (
    BUILD_DIR,
    CLI_PACKAGE,
    ROOT,
    bridge_ldflags,
    current_build_time,
    current_commit,
    executable_name,
    run,
    source_dirty_state,
)
from project.tasks import bridge_version_from_source


def build_cli(*, debug: bool = False, release: bool = False) -> Path:
    """Build the BRIDGE CLI for the current platform."""

    BUILD_DIR.mkdir(parents=True, exist_ok=True)

    destination = BUILD_DIR / executable_name()
    version = bridge_version_from_source()
    commit = current_commit()
    build_time = current_build_time()
    dirty = source_dirty_state()

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
                stripped=release or not debug,
            ),
            "-o",
            str(destination),
            CLI_PACKAGE,
        ]
    )

    print(f"built: {destination.relative_to(ROOT)}")

    return destination


def main() -> int:
    parser = argparse.ArgumentParser(description="BRIDGE build tasks")
    parser.add_argument("--debug", action="store_true")
    parser.add_argument("--release", action="store_true")
    args = parser.parse_args()

    if args.debug and args.release:
        parser.error("--debug and --release are mutually exclusive")

    build_cli(debug=args.debug, release=args.release)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
