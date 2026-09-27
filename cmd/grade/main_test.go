package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTestStatusFailsClosed(t *testing.T) {
	pkg := "industry_backend_go/tasks/task_00"
	line := func(action, test string) string {
		b, _ := json.Marshal(map[string]string{"Package": pkg, "Action": action, "Test": test})
		return string(b)
	}
	valid := []byte(line("pass", "TestGreeting") + "\n" + line("pass", ""))
	if got := testStatus(valid, 0, pkg); got != "pass" {
		t.Fatal(got)
	}
	for _, c := range []struct {
		raw  []byte
		code int
	}{{nil, 0}, {[]byte(line("pass", "")), 0}, {valid, 1}, {[]byte(line("skip", "TestGreeting") + "\n" + line("pass", "")), 0}} {
		if got := testStatus(c.raw, c.code, pkg); got != "fail" {
			t.Fatal(got)
		}
	}
}

func TestPolicyRejectsForbiddenChangesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	allow := make([]string, 0, len(tasks))
	for _, task := range tasks {
		allow = append(allow, "tasks/"+task+"/solution.go")
	}
	data, _ := json.Marshal(map[string]any{"diff": map[string]any{"allow_list": allow}})
	for _, dir := range []string{a, b} {
		must(t, os.MkdirAll(filepath.Join(dir, ".etc"), 0o755))
		must(t, os.WriteFile(filepath.Join(dir, ".etc", "config.json"), data, 0o644))
		for _, task := range tasks {
			must(t, os.MkdirAll(filepath.Join(dir, "tasks", task), 0o755))
			must(t, os.WriteFile(filepath.Join(dir, "tasks", task, "solution.go"), []byte("package main"), 0o644))
		}
	}
	cfg, err := readConfig(filepath.Join(a, ".etc", "config.json"))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(b, "tasks", "task_00", "solution.go"), []byte("package main // changed"), 0o644))
	if _, err := checkPolicy(a, b, cfg); err != nil {
		t.Fatal(err)
	}
	must(t, os.WriteFile(filepath.Join(b, "hack.go"), []byte("bad"), 0o644))
	if _, err := checkPolicy(a, b, cfg); err == nil {
		t.Fatal("forbidden file accepted")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
