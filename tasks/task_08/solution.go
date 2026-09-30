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
	limiter := &Limiter{clock: clock, rate: ratePerSec, burst: burst}
	if clock != nil && burst > 0 {
		limiter.tokens = float64(burst)
		limiter.last = clock.Now()
	}
	return limiter
}
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.clock == nil || l.burst <= 0 {
		return false
	}
	if n <= 0 || n > l.burst {
		return false
	}
	moment := l.clock.Now()
	duration := moment.Sub(l.last)
	if duration > 0 {
		rate := max(l.rate, 0)
		predict := l.tokens + (rate * duration.Seconds())
		l.tokens = min(predict, float64(l.burst))
		l.last = moment
	}
	if l.tokens < float64(n) {
		return false
	}
	l.tokens -= float64(n)
	return true
}
