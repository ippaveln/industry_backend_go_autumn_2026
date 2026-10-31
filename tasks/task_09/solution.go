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
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(in) == 0 {
		return []Result[R]{}, nil
	}
	nums := make(chan int, len(in))
	for i := range len(in) {
		nums <- i
	}
	close(nums)
	results := make([]Result[R], len(in))

	var wg sync.WaitGroup
	for range min(workers, len(in)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range nums {
				if ctx.Err() != nil {
					return
				}
				v, err := fn(ctx, in[i])
				if err != nil {
					results[i].Err = err
				} else {
					results[i].Value = v
				}
			}
		}()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return results, nil
}
