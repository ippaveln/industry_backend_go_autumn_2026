package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

type job[T any] struct {
	idx int
	val T
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	err := validateParams(ctx, workers, in, fn)
	if err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if len(in) == 0 || ctx == nil || in == nil {
		return []Result[R]{}, nil
	}

	jobs := make(chan job[T], len(in))
	go func() {
		for idx, t := range in {
			jobs <- job[T]{idx: idx, val: t}
		}
		close(jobs)
	}()

	results := make([]Result[R], len(in))
	var wg sync.WaitGroup
	for range min(workers, len(in)) {
		wg.Go(func() {
			for j := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				v, err := fn(ctx, j.val)
				var r Result[R]
				if err != nil {
					r.Err = err
				} else {
					r.Value = v
				}
				results[j.idx] = r
			}
		})
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func validateParams[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) error {
	if workers <= 0 {
		return ErrInvalidWorkers
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}
