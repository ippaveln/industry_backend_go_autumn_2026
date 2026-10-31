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

	// Каждый воркер пишет только в свою ячейку out[i], поэтому мьютекс не нужен
	// При пустом in это пустой срез, а не nil
	out := make([]Result[R], len(in))
	if len(in) == 0 {
		return out, nil
	}

	jobs := make(chan int)
	workers = min(workers, len(in))
	var wg sync.WaitGroup

	// Запуск воркеров
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				// Номер мог дойти до воркера уже после отмены
				if ctx.Err() != nil {
					continue
				}

				result, err := fn(ctx, in[i])
				if err != nil {
					out[i].Err = err
					continue
				}
				out[i].Value = result
			}
		})
	}

send:
	for i := range in {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break send
		}
	}

	close(jobs)
	// Ждём, пока завершатся все воркеры
	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, nil
}
