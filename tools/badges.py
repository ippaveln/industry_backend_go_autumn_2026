import json,pathlib,sys
r=json.loads(pathlib.Path(sys.argv[1]).read_text());out=pathlib.Path(sys.argv[2]);out.mkdir(parents=True,exist_ok=True)
for i in range(11):
 t=f"task_{i:02}";status=r.get(f"industry_backend_go/tasks/{t}",{}).get("status","missing")
 if status not in ("pass","fail","missing","unknown"):status="unknown"
 color={"pass":"#4c1","fail":"#e05d44"}.get(status,"#666")
 (out/f"{t}.svg").write_text(f'<svg xmlns="http://www.w3.org/2000/svg" width="150" height="20"><rect width="150" height="20" fill="{color}"/><text x="5" y="14" fill="white">{t}: {status}</text></svg>')
