package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const statusPkg = "industry_backend_go/tasks/task_00"

var statusTests = []string{"TestGreeting"}

func TestTestStatus_Pass(t *testing.T) {
	raw := jsonl(passes("TestGreeting", testCount), event("pass", "TestGreeting/sub"), event("pass", ""))

	got := testStatus(raw, 0, statusPkg, statusTests)

	if got != "pass" {
		t.Fatalf("got %q, want %q", got, "pass")
	}
}

func TestTestStatus_FailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  []byte
		code int
	}{
		{name: "empty output", raw: nil, code: 0},
		{name: "no test events", raw: jsonl(event("pass", "")), code: 0},
		{name: "non-zero exit code", raw: jsonl(passes("TestGreeting", testCount), event("pass", "")), code: 1},
		{name: "skipped test", raw: jsonl(event("skip", "TestGreeting"), passes("TestGreeting", testCount), event("pass", "")), code: 0},
		{name: "failed subtest", raw: jsonl(event("fail", "TestGreeting/sub"), passes("TestGreeting", testCount), event("pass", "")), code: 0},
		{name: "too few runs", raw: jsonl(passes("TestGreeting", testCount-1), event("pass", "")), code: 0},
		{name: "forged extra run", raw: jsonl(passes("TestGreeting", testCount+1), event("pass", "")), code: 0},
		{name: "only unknown tests", raw: jsonl(passes("TestFake", testCount), event("pass", "")), code: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := testStatus(tc.raw, tc.code, statusPkg, statusTests)

			if got != "fail" {
				t.Fatalf("got %q, want %q", got, "fail")
			}
		})
	}
}

func TestCheckSource_Accepts(t *testing.T) {
	src := `package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrZero = errors.New("zero")

var (
	errWrapped = fmt.Errorf("wrapped: %w", ErrZero)
	timeout    = 2 * time.Second
	handler    = func() error { return run() }
	mu         sync.Mutex
)

func run() error { return nil }
`

	err := checkSource(goRoot(t), parseSource(t, src))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckSource_Rejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{name: "wrong package", src: "package solution"},
		{name: "cgo", src: "package main\nimport \"C\""},
		{name: "third-party import", src: "package main\nimport \"github.com/x/y\""},
		{name: "os import", src: "package main\nimport \"os\""},
		{name: "os subpackage", src: "package main\nimport \"os/exec\""},
		{name: "syscall import", src: "package main\nimport \"syscall\""},
		{name: "unsafe import", src: "package main\nimport \"unsafe\""},
		{name: "runtime subpackage", src: "package main\nimport \"runtime/debug\""},
		{name: "testing import", src: "package main\nimport \"testing\""},
		{name: "flag import", src: "package main\nimport \"flag\""},
		{name: "go directive", src: "package main\n//go:noinline\nfunc f() {}"},
		{name: "build tag", src: "//+build ignore\n\npackage main"},
		{name: "init func", src: "package main\nfunc init() {}"},
		{name: "call in var", src: "package main\nfunc f() int { return 1 }\nvar x = f()"},
		{name: "called func literal", src: "package main\nvar x = func() int { return 1 }()"},
		{name: "nested call in allowed call", src: "package main\nimport \"errors\"\nfunc f() string { return \"\" }\nvar e = errors.New(f())"},
		{name: "aliased errors", src: "package main\nimport e \"errors\"\nvar x = e.New(\"x\")"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := parseSource(t, tc.src)

			err := checkSource(goRoot(t), f)

			if err == nil {
				t.Fatal("forbidden source accepted")
			}
		})
	}
}

func TestInstructorTests(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a_test.go"), "package main\nimport \"testing\"\nfunc TestB(t *testing.T) {}\nfunc TestMain(m *testing.M) {}\nfunc helper() {}")
	writeFile(t, filepath.Join(dir, "b_test.go"), "package main\nimport \"testing\"\nfunc TestA(t *testing.T) {}")
	writeFile(t, filepath.Join(dir, "solution.go"), "package main\nfunc TestNotATest() {}")

	got, err := instructorTests(dir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"TestA", "TestB"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCheckPolicy_AllowsSolutionChange(t *testing.T) {
	baseline, candidate, cfg := setupPolicyRepos(t)
	writeFile(t, filepath.Join(candidate, "tasks", "task_00", "solution.go"), "package main // changed")

	_, err := checkPolicy(baseline, candidate, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPolicy_RejectsForbiddenFile(t *testing.T) {
	baseline, candidate, cfg := setupPolicyRepos(t)
	writeFile(t, filepath.Join(candidate, "hack.go"), "bad")

	_, err := checkPolicy(baseline, candidate, cfg)

	if err == nil {
		t.Fatal("forbidden file accepted")
	}
}

func TestCheckPolicy_IgnoresBadgesAndGitignore(t *testing.T) {
	baseline, candidate, cfg := setupPolicyRepos(t)
	writeFile(t, filepath.Join(baseline, "badges", "tasks", "task_00.svg"), "<svg>unknown</svg>")
	writeFile(t, filepath.Join(candidate, "badges", "tasks", "task_00.svg"), "<svg>pass</svg>")
	writeFile(t, filepath.Join(candidate, ".gitignore"), "*.local\n")

	changed, err := checkPolicy(baseline, candidate, cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("ignored paths reported as changed: %v", changed)
	}
}

func TestCheckPolicy_RejectsUnknownBadge(t *testing.T) {
	baseline, candidate, cfg := setupPolicyRepos(t)
	writeFile(t, filepath.Join(candidate, "badges", "tasks", "extra.svg"), "<svg/>")

	_, err := checkPolicy(baseline, candidate, cfg)

	if err == nil {
		t.Fatal("unknown badge accepted")
	}
}

func event(action, test string) string {
	b, _ := json.Marshal(map[string]string{"Package": statusPkg, "Action": action, "Test": test})
	return string(b)
}

func passes(test string, n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = event("pass", test)
	}
	return strings.Join(lines, "\n")
}

func jsonl(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n"))
}

// setupPolicyRepos creates identical baseline and candidate trees with
// instructor config and stub solutions for every task.
func setupPolicyRepos(t *testing.T) (baseline, candidate string, cfg config) {
	t.Helper()
	root := t.TempDir()
	baseline, candidate = filepath.Join(root, "baseline"), filepath.Join(root, "candidate")

	allow := make([]string, 0, len(tasks))
	for _, task := range tasks {
		allow = append(allow, "tasks/"+task+"/solution.go")
	}
	data, err := json.Marshal(map[string]any{"diff": map[string]any{"allow_list": allow}})
	must(t, err)

	for _, dir := range []string{baseline, candidate} {
		writeFile(t, filepath.Join(dir, ".etc", "config.json"), string(data))
		for _, task := range tasks {
			writeFile(t, filepath.Join(dir, "tasks", task, "solution.go"), "package main")
		}
	}

	cfg, err = readConfig(filepath.Join(baseline, ".etc", "config.json"))
	must(t, err)
	return baseline, candidate, cfg
}

func parseSource(t *testing.T, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "solution.go", src, parser.ParseComments)
	must(t, err)
	return f
}

func goRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOROOT").Output()
	must(t, err)
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	must(t, os.WriteFile(path, []byte(content), 0o644))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
