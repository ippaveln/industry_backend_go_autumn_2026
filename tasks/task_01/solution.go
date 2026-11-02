package main

import (
	"fmt"
	"strings"
)

func greet(name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = "World"
	}
	return fmt.Sprintf("Hello, %s!", trimmedName)
}
