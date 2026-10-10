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
	if len(in) == 0 {
		return results, nil
	}

	indexes := make(chan int)

	var wg sync.WaitGroup
	for range min(workers, len(in)) {
		wg.Go(func() {
			for i := range indexes {
				if ctx.Err() != nil {
					continue
				}

				v, err := fn(ctx, in[i])
				if err != nil {
					results[i] = Result[R]{Err: err}
				} else {
					results[i] = Result[R]{Value: v}
				}
			}
		})
	}

	// TODO: remove the label.
dispatch:
	for i := range in {
		select {
		case indexes <- i:
		case <-ctx.Done():
			break dispatch
		}
	}
	close(indexes)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
