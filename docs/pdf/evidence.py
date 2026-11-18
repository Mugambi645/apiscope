"""Runs the real test suites at build time so the guide reports measured results."""
import collections
import json
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def go_results():
    out = subprocess.run(["go", "test", "-race", "-count=1", "-json", "./..."], cwd=ROOT,
                         capture_output=True, text=True)
    per = collections.OrderedDict()
    for line in out.stdout.splitlines():
        try:
            e = json.loads(line)
        except ValueError:
            continue
        pkg = e.get("Package", "").replace("apiscope/", "")
        if e.get("Test") and e["Action"] in ("pass", "fail"):
            per.setdefault(pkg, collections.Counter())[e["Action"]] += 1
    return per


def web_results():
    out = subprocess.run(["npm", "test"], cwd=ROOT / "web", capture_output=True, text=True)
    nums = {k: int(v) for k, v in re.findall(r"# (tests|pass|fail) (\d+)", out.stdout)}
    return nums


def loc():
    def count(globs, exclude_tests=False):
        total = 0
        for g in globs:
            for p in ROOT.glob(g):
                if "node_modules" in p.parts or ".next" in p.parts:
                    continue
                if exclude_tests == (p.name.endswith("_test.go") or p.name.endswith(".test.ts")):
                    pass
                total += len(p.read_text().splitlines())
        return total
    go_src = sum(len(p.read_text().splitlines()) for p in ROOT.rglob("*.go") if not p.name.endswith("_test.go"))
    go_test = sum(len(p.read_text().splitlines()) for p in ROOT.rglob("*_test.go"))
    web = sum(len(p.read_text().splitlines()) for p in (ROOT / "web").rglob("*")
              if p.suffix in (".ts", ".tsx", ".css") and "node_modules" not in p.parts and ".next" not in p.parts)
    return go_src, go_test, web
