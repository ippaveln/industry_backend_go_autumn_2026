package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	results := flag.String("results", "", "results directory")
	task := flag.String("task", "", "task id, e.g. task_00")
	flag.Parse()
	if *results == "" || *task == "" {
		fmt.Fprintln(os.Stderr, "usage: checkstatus --results DIR --task task_00")
		os.Exit(2)
	}
	guardBytes, err := os.ReadFile(filepath.Join(*results, "change-policy-result.json"))
	if err != nil {
		panic(err)
	}
	var guard struct {
		CheckCode string         `json:"checkCode"`
		Report    map[string]any `json:"report"`
	}
	if err := json.Unmarshal(guardBytes, &guard); err != nil {
		panic(err)
	}
	reportBytes, err := os.ReadFile(filepath.Join(*results, "package-results.json"))
	if err != nil {
		panic(err)
	}
	var report map[string]struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		panic(err)
	}
	if guard.CheckCode != "0" {
		fmt.Printf("%s: file policy violated: %v\n", *task, guard.Report["error"])
		fmt.Println("only tasks/task_XX/solution.go, .gitignore and badges may differ from upstream master; sync the fork and revert other changes")
		os.Exit(1)
	}
	status := report["industry_backend_go/tasks/"+*task].Status
	fmt.Printf("%s: %s\n", *task, status)
	if status != "pass" {
		fmt.Printf("test output: artifact autumn-results, file %s.jsonl\n", *task)
		os.Exit(1)
	}
}
