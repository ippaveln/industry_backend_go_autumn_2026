package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	if len(in) == 0 {
		return results, nil
	}
	pool := min(workers, len(in))
	var next atomic.Int64
	var wg sync.WaitGroup
	wg.Add(pool)
	for w := 0; w < pool; w++ {
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				i := int(next.Add(1)) - 1
				if i >= len(in) {
					return
				}
				v, err := fn(ctx, in[i])
				if err != nil {
					var zero R
					results[i] = Result[R]{Value: zero, Err: err}
					continue
				}
				results[i] = Result[R]{Value: v}
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
