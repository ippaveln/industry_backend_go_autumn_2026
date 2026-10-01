package main

import (
	"strings"
)

func greet(name string) string {
	trim_string := strings.TrimSpace(name)
	if trim_string == "" {
		return "Hello, World!"
	}
	return "Hello, " + trim_string + "!"

}
