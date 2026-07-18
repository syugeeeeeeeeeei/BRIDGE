from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import ROOT, git_output, resolve_command, run
from project.tasks import require_project_versions_match
from sdk.tasks import release_typescript_package


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


def main() -> int:
    parser = argparse.ArgumentParser(description="BRIDGE release tasks")
    parser.add_argument(
        "command",
        nargs="?",
        default="publish",
        choices=["publish"],
        help="Release command to run",
    )
    parser.parse_args()
    trigger_release_publish()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
