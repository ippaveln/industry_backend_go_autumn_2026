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

func ParallelMap[T any, R any](ctx context.Context,
	workers int,
	in []T,
	fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	// 1. Проверяем количество воркеров.
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}

	// 2. Проверяем отмену контекста.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 3. Пустой вход.
	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	// Не создаём больше воркеров, чем элементов.
	if workers > len(in) {
		workers = len(in)
	}

	results := make([]Result[R], len(in))
	jobs := make(chan int)

	var wg sync.WaitGroup

	// 4. Запускаем пул воркеров.
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()

			for i := range jobs {
				// Если контекст отменён, новые задачи не выполняем.
				if ctx.Err() != nil {
					return
				}

				value, err := fn(ctx, in[i])

				if err != nil {
					var zero R
					results[i] = Result[R]{
						Value: zero,
						Err:   err,
					}
					continue
				}

				results[i] = Result[R]{
					Value: value,
					Err:   nil,
				}
			}
		}()
	}

	// 5. Распределяем задачи между воркерами.
dispatch:
	for i := range in {
		// Не выдаём новые задачи после отмены.
		if ctx.Err() != nil {
			break
		}

		select {
		case jobs <- i:
		case <-ctx.Done():
			break dispatch
		}
	}

	// 6. Закрываем канал и дожидаемся всех воркеров.
	close(jobs)
	wg.Wait()

	// 7. Если контекст отменён, возвращаем общую ошибку.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
