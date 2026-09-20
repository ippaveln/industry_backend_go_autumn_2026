package main

import (
	"fmt"
)

func main() { c := NewLRUCache[string, int](2); c.Set("a", 1); fmt.Println(c.Get("a")) }
