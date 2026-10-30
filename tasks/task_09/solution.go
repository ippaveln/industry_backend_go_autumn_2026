package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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

	workerCount := workers
	if workerCount > len(in) {
		workerCount = len(in)
	}

	var next atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}

				i := int(next.Add(1) - 1)
				if i >= len(in) || ctx.Err() != nil {
					return
				}
				value, err := fn(ctx, in[i])
				if err != nil {
					results[i].Err = err
					continue
				}
				results[i].Value = value
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
