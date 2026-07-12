from pathlib import Path

ALLOWED_ROOT_FILES = {"docs/ARCHITECTURE_RULE.md", "README.md", "pyproject.toml"}
ALLOWED_ROOT_DIRS = {"bridge_py", "tests", "docs", "legacy"}


def test_repository_root_is_minimal() -> None:
    root = Path(__file__).resolve().parents[1]
    unexpected = []
    for entry in root.iterdir():
        if entry.name.startswith("."):
            continue
        if entry.is_file() and entry.name not in ALLOWED_ROOT_FILES:
            unexpected.append(entry.name)
        elif entry.is_dir() and entry.name not in ALLOWED_ROOT_DIRS:
            unexpected.append(entry.name + "/")
    assert unexpected == []
