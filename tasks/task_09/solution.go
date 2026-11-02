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
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	results := make([]Result[R], len(in))
	var pool sync.WaitGroup
	var dispatch sync.Mutex
	next := 0
	for range min(workers, len(in)) {
		pool.Go(func() {
			for {
				dispatch.Lock()
				if next == len(in) || ctx.Err() != nil {
					dispatch.Unlock()
					return
				}
				i := next
				next++
				dispatch.Unlock()

				value, err := fn(ctx, in[i])
				if err == nil {
					results[i].Value = value
				}
				results[i].Err = err
			}
		})
	}
	pool.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
