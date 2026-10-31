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

func ParallelMap[T any, R any](
	ctx context.Context,
	workers int,
	in []T,
	fn func(context.Context, T) (R, error),
) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	n := workers
	if n > len(in) {
		n = len(in)
	}

	results := make([]Result[R], len(in))
	jobs := make(chan int)

	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			for i := range jobs {
				v, err := fn(ctx, in[i])
				if err != nil {
					results[i] = Result[R]{Err: err}
					continue
				}
				results[i] = Result[R]{Value: v}
			}
		}()
	}

	for i := range in {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}