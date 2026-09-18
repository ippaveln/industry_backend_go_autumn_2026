# task_03: FizzBuzz для знаковых чисел

![task 03](../../badges/tasks/task_03.svg)

```go
var ErrZero = errors.New("zero is not allowed")

func fizzBuzz(n int) (string, error)
```

- `n` кратно 3 и 5 → `"FizzBuzz"`; только 3 → `"Fizz"`; только 5 → `"Buzz"`.
- Иначе — десятичная запись `n` со знаком.
- `n == 0` → `""` и ошибка, для которой `errors.Is(err, ErrZero)` истинно.
- Отрицательные числа обрабатываются так же, включая `math.MinInt`. Не берите модуль числа: для `math.MinInt` он переполняется.

```go
fizzBuzz(-15) // "FizzBuzz", nil
fizzBuzz(-7)  // "-7", nil
fizzBuzz(0)   // "", ErrZero
```

Проверка: `make task_03`.
