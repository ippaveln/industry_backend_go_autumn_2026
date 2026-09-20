package main

import (
	"sync"
	"testing"
)

func TestConcurrentValues(t *testing.T) {
	type pair struct{ A, B int }
	c := NewLRUCache[int, pair](32)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			<-start
			for i := 0; i < 500; i++ {
				k := (i + w) % 32
				c.Set(k, pair{i, -i})
				if v, ok := c.Get(k); ok && v.A != -v.B {
					t.Error("torn value")
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
}
