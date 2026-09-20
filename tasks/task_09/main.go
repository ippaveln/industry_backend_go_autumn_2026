package main

import (
	"context"
	"fmt"
)

func main() {
	r, e := ParallelMap(context.Background(), 2, []int{1, 2}, func(_ context.Context, n int) (int, error) { return n * n, nil })
	fmt.Println(r, e)
}
