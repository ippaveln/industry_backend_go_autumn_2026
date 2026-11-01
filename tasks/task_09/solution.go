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

	var wg sync.WaitGroup
	result := make([]Result[R], len(in))
	tasks := make(chan int)

	for range min(workers, len(in)) {
		wg.Go(func() {
			for i := range tasks {
				v, err := fn(ctx, in[i])
				if err != nil {
					result[i].Err = err
				} else {
					result[i].Value = v
				}
			}
		})
	}

	dispatch:
		for next := 0; next < len(in); next++ {
			select {
			case <-ctx.Done():
				break dispatch
			case tasks <- next:
			}
		}
	close(tasks)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
