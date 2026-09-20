#!/usr/bin/env python3
"""Run from a trusted checkout. Candidate is data, never the source of grading tools."""
import argparse, hashlib, json, os, pathlib, shutil, subprocess, tempfile
TASKS = [f"task_{i:02}" for i in range(11)]

def tree(root):
    result = {}
    for p in root.rglob("*"):
        rel = p.relative_to(root)
        if rel.parts[0] == ".git": continue
        if p.is_symlink(): raise ValueError(f"symlink forbidden: {rel}")
        if p.is_file(): result[str(rel)] = hashlib.sha256(p.read_bytes()).hexdigest()
    return result

def policy(baseline, candidate):
    config = json.loads((baseline / ".etc/config.json").read_text())
    allowed = {f"tasks/{t}/solution.go" for t in TASKS}
    if set(config["diff"]["allow_list"]) != allowed: raise ValueError("invalid instructor allow-list")
    a, b = tree(baseline), tree(candidate)
    changed = sorted(p for p in a.keys() | b.keys() if a.get(p) != b.get(p))
    forbidden = [p for p in changed if p not in allowed or p not in b]
    if forbidden: raise ValueError("forbidden changes: " + repr(forbidden))
    return config, changed

def statuses(events, returncode, package):
    # Missing package terminal event, skipped tests, build failure, malformed output fail closed.
    final = None; passed = set(); failed = False
    for line in events.splitlines():
        try: e = json.loads(line)
        except ValueError: continue
        if e.get("Package") != package: continue
        if e.get("Test"):
            if e.get("Action") == "pass": passed.add(e["Test"])
            if e.get("Action") in ("fail", "skip"): failed = True
        elif e.get("Action") in ("pass", "fail", "skip"): final = e["Action"]
    return "pass" if returncode == 0 and final == "pass" and passed and not failed else "fail"

def main():
    p=argparse.ArgumentParser();p.add_argument("--candidate",required=True);p.add_argument("--out",required=True);p.add_argument("--local",action="store_true",help="Instructor solutions only, no sandbox");a=p.parse_args()
    baseline=pathlib.Path(__file__).resolve().parents[1];candidate=pathlib.Path(a.candidate).resolve();out=pathlib.Path(a.out).resolve();out.mkdir(parents=True,exist_ok=True)
    cfg=json.loads((baseline/".etc/config.json").read_text()); report={f"industry_backend_go/tasks/{t}":{"status":"missing"} for t in TASKS}; guard={"checkCode":"1","report":{}}
    try:
        cfg,changed=policy(baseline,candidate)
        files=[str(candidate/f"tasks/{t}/solution.go") for t in TASKS]
        subprocess.run(["go","run",str(baseline/"tools/importcheck.go")],input=json.dumps(files),text=True,check=True,cwd=baseline)
        guard={"checkCode":"0","report":{"changed_paths":changed}}
        with tempfile.TemporaryDirectory(prefix="autumn-grade-") as tmp:
            work=pathlib.Path(tmp);work.chmod(0o755)
            shutil.copy2(baseline/"go.mod",work/"go.mod")
            shutil.copytree(baseline/"tasks",work/"tasks")
            for t in TASKS: shutil.copy2(candidate/f"tasks/{t}/solution.go",work/f"tasks/{t}/solution.go")
            for t in TASKS:
                cmd=["go","test","-race","-count=4","-timeout=60s","-json",f"./tasks/{t}"]
                if not a.local:
                    cmd=["docker","run","--rm","--network=none","--read-only","--cap-drop=ALL","--security-opt=no-new-privileges","--pids-limit=256","--memory=1g","--cpus=2","--user=65534:65534","--tmpfs=/tmp:rw,exec,size=768m","-e","GOCACHE=/tmp/go-build","-e","GOPATH=/tmp/go","-e","GOTOOLCHAIN=local","-e","GOPROXY=off","-v",f"{work}:/work:ro","-w","/work","golang:1.27.1",*cmd]
                try:
                    r=subprocess.run(cmd,cwd=work,text=True,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=180)
                    log=r.stdout;code=r.returncode
                except subprocess.TimeoutExpired as e: log=str(e);code=124
                (out/f"{t}.jsonl").write_text(log)
                pkg=f"industry_backend_go/tasks/{t}";report[pkg]={"status":statuses(log,code,pkg),"exit_code":code}
    except (ValueError,OSError,subprocess.CalledProcessError) as e: guard["report"]["error"]=str(e)
    (out/"package-results.json").write_text(json.dumps(report,indent=2)+"\n")
    (out/"change-policy-result.json").write_text(json.dumps(guard,indent=2)+"\n")
    payload={"schema":"github-actions-analytics-v2","stream":cfg["stream"],"github":{k:os.getenv(v,"") for k,v in {"repository":"CANDIDATE_REPOSITORY","sha":"CANDIDATE_SHA","actor":"GITHUB_ACTOR","run_id":"GITHUB_RUN_ID","run_attempt":"GITHUB_RUN_ATTEMPT"}.items()},"baseline":cfg["diff"]["original"],"config":{"allow_list_count":len(cfg["diff"]["allow_list"])},"guard":guard,"test_report":report}
    (out/"analytics.json").write_text(json.dumps(payload,indent=2)+"\n")
    return 0 if guard["checkCode"]=="0" and all(x["status"]=="pass" for x in report.values()) else 1
if __name__=="__main__": raise SystemExit(main())
