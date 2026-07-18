from __future__ import annotations

import subprocess
import sys
from pathlib import Path


TASK_ROUTES = {
    "build": ("build", []),
    "setup": ("setup", ["setup"]),
    "clean": ("setup", ["clean"]),
    "clean-typescript-dependencies": ("setup", ["clean-typescript-dependencies"]),
    "project-version": ("project", ["project-version"]),
    "publish": ("release", ["publish"]),
    "sdk-binaries": ("sdk", ["sdk-binaries"]),
    "install-typescript": ("sdk", ["install-typescript"]),
    "build-typescript": ("sdk", ["build-typescript"]),
    "test-typescript": ("sdk", ["test-typescript"]),
    "pack-typescript": ("sdk", ["pack-typescript"]),
    "verify-typescript-package": ("sdk", ["verify-typescript-package"]),
    "release-typescript-package": ("sdk", ["release-typescript-package"]),
    "publish-typescript-package": ("sdk", ["publish-typescript-package"]),
    "package-python": ("sdk", ["package-python"]),
    "package-typescript": ("sdk", ["package-typescript"]),
    "test-cli": ("test", ["test-cli"]),
    "verify": ("test", ["verify"]),
    "benchmark-run": ("test", ["benchmark-run"]),
}


def main(argv: list[str]) -> int:
    if not argv or argv[0] not in TASK_ROUTES:
        commands = ", ".join(sorted(TASK_ROUTES))
        print(f"usage: python tasks/dev_tasks.py <command> [args...]\ncommands: {commands}", file=sys.stderr)
        return 2

    command, rest = argv[0], argv[1:]
    domain, prefix = TASK_ROUTES[command]
    script = Path(__file__).resolve().parent / domain / "tasks.py"
    return subprocess.call([sys.executable, str(script), *prefix, *rest])


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
