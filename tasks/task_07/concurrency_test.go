package main

import (
	"sync"
	"testing"
)

func TestConcurrentValues(t *testing.T) {
	const (
		workers  = 16
		ops      = 500
		keys     = 32
		capacity = 8 // smaller than the key space, so Set also evicts concurrently
	)
	type pair struct{ A, B int }
	c := NewLRUCache[int, pair](capacity)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for w := range workers {
		wg.Go(func() {
			<-start
			for i := range ops {
				k := (i + w) % keys
				c.Set(k, pair{i, -i})
				if v, ok := c.Get(k); ok && v.A != -v.B {
					t.Errorf("torn value for key %d: %+v", k, v)
				}
			}
		})
	}

	close(start)
	wg.Wait()
}
