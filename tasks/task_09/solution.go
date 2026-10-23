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
	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	results := make([]Result[R], len(in))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range min(workers, len(in)) {
		wg.Go(func() {
			for i := range jobs {
				v, err := fn(ctx, in[i])
				if err != nil {
					results[i] = Result[R]{Err: err}
				} else {
					results[i] = Result[R]{Value: v}
				}
			}
		})
	}

feed:
	for i := range in {
		if ctx.Err() != nil {
			break
		}
		select {
		case jobs <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
