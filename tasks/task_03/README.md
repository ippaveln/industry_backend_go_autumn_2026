# task_03: FizzBuzz для знаковых чисел

![task 03](../../badges/tasks/task_03.svg)

`func fizzBuzz(n int) (string, error)` принимает положительные и отрицательные числа. Кратные 3 → `Fizz`, 5 → `Buzz`, обоим → `FizzBuzz`; остальные → десятичное представление со знаком. Только ноль недопустим: вернуть пустую строку и ошибку, распознаваемую `errors.Is(err, ErrZero)`. Объявите `var ErrZero = errors.New("zero is not allowed")`. Не берите модуль числа: поддержите минимальный int. Примеры: -15 → FizzBuzz, -7 → -7, 0 → ошибка.

Меняйте только solution.go. Используйте stdlib. Проверка: `go test -race -count=4 ./tasks/task_03`.
