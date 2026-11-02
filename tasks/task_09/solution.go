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

	n := len(in)
	results := make([]Result[R], n)

	numWorkers := min(workers, n)

	ch := make(chan int, n)

	for i := range n {
		ch <- i
	}

	close(ch)

	var wg sync.WaitGroup

	for range numWorkers {
		wg.Go(func() {
			for i := range ch {
				if ctx.Err() != nil {
					return
				}

				val, err := fn(ctx, in[i])

				if err != nil {
					var zero R
					results[i] = Result[R]{Value: zero, Err: err}
				} else {
					results[i] = Result[R]{Value: val, Err: nil}
				}
			}
		})
	}

	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return results, nil
}
