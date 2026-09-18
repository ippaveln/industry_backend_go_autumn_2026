# task_05: Generic-кэш с отказом при заполнении

![task 05](../../badges/tasks/task_05.svg)

```go
func NewCache[K comparable, V any](capacity int) *Cache[K, V]
func (c *Cache[K, V]) Get(key K) (V, bool)
func (c *Cache[K, V]) Set(key K, value V) bool
```

Кэш хранит не больше `capacity` ключей и ничего не вытесняет.

- `Set` нового ключа в заполненном кэше возвращает `false` и не меняет состояние.
- `Set` существующего ключа обновляет значение и возвращает `true`, даже если кэш заполнен.
- `capacity <= 0` выключает кэш: `Set` всегда возвращает `false`, `Get` — промах.
- Промах `Get` → нулевое значение `V` и `false`. Сохранённое нулевое значение → нулевое значение `V` и `true`.
- Кэш создаётся только через `NewCache`; nil-получатель поддерживать не нужно.
- Потокобезопасность не нужна.

```go
c := NewCache[string, int](1)
c.Set("a", 1) // true
c.Set("b", 2) // false: кэш заполнен
c.Set("a", 3) // true: обновление существующего ключа
```

Проверка: `make task_05`.
