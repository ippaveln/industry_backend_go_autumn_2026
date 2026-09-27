// Command grade runs the instructor-owned course checks.
// The candidate is treated solely as input: only its permitted solution.go
// files are copied into a temporary workspace with instructor tests.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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
	} else if err := checkImports(candidate); err != nil {
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
	for _, task := range tasks {
		allowed["tasks/"+task+"/solution.go"] = true
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
		if a[p] != b[p] {
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

func checkImports(candidate string) error {
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
		if f.Name.Name != "main" {
			return fmt.Errorf("%s: package must be main", task)
		}
		for _, imp := range f.Imports {
			pkg := strings.Trim(imp.Path.Value, `"`)
			if pkg == "C" || !isStdlibPackage(goRoot, pkg) {
				return fmt.Errorf("%s: only stdlib imports are allowed: %s", task, pkg)
			}
		}
		for _, group := range f.Comments {
			for _, c := range group.List {
				if strings.HasPrefix(c.Text, "//go:") || strings.HasPrefix(c.Text, "// +build") {
					return fmt.Errorf("%s: compiler/build directives are forbidden", task)
				}
			}
		}
	}
	return nil
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
	for _, task := range tasks {
		cmd := []string{"go", "test", "-race", "-count=4", "-timeout=60s", "-json", "./tasks/" + task}
		if !local {
			cmd = append([]string{"docker", "run", "--rm", "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=256", "--memory=1g", "--cpus=2", "--user=65534:65534", "--tmpfs=/tmp:rw,exec,size=768m", "-e", "GOCACHE=/tmp/go-build", "-e", "GOPATH=/tmp/go", "-e", "GOTOOLCHAIN=local", "-e", "GOPROXY=off", "-v", work + ":/work:ro", "-w", "/work", "golang:1.27.1"}, cmd...)
		}
		ctx, cancel := contextCommand(cmd, work)
		output, code := ctx()
		cancel()
		_ = os.WriteFile(filepath.Join(out, task+".jsonl"), output, 0o644)
		key := "industry_backend_go/tasks/" + task
		report[key] = packageResult{Status: testStatus(output, code, key), ExitCode: code}
	}
	return report
}

func contextCommand(args []string, dir string) (func() ([]byte, int), func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Dir = dir
	return func() ([]byte, int) {
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
	}, cancel
}

func testStatus(raw []byte, code int, pkg string) string {
	final, passed, bad := "", false, false
	for _, line := range strings.Split(string(raw), "\n") {
		var event struct{ Package, Action, Test string }
		if json.Unmarshal([]byte(line), &event) != nil || event.Package != pkg {
			continue
		}
		if event.Test != "" {
			if event.Action == "pass" {
				passed = true
			}
			if event.Action == "fail" || event.Action == "skip" {
				bad = true
			}
		} else if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
			final = event.Action
		}
	}
	if code == 0 && final == "pass" && passed && !bad {
		return "pass"
	}
	return "fail"
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
