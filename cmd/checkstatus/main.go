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
		CheckCode string `json:"checkCode"`
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
	if guard.CheckCode != "0" || report["industry_backend_go/tasks/"+*task].Status != "pass" {
		os.Exit(1)
	}
}
