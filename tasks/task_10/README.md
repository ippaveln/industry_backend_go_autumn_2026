# task_10: In-memory репозиторий и HTTP API задач

![task 10](../../badges/tasks/task_10.svg)

Потокобезопасный in-memory репозиторий задач и HTTP API к нему.

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

`clock` не бывает nil. `ErrNotFound` и `ErrInvalidTitle` проверяются через `errors.Is`.

## Репозиторий

- Все методы потокобезопасны.
- Время берётся только из `clock`; `time.Now()` в репозитории не вызывается.
- ID непустой и уникален в пределах процесса; способ генерации — любой (счётчик, UUID).
- `Create` сохраняет title после `strings.TrimSpace`; если он пустой — `ErrInvalidTitle`. У новой задачи `UpdatedAt = clock.Now()`, `Done = false`.
- `Get` несуществующего id → `Task{}, false`.
- `SetDone` несуществующего id → `ErrNotFound`. `UpdatedAt` обновляется, только если `Done` действительно изменился; установка того же значения возвращает задачу без изменений.
- `List(done)` возвращает задачи с заданным `Done`. Изменение возвращённого среза не влияет на репозиторий.
- Метод, вернувший ошибку, не меняет состояние.

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

- `GET /tasks` без `done` равнозначен `done=false`. Допустимы только значения `true` и `false`; пустой, повторённый или иной `done` → 400. Остальные query-параметры игнорируются.
- Пустой список кодируется как `[]`, не `null`.
- Сортировка: по `UpdatedAt` по убыванию, при равенстве — по ID по возрастанию (ID сравниваются как строки).

### Тело запроса

Тело POST и PATCH — ровно один JSON-объект. 400 возвращается, если:

- JSON невалиден;
- есть неизвестные поля (`json.Decoder.DisallowUnknownFields`);
- обязательное поле отсутствует, равно `null` или имеет неверный тип;
- после объекта идёт второй объект или другие данные.

Пробельные символы после объекта допустимы. Повторённое известное поле обрабатывается как в `encoding/json`: берётся последнее значение.

### Статусы и заголовки

- Успешные ответы имеют `Content-Type: application/json` (параметры вроде `; charset=utf-8` допустимы).
- Неизвестный путь, пустой id или вложенный путь (`/tasks/x/y`) → 404.
- Неподдерживаемый метод на известном пути (например, `DELETE /tasks/{id}` или `PUT /tasks`) → 405.
- Handler обращается к данным только через переданный `TaskRepo`. Ошибки репозитория:
  - `ErrInvalidTitle` из `Create` → 400;
  - `ErrNotFound` из `SetDone` → 404;
  - любая другая ошибка `Create` или `SetDone` → 500.
- Формат тела ошибки не фиксируется.

## Пример

1. `POST /tasks` создаёт задачу.
2. `PATCH /tasks/{id}` с `{"done":true}` обновляет `Done` и `UpdatedAt`.
3. Повторный такой же PATCH позже не меняет `UpdatedAt`.
4. `GET /tasks` возвращает `[]`, `GET /tasks?done=true` — массив с этой задачей.

Проверка: `make task_10`.
