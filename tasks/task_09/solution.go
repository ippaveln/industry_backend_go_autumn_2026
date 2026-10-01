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

	result := make([]Result[R], len(in))
	jobs := make(chan int)

	var wg sync.WaitGroup // атомарный счетчик для горутин

	for range min(workers, len(in)) {

		wg.Go(func() { // делает wg.Add(1) и defer wg.Done()
			for i := range jobs {
				v, err := fn(ctx, in[i])

				if err != nil {
					result[i].Err = err
					continue
				}

				result[i].Value = v

			}
		})
	}

waiter:
	for i := range in {
		select {
		case jobs <- i: // ждет пока кто-то возьмет i
		case <-ctx.Done():
			break waiter
		}
	}

	close(jobs)
	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return result, nil
}
