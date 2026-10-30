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

	jobs := make(chan int)
	results := make([]Result[R], len(in))

	var wg sync.WaitGroup

	if workers > len(in) {
		workers = len(in)
	}

	for range workers {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case i, ok := <-jobs:
					if !ok {
						return
					}

					value, err := fn(ctx, in[i])

					if err != nil {
						results[i].Err = err

						continue
					}

					results[i].Value = value
				}
			}

		})
	}

	for i := range in {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
			continue
		}

		break
	}

	close(jobs)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
