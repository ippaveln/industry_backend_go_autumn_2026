package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	result := make([]Result[R], len(in))
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(in) == 0 {
		return []Result[R]{}, nil
	}
	next := 0
	var mutex sync.Mutex
	var wait sync.WaitGroup
	work := min(len(in), workers)
	worker := func() {
		for {
			if ctx.Err() != nil {
				return
			}
			mutex.Lock()
			i := next
			next++
			mutex.Unlock()
			if i >= len(in) {
				return
			}
			val, err := fn(ctx, in[i])
			if err == nil {
				result[i] = Result[R]{val, err}
			} else {
				result[i] = Result[R]{Err: err}
			}
		}
	}
	for i := 0; i < work; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			worker()
		}()
	}
	wait.Wait()
	if ctx.Err() == nil {
		return result, nil
	} else {
		return nil, ctx.Err()
	}
}
