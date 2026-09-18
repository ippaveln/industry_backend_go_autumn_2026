package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	in := flag.String("in", "", "package-results.json")
	out := flag.String("out", "", "output directory")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: badges --in FILE --out DIR")
		os.Exit(2)
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		panic(err)
	}
	var report map[string]struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(b, &report); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		panic(err)
	}
	for i := 0; i <= 10; i++ {
		task := fmt.Sprintf("task_%02d", i)
		status := report["industry_backend_go/tasks/"+task].Status
		if status != "pass" && status != "fail" && status != "missing" && status != "unknown" {
			status = "unknown"
		}
		color := "#666"
		if status == "pass" {
			color = "#4c1"
		}
		if status == "fail" {
			color = "#e05d44"
		}
		data := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="150" height="20"><rect width="150" height="20" fill="%s"/><text x="5" y="14" fill="white">%s: %s</text></svg>`, color, task, status)
		if err := os.WriteFile(filepath.Join(*out, task+".svg"), []byte(data), 0o644); err != nil {
			panic(err)
		}
	}
}
