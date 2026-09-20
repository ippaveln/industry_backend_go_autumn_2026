package main

import (
	"fmt"
)

func main() {
	c := NewLRUCache[string, int](2)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("a", 3)
	c.Set("c", 4)
	fmt.Println(c.Get("a"))
}
