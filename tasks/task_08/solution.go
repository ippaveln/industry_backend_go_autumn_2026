package main

import (
	"math"
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
	limiter := &Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
	}
	if clock != nil {
		limiter.last = clock.Now()
	}
	return limiter
}
func (l *Limiter) AllowN(n int) bool {
	if n <= 0 || n > l.burst || l.burst <= 0 || l.clock == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rate > 0 {
		now := l.clock.Now()
		diff := now.Sub(l.last).Seconds()
		if diff > 0 {
			l.tokens = math.Min(float64(l.burst), diff*l.rate+l.tokens)
			l.last = now
		}
	}
	if float64(n) > l.tokens {
		return false
	}
	l.tokens -= float64(n)
	return true
}
