package main

import (
	"fmt"
)

func main() {
	c := NewCache[string, int](1)
	fmt.Println(c.Set("a", 1), c.Set("b", 2))
	fmt.Println(c.Get("a"))
}
