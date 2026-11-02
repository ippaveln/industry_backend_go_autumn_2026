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
	workerCount := min(workers, len(in))
	var group sync.WaitGroup
	var mutex sync.Mutex
	nextIndex := 0

	for i := 0; i < workerCount; i++ {
		group.Add(1)
		go func() {
			defer group.Done()

			for {
				mutex.Lock()
				if ctx.Err() != nil || nextIndex == len(in) {
					mutex.Unlock()
					return
				}

				index := nextIndex
				nextIndex++
				mutex.Unlock()

				value, err := fn(ctx, in[index])
				if err != nil {
					results[index].Err = err
				} else {
					results[index].Value = value
				}
			}
		}()
	}

	group.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
