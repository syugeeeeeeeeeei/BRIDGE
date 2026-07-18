#!/usr/bin/env python3
from __future__ import annotations
import json, os, shutil, subprocess, sys, tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

def run(*args: str, cwd: Path = ROOT, input_text: str | None = None) -> subprocess.CompletedProcess[str]:
    cp = subprocess.run(args, cwd=cwd, input=input_text, text=True, capture_output=True)
    if cp.returncode:
        print(cp.stdout)
        print(cp.stderr, file=sys.stderr)
        raise SystemExit(f"command failed ({cp.returncode}): {' '.join(args)}")
    return cp

def main() -> int:
    work = Path(tempfile.mkdtemp(prefix="bridge-v0153-audit-"))
    try:
        binary = work / ("bridge.exe" if os.name == "nt" else "bridge")
        run("go", "build", "-o", str(binary), "./src/products/cli/cmd/bridge")
        route = (ROOT / "tests/examples/route-request.json").read_text()
        response = json.loads(run(str(binary), "route", input_text=route).stdout)
        assert response["schema_version"] == "bridge.route.result.v1"
        source_scenario = ROOT / "tests/examples/benchmark-smoke-v1.json"
        scenario_data = json.loads(source_scenario.read_text())
        scenario_data["output"]["directory"] = str(work / "artifacts")
        scenario = work / "benchmark-smoke-v1.json"
        scenario.write_text(json.dumps(scenario_data, ensure_ascii=False, indent=2))
        run(str(binary), "scenario", "validate", str(scenario))
        bench = run(str(binary), "benchmark", "run", str(scenario))
        candidates = sorted(work.rglob("*.zip"))
        if not candidates:
            raise SystemExit(f"benchmark did not create zip artifact\n{bench.stdout}\n{bench.stderr}")
        artifact = candidates[-1]
        run(str(binary), "artifact", "validate", str(artifact))
        run(str(binary), "artifact", "evaluate", str(artifact))
        run(sys.executable, "-m", "unittest", "discover", "-s", "tests", "-p", "test_*.py", cwd=ROOT / "src/sdk/python")
        run("npm", "test", cwd=ROOT / "src/sdk/typescript")
        print(json.dumps({"status":"pass","artifact":str(artifact),"checks":["route","scenario","benchmark","artifact_validate","artifact_evaluate","python_sdk","typescript_sdk"]}, ensure_ascii=False, indent=2))
        return 0
    finally:
        shutil.rmtree(work, ignore_errors=True)

if __name__ == "__main__":
    raise SystemExit(main())
