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
	l := &Limiter{clock: clock, rate: max(ratePerSec, 0), burst: burst}
	if clock != nil && burst > 0 {
		l.last = clock.Now()
		l.tokens = float64(burst)
	}
	return l
}
func (l *Limiter) AllowN(n int) bool {
	if l.clock == nil || n <= 0 || n > l.burst {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	if now.After(l.last) {
		elapsed := now.Sub(l.last).Seconds()
		l.tokens = min(float64(l.burst), l.tokens+elapsed*l.rate)
		l.last = now
	}
	cost := float64(n)
	if cost > l.tokens {
		return false
	}
	l.tokens -= cost
	return true
}
