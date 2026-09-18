// Command grade runs the instructor-owned course checks.
// The candidate is treated solely as input: only its permitted solution.go
// files are copied into a temporary workspace with instructor tests.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// testCount is how many times every instructor test must pass.
const testCount = 4

var tasks = []string{"task_00", "task_01", "task_02", "task_03", "task_04", "task_05", "task_06", "task_07", "task_08", "task_09", "task_10"}

type config struct {
	Stream string `json:"stream"`
	Diff   struct {
		Original struct {
			Repo string `json:"repo"`
			Ref  string `json:"ref"`
		} `json:"original"`
		AllowList []string `json:"allow_list"`
	} `json:"diff"`
}

type packageResult struct {
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code,omitempty"`
}

type guardResult struct {
	CheckCode string         `json:"checkCode"`
	Report    map[string]any `json:"report"`
}

func main() {
	candidateFlag := flag.String("candidate", "", "candidate repository directory")
	outFlag := flag.String("out", "", "output directory")
	baselineFlag := flag.String("baseline", ".", "trusted instructor repository directory")
	local := flag.Bool("local", false, "run tests directly, without Docker")
	flag.Parse()
	if *candidateFlag == "" || *outFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: grade --candidate DIR --out DIR [--local]")
		os.Exit(2)
	}

	baseline, err := filepath.Abs(*baselineFlag)
	if err != nil {
		fatal(err)
	}
	candidate, err := filepath.Abs(*candidateFlag)
	if err != nil {
		fatal(err)
	}
	out, err := filepath.Abs(*outFlag)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fatal(err)
	}

	cfg, err := readConfig(filepath.Join(baseline, ".etc", "config.json"))
	if err != nil {
		fatal(err)
	}
	report := initialReport()
	guard := guardResult{CheckCode: "1", Report: map[string]any{}}

	if changed, err := checkPolicy(baseline, candidate, cfg); err != nil {
		guard.Report["error"] = err.Error()
	} else if err := checkSources(candidate); err != nil {
		guard.Report["error"] = err.Error()
	} else {
		guard = guardResult{CheckCode: "0", Report: map[string]any{"changed_paths": changed}}
		report = runTests(baseline, candidate, out, *local)
	}

	writeJSON(filepath.Join(out, "package-results.json"), report)
	writeJSON(filepath.Join(out, "change-policy-result.json"), guard)
	payload := map[string]any{
		"schema": "github-actions-analytics-v2",
		"stream": cfg.Stream,
		"github": map[string]string{
			"repository": os.Getenv("CANDIDATE_REPOSITORY"), "sha": os.Getenv("CANDIDATE_SHA"),
			"actor": os.Getenv("GITHUB_ACTOR"), "run_id": os.Getenv("GITHUB_RUN_ID"), "run_attempt": os.Getenv("GITHUB_RUN_ATTEMPT"),
		},
		"baseline": map[string]string{"repo": cfg.Diff.Original.Repo, "ref": cfg.Diff.Original.Ref},
		"config":   map[string]int{"allow_list_count": len(cfg.Diff.AllowList)},
		"guard":    guard, "test_report": report,
	}
	writeJSON(filepath.Join(out, "analytics.json"), payload)
	if guard.CheckCode != "0" || !allPass(report) {
		os.Exit(1)
	}
}

func readConfig(path string) (config, error) {
	var c config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(b, &c)
}

func initialReport() map[string]packageResult {
	r := make(map[string]packageResult, len(tasks))
	for _, task := range tasks {
		r["industry_backend_go/tasks/"+task] = packageResult{Status: "missing"}
	}
	return r
}

func allPass(report map[string]packageResult) bool {
	for _, v := range report {
		if v.Status != "pass" {
			return false
		}
	}
	return true
}

func checkPolicy(baseline, candidate string, cfg config) ([]string, error) {
	allowed := map[string]bool{}
	// Ignored paths may differ freely: they never reach the test workspace.
	// Fork CI commits fresh badges, and students may tune .gitignore.
	ignored := map[string]bool{".gitignore": true}
	for _, task := range tasks {
		allowed["tasks/"+task+"/solution.go"] = true
		ignored["badges/tasks/"+task+".svg"] = true
	}
	if len(cfg.Diff.AllowList) != len(allowed) {
		return nil, errors.New("invalid instructor allow-list")
	}
	for _, path := range cfg.Diff.AllowList {
		if !allowed[path] {
			return nil, errors.New("invalid instructor allow-list")
		}
	}
	a, err := snapshot(baseline)
	if err != nil {
		return nil, err
	}
	b, err := snapshot(candidate)
	if err != nil {
		return nil, err
	}
	paths := map[string]bool{}
	for p := range a {
		paths[p] = true
	}
	for p := range b {
		paths[p] = true
	}
	changed := make([]string, 0)
	for p := range paths {
		if a[p] != b[p] && !ignored[p] {
			if !allowed[p] || b[p] == "" {
				return nil, fmt.Errorf("forbidden changes: %s", p)
			}
			changed = append(changed, p)
		}
	}
	sort.Strings(changed)
	return changed, nil
}

func snapshot(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			return filepath.SkipDir
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink forbidden: %s", rel)
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("non-regular file: %s", rel)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err = io.Copy(h, f); err != nil {
			return err
		}
		result[filepath.ToSlash(rel)] = hex.EncodeToString(h.Sum(nil))
		return nil
	})
	return result, err
}

// forbiddenImports lets a solution leave the test process by itself (raw exit,
// signals, subprocesses), write files or reach into the testing machinery
// (flag exposes the test.* flags).
// Without them the test binary exits 0 only when testing.M.Run reports success.
var forbiddenImports = []string{"C", "unsafe", "os", "syscall", "runtime", "testing", "plugin", "flag", "io/ioutil", "log/syslog", "internal", "cmd", "vendor"}

// initCalls may appear in package-level variable initializers, e.g. sentinel errors.
var initCalls = map[string]bool{"errors.New": true, "fmt.Errorf": true}

func checkSources(candidate string) error {
	output, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return err
	}
	goRoot := strings.TrimSpace(string(output))
	if goRoot == "" {
		return errors.New("Go installation has no GOROOT")
	}
	for _, task := range tasks {
		path := filepath.Join(candidate, "tasks", task, "solution.go")
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		if err := checkSource(goRoot, f); err != nil {
			return fmt.Errorf("%s: %w", task, err)
		}
	}
	return nil
}

func checkSource(goRoot string, f *ast.File) error {
	if f.Name.Name != "main" {
		return errors.New("package must be main")
	}
	imported := map[string]bool{}
	for _, imp := range f.Imports {
		pkg := strings.Trim(imp.Path.Value, `"`)
		if !isStdlibPackage(goRoot, pkg) {
			return fmt.Errorf("only stdlib imports are allowed: %s", pkg)
		}
		if isForbiddenImport(pkg) {
			return fmt.Errorf("import is forbidden: %s", pkg)
		}
		if imp.Name == nil {
			imported[pkg] = true
		}
	}
	for _, group := range f.Comments {
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, "//go:") || strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(c.Text, "//")), "+build") {
				return errors.New("compiler/build directives are forbidden")
			}
		}
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "init" {
				return errors.New("func init is forbidden")
			}
		case *ast.GenDecl:
			if d.Tok != token.VAR {
				continue
			}
			for _, spec := range d.Specs {
				for _, value := range spec.(*ast.ValueSpec).Values {
					if call := forbiddenInitCall(value, imported); call != "" {
						return fmt.Errorf("package-level variables may only call errors.New or fmt.Errorf, found %s", call)
					}
				}
			}
		}
	}
	return nil
}

func isForbiddenImport(pkg string) bool {
	for _, bad := range forbiddenImports {
		if pkg == bad || strings.HasPrefix(pkg, bad+"/") {
			return true
		}
	}
	return false
}

// forbiddenInitCall returns the first call in a package-level initializer
// that is not an allowed constructor. Function literal bodies are skipped:
// they do not run during initialization unless called, and that call is found.
func forbiddenInitCall(expr ast.Expr, imported map[string]bool) string {
	found := ""
	ast.Inspect(expr, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			name := callName(n.Fun)
			pkg, _, _ := strings.Cut(name, ".")
			if !initCalls[name] || !imported[pkg] {
				found = name
				if found == "" {
					found = "a function call"
				}
				return false
			}
		}
		return true
	})
	return found
}

func callName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name
		}
	}
	return ""
}

func isStdlibPackage(goRoot, pkg string) bool {
	if pkg == "" || strings.HasPrefix(pkg, ".") || strings.Contains(pkg, "\\") {
		return false
	}
	info, err := os.Stat(filepath.Join(goRoot, "src", filepath.FromSlash(pkg)))
	return err == nil && info.IsDir()
}

func runTests(baseline, candidate, out string, local bool) map[string]packageResult {
	report := initialReport()
	work, err := os.MkdirTemp("", "autumn-grade-")
	if err != nil {
		return report
	}
	defer os.RemoveAll(work)
	if err := os.Chmod(work, 0o755); err != nil {
		return report
	}
	if err := copyFile(filepath.Join(baseline, "go.mod"), filepath.Join(work, "go.mod")); err != nil {
		return report
	}
	if err := copyDir(filepath.Join(baseline, "tasks"), filepath.Join(work, "tasks")); err != nil {
		return report
	}
	for _, task := range tasks {
		if err := copyFile(filepath.Join(candidate, "tasks", task, "solution.go"), filepath.Join(work, "tasks", task, "solution.go")); err != nil {
			return report
		}
	}
	bin, err := os.MkdirTemp("", "autumn-bin-")
	if err != nil {
		return report
	}
	defer os.RemoveAll(bin)
	// The sandbox user writes the test binaries here.
	if err := os.Chmod(bin, 0o777); err != nil {
		return report
	}

	expected := map[string][]string{}
	var buildable []string
	for _, task := range tasks {
		names, err := instructorTests(filepath.Join(baseline, "tasks", task))
		if err != nil || len(names) == 0 {
			report["industry_backend_go/tasks/"+task] = packageResult{Status: "unknown"}
			continue
		}
		expected[task] = names
		buildable = append(buildable, task)
	}

	buildAll(work, bin, local, buildable)
	for _, task := range buildable {
		key := "industry_backend_go/tasks/" + task
		output, code := buildResult(bin, task)
		if code == 0 {
			output, code = runTest(work, bin, local, task)
			report[key] = packageResult{Status: testStatus(output, code, key, expected[task]), ExitCode: code}
		} else {
			report[key] = packageResult{Status: "fail", ExitCode: code}
		}
		_ = os.WriteFile(filepath.Join(out, task+".jsonl"), output, 0o644)
	}
	return report
}

// buildScript compiles every task in one sandbox, so the race-instrumented
// standard library is built once. Compiling runs no candidate code: cgo and
// compiler directives are rejected earlier. A plain build ignores _test.go
// files, so a solution that calls instructor test helpers fails there.
const buildScript = `for task in "$@"; do
	{ go build -o /dev/null "./tasks/$task" && go test -race -c -o "$OUT/$task.test" "./tasks/$task"; } >"$OUT/$task.build" 2>&1
	echo $? >"$OUT/$task.code"
done`

func buildAll(work, bin string, local bool, tasks []string) {
	args := append([]string{"sh", "-c", buildScript, "build"}, tasks...)
	if local {
		runCommand(work, buildTimeout, nil, append([]string{"env", "OUT=" + bin}, args...)...)
		return
	}
	runCommand(work, buildTimeout, nil, append(dockerRun("/work", "-e", "OUT=/out",
		"-v", work+":/work:ro", "-v", bin+":/out"), args...)...)
}

// buildResult reports a task's build log and exit code; a build that never
// finished counts as failed.
func buildResult(bin, task string) ([]byte, int) {
	log, _ := os.ReadFile(filepath.Join(bin, task+".build"))
	raw, err := os.ReadFile(filepath.Join(bin, task+".code"))
	if err != nil {
		return append(log, "\nbuild did not finish\n"...), 124
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return append(log, "\nbuild status unreadable\n"...), 124
	}
	return log, code
}

// runTest runs one task's test binary in its own sandbox that sees only the
// binary and the task directory, then converts the output to test2json events
// on the host, as go test -json would.
func runTest(work, bin string, local bool, task string) ([]byte, int) {
	flags := []string{fmt.Sprintf("-test.count=%d", testCount), "-test.timeout=60s", "-test.v=test2json"}
	binary := filepath.Join(bin, task+".test")
	var output []byte
	var code int
	if local {
		output, code = runCommand(filepath.Join(work, "tasks", task), testTimeout, nil, append([]string{binary}, flags...)...)
	} else {
		args := dockerRun("/work", "-v", filepath.Join(work, "tasks", task)+":/work:ro", "-v", binary+":/task.test:ro")
		output, code = runCommand(work, testTimeout, nil, append(append(args, "/task.test"), flags...)...)
	}
	events, _ := runCommand(work, testTimeout, output, "go", "tool", "test2json", "-p", "industry_backend_go/tasks/"+task)
	return events, code
}

const (
	buildTimeout = 10 * time.Minute
	testTimeout  = 3 * time.Minute
)

func dockerRun(workdir string, extra ...string) []string {
	args := []string{"docker", "run", "--rm", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=1g", "--cpus=2", "--user=65534:65534", "--tmpfs=/tmp:rw,exec,size=768m", "-e", "GOCACHE=/tmp/go-build", "-e", "GOPATH=/tmp/go", "-e", "GOTOOLCHAIN=local", "-e", "GOPROXY=off", "-w", workdir}
	return append(append(args, extra...), "golang:1.27.1")
}

// instructorTests lists top-level Test functions of the trusted test files.
func instructorTests(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, path := range files {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") && fn.Name.Name != "TestMain" {
				names = append(names, fn.Name.Name)
			}
		}
	}
	sort.Strings(names)
	return names, nil
}

func runCommand(dir string, timeout time.Duration, stdin []byte, args ...string) ([]byte, int) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Dir = dir
	if stdin != nil {
		command.Stdin = bytes.NewReader(stdin)
	}
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return append(output, []byte("\n"+ctx.Err().Error())...), 124
	}
	if err == nil {
		return output, 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return output, exit.ExitCode()
	}
	return append(output, []byte("\n"+err.Error())...), 124
}

// testStatus fails closed: the run must exit 0, report package success, have
// no failed or skipped tests, and every instructor test must pass exactly
// testCount times, so a truncated or forged event stream is rejected.
func testStatus(raw []byte, code int, pkg string, expected []string) string {
	final, bad := "", false
	passes := map[string]int{}
	for _, line := range strings.Split(string(raw), "\n") {
		var event struct{ Package, Action, Test string }
		if json.Unmarshal([]byte(line), &event) != nil || event.Package != pkg {
			continue
		}
		switch {
		case event.Test == "" && (event.Action == "pass" || event.Action == "fail" || event.Action == "skip"):
			final = event.Action
		case event.Action == "fail" || event.Action == "skip":
			bad = true
		case event.Action == "pass" && !strings.Contains(event.Test, "/"):
			passes[event.Test]++
		}
	}
	if code != 0 || final != "pass" || bad {
		return "fail"
	}
	for _, name := range expected {
		if passes[name] != testCount {
			return "fail"
		}
	}
	return "pass"
}

func copyFile(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
func copyDir(from, to string) error {
	return filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}
func writeJSON(path string, value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
