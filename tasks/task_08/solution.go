package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	limit := Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
	}
	if clock != nil {
		limit.last = clock.Now()
	}
	return &limit
}

func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n <= 0 || n > l.burst || l.burst <= 0 || l.clock == nil {
		return false
	}
	c := l.clock.Now()
	t := c.Sub(l.last)
	if t > 0 && l.rate > 0 {
		r := float64(t.Seconds()) * l.rate
		l.last = c
		if l.tokens+r <= float64(l.burst) {
			l.tokens += r
		} else {
			l.tokens = float64(l.burst)
		}
	}
	if l.tokens-float64(n) >= 0 {
		l.tokens -= float64(n)
		return true
	} else {
		return false
	}
}
