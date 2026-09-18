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

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// step is one call in a sequential limiter scenario: the clock is advanced
// first, then AllowN(n) is expected to return want.
type step struct {
	name    string
	advance time.Duration
	n       int
	want    bool
}

func runSteps(t *testing.T, clock *fakeClock, l *Limiter, steps []step) {
	t.Helper()
	for _, s := range steps {
		clock.add(s.advance)
		got := l.AllowN(s.n)
		if got != s.want {
			t.Fatalf("%s: AllowN(%d) = %t, want %t", s.name, s.n, got, s.want)
		}
	}
}

func TestAtomicSpend(t *testing.T) {
	clock := &fakeClock{}
	l := NewLimiter(clock, 0, 3)

	runSteps(t, clock, l, []step{
		{name: "zero tokens", n: 0, want: false},
		{name: "negative tokens", n: -1, want: false},
		{name: "more than burst", n: 4, want: false},
		{name: "spend two of three", n: 2, want: true},
		{name: "not enough left", n: 2, want: false},
		{name: "failed spend kept the token", n: 1, want: true},
		{name: "bucket empty", n: 1, want: false},
	})
}

func TestRefillAndRollback(t *testing.T) {
	clock := &fakeClock{}
	l := NewLimiter(clock, 2.5, 3)

	runSteps(t, clock, l, []step{
		{name: "starts full", n: 3, want: true},
		{name: "fraction is not a token", advance: 200 * time.Millisecond, n: 1, want: false},
		{name: "fractions accumulate", advance: 200 * time.Millisecond, n: 1, want: true},
		{name: "clock rollback does not refill", advance: -time.Second, n: 1, want: false},
		{name: "no double refill after rollback", advance: time.Second, n: 1, want: false},
		{name: "refill is capped by burst", advance: 10 * time.Second, n: 3, want: true},
		{name: "capped bucket is empty", n: 1, want: false},
	})
}

func TestDisabled(t *testing.T) {
	for _, tc := range []struct {
		name  string
		clock Clock
		burst int
	}{
		{name: "negative burst", clock: &fakeClock{}, burst: -1},
		{name: "zero burst", clock: &fakeClock{}, burst: 0},
		{name: "nil clock", clock: nil, burst: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := NewLimiter(tc.clock, 1, tc.burst)

			got := l.AllowN(1)

			if got {
				t.Fatal("disabled limiter must reject AllowN(1)")
			}
		})
	}
}

func TestNegativeRateDoesNotRefill(t *testing.T) {
	clock := &fakeClock{}
	l := NewLimiter(clock, -1, 1)

	runSteps(t, clock, l, []step{
		{name: "time does not burn tokens", advance: time.Second, n: 1, want: true},
		{name: "no refill", advance: time.Second, n: 1, want: false},
	})
}

func TestConcurrentBudget(t *testing.T) {
	const (
		burst   = 99
		callers = 100
		cost    = 3
	)
	l := NewLimiter(&fakeClock{}, 0, burst)
	var spent atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range callers {
		wg.Go(func() {
			<-start
			if l.AllowN(cost) {
				spent.Add(cost)
			}
		})
	}

	close(start)
	wg.Wait()

	if got := spent.Load(); got != burst {
		t.Fatalf("spent %d tokens, want %d", got, burst)
	}
}
