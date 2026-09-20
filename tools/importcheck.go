package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	b, e := exec.Command("go", "list", "std").Output()
	if e != nil {
		panic(e)
	}
	std := map[string]bool{}
	for _, p := range strings.Fields(string(b)) {
		std[p] = true
	}
	var files []string
	if e = json.NewDecoder(os.Stdin).Decode(&files); e != nil {
		panic(e)
	}
	for _, p := range files {
		f, e := parser.ParseFile(token.NewFileSet(), p, nil, parser.ParseComments)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		if f.Name.Name != "main" {
			panic("package must be main")
		}
		for _, im := range f.Imports {
			v, _ := strconv.Unquote(im.Path.Value)
			if !std[v] || v == "C" {
				panic("only stdlib imports: " + v)
			}
		}
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if strings.HasPrefix(c.Text, "//go:") || strings.HasPrefix(c.Text, "// +build") {
					panic("compiler/build directives forbidden: " + filepath.Base(p))
				}
			}
		}
	}
}
