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
	l := &Limiter{
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
	}

	if l.clock != nil {
		l.last = l.clock.Now()
	}

	return l
}
func (l *Limiter) refill() {
	if l.rate <= 0 {
		return
	}

	now := l.clock.Now()
	if elapsed := now.Sub(l.last); elapsed > 0 {
		l.tokens = math.Min(float64(l.burst), l.tokens+elapsed.Seconds()*l.rate)
		l.last = now
	}
}

func (l *Limiter) AllowN(n int) bool {
	if l.clock == nil || l.burst <= 0 || n <= 0 || n > l.burst {
		return false
	}

	l.refill()

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.tokens < float64(n) {
		return false
	}

	l.tokens -= float64(n)
	return true
}
