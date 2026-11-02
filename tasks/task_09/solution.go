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
	results := make([]Result[R], len(in))
	var wg sync.WaitGroup
	ch := make(chan int)
	for range min(workers, len(in)) {
		wg.Go(func() {
			for i := range ch {
				if ctx.Err() != nil {
					return
				}
				result, err := fn(ctx, in[i])
				if err != nil {
					results[i] = Result[R]{Err: err}
				} else {
					results[i] = Result[R]{Value: result}
				}
			}
		})
	}

loop:
	for i := range in {
		select {
		case ch <- i:
		case <-ctx.Done():
			break loop
		}
	}
	close(ch)
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return results, nil
}
