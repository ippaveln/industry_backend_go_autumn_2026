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
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range min(workers, len(in)) {
		wg.Go(func() {
			for i := range jobs {
				if ctx.Err() != nil {
					return
				}
				value, err := fn(ctx, in[i])
				// Каждый индекс выдаётся один раз: воркеры пишут в разные элементы среза.
				results[i].Err = err
				if err == nil {
					results[i].Value = value
				}
			}
		})
	}

	// Метка позволяет при отмене выйти из всего цикла отправки заданий.
dispatch:
	for i := range in {
		select {
		case <-ctx.Done():
			break dispatch
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
