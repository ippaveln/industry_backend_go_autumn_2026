package main

import (
	"sync"
	"time"
)

type (
	Clock   interface{ Now() time.Time }
	Limiter struct {
		mu     sync.Mutex
		clock  Clock
		rate   float64
		burst  int
		tokens float64
		last   time.Time
	}
)

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	l := &Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
	}
	if clock != nil {
		l.last = clock.Now()
	}
	return l
}

func (l *Limiter) AllowN(n int) bool {
	if l.burst <= 0 || l.clock == nil || n <= 0 || n > l.burst {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if now := l.clock.Now(); l.rate > 0 && now.After(l.last) {
		seconds := now.Sub(l.last).Seconds()
		l.tokens = min(float64(l.burst), l.rate*seconds+l.tokens)
		l.last = now
	}

	if float64(n) > l.tokens {
		return false
	}

	l.tokens -= float64(n)
	return true
}
