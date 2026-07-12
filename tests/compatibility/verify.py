from __future__ import annotations
import json, pathlib, subprocess, sys, tempfile
ROOT=pathlib.Path(__file__).resolve().parents[2]
def run(cmd): return subprocess.check_output(cmd,cwd=ROOT,text=True)
run([sys.executable,"tests/compatibility/verify_glossary.py"])
py=json.loads(run([sys.executable,"tests/compatibility/python_reference_runner.py"]))
go=json.loads(run(["go","run","./others/legacy/cmd-v0.10/bridge-compat"]))
if py!=go:
    for i,(a,b) in enumerate(zip(py,go)):
        if a!=b: raise SystemExit(f"case {i} differs\npython={a}\ngo={b}")
    raise SystemExit(f"result length differs: python={len(py)} go={len(go)}")
print(f"Python-Go semantic parity: {len(py)} results matched")
