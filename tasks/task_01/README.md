# task_01: Приветствие с нормализацией

![task 01](../../badges/tasks/task_01.svg)

`func greet(name string) string` убирает крайние Unicode-пробелы по правилам `strings.TrimSpace`. Пустое после очистки имя заменяется на `World`. Внутренние пробелы сохраняются. Возвращает `Hello, <name>!`. Например, `greet("\t Анна  ")` → `"Hello, Анна!"`; `greet("\u2003")` → `"Hello, World!"`.

Меняйте только solution.go. Используйте stdlib. Проверка: `go test -race -count=4 ./tasks/task_01`.
