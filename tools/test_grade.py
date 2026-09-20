import importlib.util, json, pathlib, tempfile, unittest
spec=importlib.util.spec_from_file_location("grade",pathlib.Path(__file__).with_name("grade.py"));grade=importlib.util.module_from_spec(spec);spec.loader.exec_module(grade)
class GradeTests(unittest.TestCase):
 def test_fail_closed(self):
  pkg="industry_backend_go/tasks/task_00"
  def events(*items): return "\n".join(json.dumps(dict(Package=pkg,**e)) for e in items)
  valid=events({"Action":"pass","Test":"TestGreeting"},{"Action":"pass"})
  self.assertEqual(grade.statuses(valid,0,pkg),"pass")
  for data,code in [("",0),(events({"Action":"skip"}),0),(events({"Action":"pass"}),0),(valid,1),(valid+"\n"+events({"Action":"fail"}),0),(events({"Test":"TestGreeting","Action":"skip"},{"Action":"pass"}),0)]: self.assertEqual(grade.statuses(data,code,pkg),"fail")
 def test_policy(self):
  with tempfile.TemporaryDirectory() as temp:
   a=pathlib.Path(temp)/"a";b=pathlib.Path(temp)/"b"
   for p in (a,b):
    (p/".etc").mkdir(parents=True);(p/".etc/config.json").write_text(json.dumps({"diff":{"allow_list":[f"tasks/{t}/solution.go" for t in grade.TASKS]}}));(p/"tasks/task_00").mkdir(parents=True);(p/"tasks/task_00/solution.go").write_text("package main")
   (b/"tasks/task_00/solution.go").write_text("package main // changed");grade.policy(a,b)
   (b/"hack.py").write_text("bad")
   with self.assertRaises(ValueError):grade.policy(a,b)
   (b/"hack.py").unlink();(b/"tasks/task_00/solution.go").unlink()
   with self.assertRaises(ValueError):grade.policy(a,b)
   (b/"tasks/task_00/solution.go").symlink_to(a/"tasks/task_00/solution.go")
   with self.assertRaises(ValueError):grade.policy(a,b)
if __name__=="__main__":unittest.main()
