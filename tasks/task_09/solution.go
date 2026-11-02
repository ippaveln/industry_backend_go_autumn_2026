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
	next := 0

	var wg sync.WaitGroup
	var mu sync.Mutex

	worker := func() {
		for {
			mu.Lock()

			if ctx.Err() != nil || next == len(in) {
				mu.Unlock()
				return
			}

			i := next
			next++
			mu.Unlock()

			if value, err := fn(ctx, in[i]); err != nil {
				results[i].Err = err
			} else {
				results[i].Value = value
			}
		}
	}

	for range min(workers, len(in)) {
		wg.Go(worker)
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
