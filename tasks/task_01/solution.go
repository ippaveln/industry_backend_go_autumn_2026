package main

import "strings"

func greet(name string) string {
	var trimName string = strings.TrimSpace(name)
	if trimName == "" {
		return "Hello, World!"
	} else {
		return "Hello, " + trimName + "!"
	}
}
