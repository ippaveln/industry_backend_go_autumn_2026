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

type Job[T any] struct {
	Index int
	Value T
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
	jobs := make(chan Job[T])

	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()

		for job := range jobs {
			out, err := fn(ctx, job.Value)
			if err != nil {
				var zero R
				out = zero
			}
			result := Result[R]{Value: out, Err: err}
			results[job.Index] = result
		}
	}

	workers = min(workers, len(in))

	wg.Add(workers)
	for range workers {
		go worker()
	}

	cancelled := false

	for i, value := range in {
		select {
		case jobs <- Job[T]{
			Index: i,
			Value: value,
		}:

		case <-ctx.Done():
			cancelled = true
		}

		if cancelled {
			break
		}
	}
	close(jobs)
	wg.Wait()

	if cancelled {
		return nil, ctx.Err()
	}

	return results, nil
}
