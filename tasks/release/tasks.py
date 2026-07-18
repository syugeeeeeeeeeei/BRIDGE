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


def release_tag_presence(tag: str) -> tuple[bool, bool]:
    """Return whether the release tag exists locally and remotely."""

    local = subprocess.run(
        resolve_command(["git", "rev-parse", "--verify", "--quiet", f"refs/tags/{tag}"]),
        cwd=ROOT,
        shell=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    remote = subprocess.run(
        resolve_command(["git", "ls-remote", "--exit-code", "--tags", "origin", tag]),
        cwd=ROOT,
        shell=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    return local.returncode == 0, remote.returncode == 0


def confirm_recreate_release_tag(tag: str, *, local: bool, remote: bool) -> None:
    """Warn and require explicit confirmation before replacing a release tag."""

    locations = []

    if local:
        locations.append("local")

    if remote:
        locations.append("origin")

    print("")
    print("WARNING: release tag already exists")
    print(f"tag: {tag}")
    print(f"locations: {', '.join(locations)}")
    print("")
    print("This will delete the existing tag, recreate it at the current main commit,")
    print("and push it to origin. If GitHub Packages already contains this package")
    print("version, the publish workflow will fail because npm versions cannot be reused.")
    print("")

    if not sys.stdin.isatty():
        raise RuntimeError(
            f"tag already exists: {tag}; rerun interactively to confirm recreation"
        )

    expected = f"recreate {tag}"
    answer = input(f'Type "{expected}" to delete and recreate the tag: ').strip()

    if answer != expected:
        raise RuntimeError("publish cancelled; release tag was left unchanged")


def delete_release_tag(tag: str, *, local: bool, remote: bool) -> None:
    """Delete an existing release tag locally and/or remotely."""

    if remote:
        run(["git", "push", "origin", f":refs/tags/{tag}"])

    if local:
        run(["git", "tag", "-d", tag])


def create_and_push_release_tag(tag: str) -> None:
    """Create and push the release tag."""

    run(["git", "tag", "-a", tag, "-m", f"BRIDGE {tag}"])
    run(["git", "push", "origin", tag])


def trigger_release_publish() -> None:
    """Verify the release candidate, create the version tag, and push it."""

    version = require_project_versions_match()
    tag = f"v{version}"

    ensure_current_branch_is_main()
    ensure_clean_tracked_worktree()

    local_tag_exists, remote_tag_exists = release_tag_presence(tag)

    if local_tag_exists or remote_tag_exists:
        confirm_recreate_release_tag(
            tag,
            local=local_tag_exists,
            remote=remote_tag_exists,
        )

    release_typescript_package()

    if local_tag_exists or remote_tag_exists:
        delete_release_tag(
            tag,
            local=local_tag_exists,
            remote=remote_tag_exists,
        )

    create_and_push_release_tag(tag)

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
