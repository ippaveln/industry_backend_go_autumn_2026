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

	numWorkers := min(workers, len(in))
	results := make([]Result[R], len(in))
	jobs := make(chan int)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for range numWorkers {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				value, err := fn(ctx, in[idx])
				if err != nil {
					results[idx] = Result[R]{Err: err}
					continue
				}
				results[idx] = Result[R]{Value: value}
			}
		}()
	}

dispatch:
	for i := range in {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break dispatch
		}
	}
	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
