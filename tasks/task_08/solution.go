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
	if ratePerSec < 0 {
		ratePerSec = 0
	}

	return &Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
	}
}
func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clock == nil || l.burst <= 0 || n <= 0 || n > l.burst {
		return false
	}

	now := l.clock.Now()

	if now.After(l.last) {
		sec := now.Sub(l.last).Seconds()
		l.last = now

		l.tokens += l.rate * sec
		if l.tokens > float64(l.burst) {
			l.tokens = float64(l.burst)
		}
	}

	if float64(n) > l.tokens {
		return false
	}

	l.tokens -= float64(n)

	return true
}
