package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }

type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64   // скорость наливки
	burst  int       // максимальное количество токенов
	tokens float64   // текущее количество токенов
	last   time.Time // время последней наливки
}

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
	if l.clock == nil || l.burst <= 0 || n <= 0 || n > l.burst {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()
	if l.tokens < float64(n) {
		return false
	}
	l.tokens -= float64(n)
	return true
}

func (l *Limiter) refill() {
	now := l.clock.Now()
	if !now.After(l.last) {
		return
	}
	if l.rate > 0 {
		earned := now.Sub(l.last).Seconds() * l.rate
		l.tokens = min(float64(l.burst), l.tokens+earned)
	}
	l.last = now
}
