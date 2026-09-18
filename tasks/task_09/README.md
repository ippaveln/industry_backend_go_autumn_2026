# task_09: Worker pool: независимые результаты

![task 09](../../badges/tasks/task_09.svg)

```go
type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T,
	fn func(context.Context, T) (R, error)) ([]Result[R], error)
```

Применяет `fn` к каждому элементу `in` пулом из `workers` воркеров.

Аргументы, в порядке проверки:

- `workers <= 0` → `nil, ErrInvalidWorkers`.
- `ctx` уже отменён → `nil, ctx.Err()`; `fn` не вызывается, в том числе при пустом `in`.
- Пустой `in` → пустой срез, `nil`.
- nil `ctx` и nil `fn` не поддерживаются.

Обработка:

- `fn` вызывается ровно один раз для каждого элемента.
- Одновременно выполняется не больше `workers` вызовов `fn`. Пока есть невыданные элементы, заняты все `min(workers, len(in))` воркеров.
- Число горутин — O(min(workers, len(in))); горутина на каждый элемент не допускается.
- `results[i]` соответствует `in[i]`.
- Ошибка `fn` сохраняется в `results[i].Err` так, что `errors.Is` находит исходную ошибку; `Value` при этом — нулевое значение `R`.
- **Ошибка одного элемента не отменяет обработку остальных**, общая ошибка остаётся `nil`.
- Паники `fn` не обрабатываются.

Отмена `ctx` во время работы:

- новые элементы воркерам не выдаются;
- уже запущенные вызовы `fn` получают отменённый контекст; `fn` обязана завершиться после отмены;
- `ParallelMap` дожидается их и возвращает `nil, ctx.Err()`.

После возврата из `ParallelMap` все горутины пула завершены.

Пример: при `workers = 1` ошибка на `in[0]` не мешает успешно обработать `in[1]`.

Проверка: `make task_09`.
