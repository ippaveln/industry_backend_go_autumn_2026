package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *fakeClock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }
func TestAtomicSpend(t *testing.T) {
	c := &fakeClock{}
	l := NewLimiter(c, 0, 3)
	if l.AllowN(0) || l.AllowN(-1) || l.AllowN(4) {
		t.Fatal("invalid spend")
	}
	if !l.AllowN(2) || l.AllowN(2) || !l.AllowN(1) || l.AllowN(1) {
		t.Fatal("failed spend must not consume tokens")
	}
}
func TestRefillAndRollback(t *testing.T) {
	c := &fakeClock{}
	l := NewLimiter(c, 2.5, 3)
	if !l.AllowN(3) {
		t.Fatal("initial full")
	}
	c.add(200 * time.Millisecond)
	if l.AllowN(1) {
		t.Fatal("fraction")
	}
	c.add(200 * time.Millisecond)
	if !l.AllowN(1) {
		t.Fatal("refill")
	}
	c.add(-time.Second)
	if l.AllowN(1) {
		t.Fatal("rollback")
	}
	c.add(time.Second)
	if l.AllowN(1) {
		t.Fatal("double refill after rollback")
	}
	c.add(10 * time.Second)
	if !l.AllowN(3) || l.AllowN(1) {
		t.Fatal("cap")
	}
}
func TestDisabled(t *testing.T) {
	for _, n := range []int{-1, 0} {
		if NewLimiter(&fakeClock{}, 1, n).AllowN(1) {
			t.Fatal("burst")
		}
	}
	if NewLimiter(nil, 1, 1).AllowN(1) {
		t.Fatal("nil clock")
	}
	l := NewLimiter(&fakeClock{}, -1, 1)
	if !l.AllowN(1) || l.AllowN(1) {
		t.Fatal("negative rate")
	}
}
func TestConcurrentBudget(t *testing.T) {
	l := NewLimiter(&fakeClock{}, 0, 99)
	var spent atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if l.AllowN(3) {
				spent.Add(3)
			}
		}()
	}
	close(start)
	wg.Wait()
	if spent.Load() != 99 {
		t.Fatal(spent.Load())
	}
}
