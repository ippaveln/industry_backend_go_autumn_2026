# task_01: Приветствие с нормализацией

![task 01](../../badges/tasks/task_01.svg)

```go
func greet(name string) string
```

Возвращает `"Hello, <name>!"`.

- Пробельные символы Unicode по краям `name` отбрасываются по правилам `strings.TrimSpace`; пробелы внутри сохраняются.
- Если после обрезки имя пустое, вместо него подставляется `World`.

```go
greet("\t Анна  ") == "Hello, Анна!"
greet(" ")    == "Hello, World!"
```

Проверка: `make task_01`.
