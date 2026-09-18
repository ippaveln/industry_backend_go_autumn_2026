# task_10: In-memory репозиторий и HTTP API задач

![task 10](../../badges/tasks/task_10.svg)

Сделайте небольшой сервис задач: потокобезопасный in-memory репозиторий и HTTP API поверх него. Решение проверяется тестами, поэтому строго следуйте контракту ниже.

## Типы и конструкторы

```go
type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}

type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

func NewInMemoryTaskRepo(clock Clock) TaskRepo
func NewHTTPHandler(repo TaskRepo) http.Handler
```

Clock не nil. `ErrNotFound` и `ErrInvalidTitle` — sentinel-ошибки, проверяются через `errors.Is`.

## Репозиторий

- Все методы потокобезопасны.
- Время берётся только из переданного Clock, `time.Now()` в репозитории не вызывается.
- ID уникален в рамках процесса и не пустой (счётчик, UUID — на ваше усмотрение).
- `Create` хранит title после `strings.TrimSpace`. Пустой после обрезки title → `ErrInvalidTitle`. `UpdatedAt` = `clock.Now()`, `Done` = false.
- `Get` несуществующего id → `Task{}, false`.
- `SetDone` несуществующего id → `ErrNotFound`. Если Done действительно меняется, обновляется `UpdatedAt`. Повторная установка того же Done возвращает прежнюю задачу без изменения `UpdatedAt`.
- `List(done)` возвращает только задачи с таким Done. Возвращённый срез не разделяет изменяемые данные с репозиторием.
- Ошибки репозитория не меняют состояние.

## HTTP API

| Метод и путь | Тело запроса | Успех |
|---|---|---|
| `POST /tasks` | `{"title":"buy milk"}` | 201, созданная задача |
| `GET /tasks/{id}` | — | 200, задача |
| `GET /tasks?done=false` | — | 200, массив незавершённых задач |
| `GET /tasks?done=true` | — | 200, массив завершённых задач |
| `PATCH /tasks/{id}` | `{"done":true}` | 200, обновлённая задача |

Пример ответа:

```json
{"id":"1","title":"buy milk","done":false,"updatedAt":"2026-01-24T12:00:00Z"}
```

### Список

- `GET /tasks` без параметра done равнозначен `done=false`. Допустимы только строки `true` и `false`; пустой, повторный или другой done → 400. Другие query-параметры игнорируются.
- Пустой список — JSON-массив `[]`, не `null`.
- Порядок: по `UpdatedAt` по убыванию (новые сначала), при равенстве — по ID по возрастанию; ID сравниваются как строки.

### Тело запроса

POST и PATCH принимают ровно один JSON-объект. Всё перечисленное → 400:

- невалидный JSON;
- неизвестные поля (используйте `json.Decoder` с `DisallowUnknownFields`);
- отсутствующее обязательное поле, `null` вместо значения, неверный тип;
- второй JSON-объект или мусор после объекта.

Пробелы после JSON допустимы. Повторные известные поля обрабатываются стандартным `encoding/json` (берётся последнее значение).

### Статусы и заголовки

- Успешные ответы имеют `Content-Type: application/json` (параметры вроде `; charset=utf-8` допустимы).
- Неизвестный путь, пустой id или вложенный путь (`/tasks/x/y`) → 404.
- Неподдерживаемый метод на известном пути (например, `DELETE /tasks/{id}` или `PUT /tasks`) → 405.
- Handler работает только через переданный `TaskRepo`. Ошибки репозитория:
  - `ErrInvalidTitle` из `Create` → 400;
  - `ErrNotFound` из `SetDone` → 404;
  - любая другая ошибка `Create` или `SetDone` → 500.
- Формат тела ошибки не фиксируется.

## Пример

Создать задачу, выполнить `PATCH {"done":true}`, повторить PATCH позднее — `UpdatedAt` не меняется. После этого `GET /tasks` возвращает `[]`, а `GET /tasks?done=true` — массив с этой задачей.

Меняйте только solution.go. Используйте stdlib. Проверка: `go test -race -count=4 ./tasks/task_10`.
