from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GLOSSARY = ROOT / "docs" / "WORD_DEFINITION.md"
ARCHITECTURE = ROOT / "docs" / "ARCHITECTURE_RULE.md"

REQUIRED_TERMS = (
    "Work",
    "Step",
    "Scenario",
    "Run",
    "Raw Run",
    "Query ID",
    "Graph Instance ID",
    "Execution Target",
    "Execution Path",
    "Observation Mode",
    "Time Breakdown",
    "System Metrics",
    "Stable Digest",
    "Effective Configuration Digest",
)


def main() -> None:
    if not GLOSSARY.is_file():
        raise SystemExit("canonical glossary is missing: docs/WORD_DEFINITION.md")
    if not ARCHITECTURE.is_file():
        raise SystemExit("canonical architecture rule is missing: docs/ARCHITECTURE_RULE.md")

    text = GLOSSARY.read_text(encoding="utf-8")
    missing = [term for term in REQUIRED_TERMS if f"### {term}" not in text]
    if missing:
        raise SystemExit("undefined required BRIDGE terms: " + ", ".join(missing))

    architecture = ARCHITECTURE.read_text(encoding="utf-8")
    if "用語集更新義務" not in architecture:
        raise SystemExit("architecture rule does not require glossary maintenance")

    print(f"Glossary governance: {len(REQUIRED_TERMS)} required terms defined")


if __name__ == "__main__":
    main()
