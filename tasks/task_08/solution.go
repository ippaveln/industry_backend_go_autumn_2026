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
	var lastTime time.Time
	if clock != nil {
		lastTime = clock.Now()
	}
	return &Limiter{
		clock: clock,
		rate: ratePerSec,
		burst: burst,
		tokens: float64(burst),
		last: lastTime,
	}
}
func (l *Limiter) AllowN(n int) bool {
	if l.clock == nil || l.burst <= 0 || n <= 0 || n > l.burst {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	
	now := l.clock.Now()

	if l.rate > 0 && now.After(l.last) {
		diff := now.Sub(l.last).Seconds()
		l.tokens += diff * l.rate
		if l.tokens > float64(l.burst) {
			l.tokens = float64(l.burst)
		}
		l.last = now
	}

	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		return true
	}

	return false
}
