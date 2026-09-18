# task_08: Token bucket: атомарное списание

![task 08](../../badges/tasks/task_08.svg)

```go
type Clock interface{ Now() time.Time }

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter
func (l *Limiter) AllowN(n int) bool
```

Ограничитель частоты по алгоритму token bucket.

- В корзине не больше `burst` токенов; при создании она полная.
- За секунду добавляется `ratePerSec` токенов, дробная часть учитывается.
- `AllowN(n)` атомарно списывает ровно `n` токенов и возвращает `true`. Если токенов не хватает, возвращает `false` и **ничего не списывает**.
- `n <= 0` или `n > burst` → `false`.
- `burst <= 0` или `clock == nil` выключает лимитер: `AllowN` всегда возвращает `false`.
- `ratePerSec <= 0` — запас не пополняется. `ratePerSec` всегда конечен.
- Время берётся только из `clock`. `time.Now`, `time.Sleep` и любые ожидания внутри лимитера запрещены.
- Если часы перевели назад, запас не пополняется, а запомненное время последнего пополнения не уменьшается.
- `AllowN` потокобезопасен.

```go
l := NewLimiter(clock, 0, 3) // burst = 3, без пополнения
l.AllowN(4) // false: больше burst
l.AllowN(2) // true, осталось 1
l.AllowN(2) // false, осталось 1
l.AllowN(1) // true, осталось 0
```

Проверка: `make task_08`.
